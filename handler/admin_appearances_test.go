package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golog/entity"
	"golog/system"
	"golog/view"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

// restoreSystemState 快照并还原 system 包的全局状态。AppearancesEdit 会调用
// system.SaveConfig()，它会重新加载主题模板；若不还原，会让后续依赖
// NotFoundTmpl == nil 的用例（如 pow 中间件测试）走向数据库分支而失败。
func restoreSystemState(t *testing.T) {
	t.Helper()
	oldConfig := system.Config
	oldLocale := system.Locale
	old := [9]*template.Template{
		system.IndexTmpl, system.PostTmpl, system.SingularTmpl, system.AuthorTmpl,
		system.MomentTmpl, system.WhisperTmpl, system.AboutTmpl, system.NotFoundTmpl, system.PowTmpl,
	}
	t.Cleanup(func() {
		system.Config = oldConfig
		system.Locale = oldLocale
		system.IndexTmpl, system.PostTmpl, system.SingularTmpl, system.AuthorTmpl =
			old[0], old[1], old[2], old[3]
		system.MomentTmpl, system.WhisperTmpl, system.AboutTmpl, system.NotFoundTmpl, system.PowTmpl =
			old[4], old[5], old[6], old[7], old[8]
	})
}

// TestAppearancesEditPersistsInjectedCode 回归测试：外观主表单提交时，
// “自定义代码”（网页开头/结尾、文章开头/结尾）必须与外观字段一起落库。
//
// 历史缺陷：注入代码的四个字段属于独立的 <form action="appearances/injected">，
// 用户在外观表单点击“保存更改”时它们不会被提交，页面却提示更新成功，
// 造成“修改自定义代码没有写入”的现象。
func TestAppearancesEditPersistsInjectedCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restoreSystemState(t)

	system.Config = &entity.Config{
		Theme:        "default",
		Locale:       "zh-cn",
		DateFormat:   "2006-01-02",
		TimeFormat:   "15:04",
		PostsPerPage: 10,
	}
	system.ReloadLocale("zh-cn")

	var saved *entity.Config
	system.SetConfigWriter(func(c *entity.Config) error {
		cp := *c
		saved = &cp
		return nil
	})

	r := gin.New()
	r.Use(sessions.Sessions("golog", cookie.NewStore([]byte("test-secret"))))
	r.POST("/admin/appearances", handleForm(AppearancesEdit))

	form := url.Values{}
	form.Set("footer_text", "<p>footer</p>")
	form.Set("color_scheme", "dark")
	form.Set("container_width", "large")
	form.Set("highlight_js", "true")
	form.Set("author_block", "start")
	form.Set("posts_per_page", "20")
	form.Set("theme", "default")
	form.Set("favicon", "https://example.com/fav.ico")
	form.Set("custom_css", ":root { --x: 1; }")
	form.Set("injected_head", `<script id="head">H</script>`)
	form.Set("injected_foot", `<script id="foot">F</script>`)
	form.Set("injected_post_start", `<div id="start">S</div>`)
	form.Set("injected_post_end", `<div id="end">E</div>`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/appearances", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("unexpected status %d, body=%s", w.Code, w.Body.String())
	}
	if saved == nil {
		t.Fatal("configWriter was not called: appearance settings were not persisted")
	}
	if saved.CustomCSS != ":root { --x: 1; }" {
		t.Errorf("custom_css not saved: %q", saved.CustomCSS)
	}
	if saved.InjectedHead != `<script id="head">H</script>` {
		t.Errorf("injected_head not saved: %q", saved.InjectedHead)
	}
	if saved.InjectedFoot != `<script id="foot">F</script>` {
		t.Errorf("injected_foot not saved: %q", saved.InjectedFoot)
	}
	if saved.InjectedPostStart != `<div id="start">S</div>` {
		t.Errorf("injected_post_start not saved: %q", saved.InjectedPostStart)
	}
	if saved.InjectedPostEnd != `<div id="end">E</div>` {
		t.Errorf("injected_post_end not saved: %q", saved.InjectedPostEnd)
	}
}

// TestAppearancesTemplateInjectedCodeInMainForm 结构性回归测试：后台外观页面
// 只能有一个内容表单，“自定义代码”的四个 textarea 必须属于它。若再次把它们
// 拆到独立的 <form> 里，主表单的“保存更改”就会静默丢弃这些修改。
func TestAppearancesTemplateInjectedCodeInMainForm(t *testing.T) {
	b, err := view.Templates.ReadFile("templates/admin_appearances.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	doc, err := html.Parse(strings.NewReader(string(b)))
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	injected := map[string]bool{
		"injected_head":       false,
		"injected_foot":       false,
		"injected_post_start": false,
		"injected_post_end":   false,
	}

	var (
		formDepth int
		formCount int
	)
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			formCount++
			formDepth++
			for _, a := range n.Attr {
				if a.Key == "action" && strings.Contains(a.Val, "injected") {
					t.Errorf("injected code must not live in its own form (action=%q)", a.Val)
				}
			}
		}
		if n.Type == html.ElementNode && (n.Data == "textarea" || n.Data == "input") {
			for _, a := range n.Attr {
				if a.Key != "name" {
					continue
				}
				if _, ok := injected[a.Val]; ok {
					if formDepth == 0 {
						t.Errorf("control %q is not inside any form", a.Val)
					}
					injected[a.Val] = true
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
		if n.Type == html.ElementNode && n.Data == "form" {
			formDepth--
		}
	}
	walk(doc)

	if formCount != 1 {
		t.Errorf("expected exactly 1 form on the appearances page, got %d", formCount)
	}
	for name, found := range injected {
		if !found {
			t.Errorf("control %q is missing from the appearances main form", name)
		}
	}
}
