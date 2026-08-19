package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"golog/entity"
	"golog/store"
	"golog/system"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// TestNavigationEditConformRegression 回归测试：POST /admin/navigations/edit
// 曾在 handleForm 的 conform.Strings 阶段 panic：
//
//	reflect.Value.Convert: value of type string cannot be converted to type int
//
// 根因：NavigationEditRequest.Sequences 原先为 []int，而 Go 中 int 可转换为 string
// （string(65) == "A"），conform 库因此把 []int 当字符串切片处理，对每个元素执行
// string→int 转换时 panic。现 Sequences 已改为 []string，此处验证不再 panic。
func TestNavigationEditConformRegression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var got *NavigationEditRequest
	called := false
	h := handleForm(func(c *gin.Context, req *NavigationEditRequest) {
		called = true
		got = req
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	form := url.Values{}
	form.Add("name[]", "Home")
	form.Add("url[]", "https://example.com")
	form.Add("sequence[]", "1")
	form.Add("is_deleted[]", "false")
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/navigations/edit", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	h(c) // 修复前此处 panic

	if !called {
		t.Fatal("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", w.Code)
	}
	if len(got.Names) != 1 || got.Names[0] != "Home" {
		t.Fatalf("unexpected names: %#v", got.Names)
	}
	if len(got.Sequences) != 1 || got.Sequences[0] != "1" {
		t.Fatalf("unexpected sequences: %#v", got.Sequences)
	}
}

// TestNavigationEditHandler 端到端验证 /admin/navigations/edit：
// 提交重新排序 + 删除行的表单后，数据库中的导航按新顺序落库。
func TestNavigationEditHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// NavigationEdit 会调用 setMessage → system.Locale.String(...)，
	// 测试里初始化 locale（生产环境由 SaveConfig 完成）。
	system.ReloadLocale("zh-CN")

	// 使用独立的临时数据库，避免污染本地 golog.sqlite
	if err := store.Open(filepath.Join(t.TempDir(), "test.sqlite")); err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	if err := store.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	seeds := []struct {
		name, url string
	}{
		{"Home", "https://example.com"},
		{"Blog", "https://example.com/blog"},
	}
	for i, s := range seeds {
		if err := store.CreateNavigation(&entity.NavigationW{
			ID:       "seed-" + string(rune('1'+i)),
			Name:     s.name,
			URL:      s.url,
			Sequence: i + 1,
		}); err != nil {
			t.Fatalf("CreateNavigation: %v", err)
		}
	}

	router := gin.New()
	router.Use(sessions.Sessions("golog_session", cookie.NewStore([]byte("test-secret"))))
	router.POST("/admin/navigations/edit", handleForm(NavigationEdit))

	// 两行：Blog(seq=2, 保留)、Home(seq=1, 勾选删除)
	form := url.Values{}
	form.Add("name[]", "Blog")
	form.Add("name[]", "Home")
	form.Add("url[]", "https://example.com/blog")
	form.Add("url[]", "https://example.com")
	form.Add("sequence[]", "2")
	form.Add("sequence[]", "1")
	form.Add("is_deleted[]", "false")
	form.Add("is_deleted[]", "true")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/navigations/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("unexpected status %d, body=%s", w.Code, w.Body.String())
	}

	navs, err := store.ListNavigations()
	if err != nil {
		t.Fatalf("ListNavigations: %v", err)
	}
	// Home 被删除，只剩 Blog，且重排后 Sequence 应为 1
	if len(navs) != 1 {
		t.Fatalf("expected 1 navigation, got %d: %#v", len(navs), navs)
	}
	if navs[0].Name != "Blog" || navs[0].Sequence != 1 {
		t.Fatalf("unexpected navigation: %#v", navs[0])
	}
}
