package system

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"golog/entity"
)

// corporatePost 构造一篇用于渲染测试的文章。
func corporatePost(id, title, content string) *entity.PostR {
	return &entity.PostR{
		Type:        "blog",
		ID:          id,
		Title:       title,
		Slug:        id,
		Content:     content,
		Visibility:  entity.VisibilityPublic,
		PublishedAt: 1750000000,
	}
}

func setupCorporateTheme(t *testing.T) {
	t.Helper()

	original := Config
	Config = &entity.Config{
		Name:        "极海 | GeoHey Blog",
		Description: "探索地理位置的价值",
		Theme:       "corporate",
		Locale:      "zh-cn",
		DateFormat:  "2006-01-02",
	}
	SetConfigWriter(func(*entity.Config) error { return nil })
	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	t.Cleanup(func() { Config = original })
}

// indexData 构造与 handler.data() 一致的首页模板数据，保证模板不会因缺字段而失败。
func indexData(posts []*entity.PostR, navigations []*entity.NavigationR, total int) map[string]any {
	// data() 传入的是 *map[[2]string]int（与 store 的分组查询返回类型一致），
	// 这里保持同样的指针形态，避免测试与线上数据形态不一致。
	tagMap := map[[2]string]int{{"产品", "product"}: 1}
	stats := map[[2]string]int{{"2025", "09"}: 1}
	momentStats := map[[2]string]int{{"2025", ""}: 1}

	return map[string]any{
		"Config": Config,
		"URL": map[string]string{
			"Root":         "http://localhost/",
			"Absolute":     "http://localhost/",
			"AbsoluteHost": "http://localhost/",
			"RelativeRoot": "/",
			"PageType":     "index",
		},
		"Posts":       posts,
		"Navigations": navigations,
		"Routes":      []entity.Route{{Name: "首页", Path: "/"}},
		"Stats":       &stats,
		"MomentStats": &momentStats,
		"TagMap":      &tagMap,
		"Self":        nil,
		"Message":     "",
		"Search":      "",
		"Pagination":  &entity.Pagination{CurrentPage: 1, TotalCount: total, TotalPages: 1},
	}
}

func TestCorporateThemeRegistered(t *testing.T) {
	if !ThemeExists("corporate") {
		t.Fatalf("corporate theme not registered, got %v", Themes())
	}
}

// TestCorporateThemeTemplatesParse 确保主题的全部模板都能被加载。
func TestCorporateThemeTemplatesParse(t *testing.T) {
	setupCorporateTheme(t)

	for name, tmpl := range map[string]any{
		"index": IndexTmpl, "post": PostTmpl, "singular": SingularTmpl, "author": AuthorTmpl,
		"moment": MomentTmpl, "whisper": WhisperTmpl, "about": AboutTmpl, "404": NotFoundTmpl,
	} {
		if tmpl == nil {
			t.Fatalf("%s template is nil", name)
		}
	}
}

// TestCorporateTwoColumnGrid 校验首页把所有内容平铺成左右两栏，不再有精选大卡片。
func TestCorporateTwoColumnGrid(t *testing.T) {
	setupCorporateTheme(t)

	posts := []*entity.PostR{
		corporatePost("first", "便利蜂是怎样选址的？", "正文一。"),
		corporatePost("second", "一千一百人遇难之后", "正文二。"),
		corporatePost("third", "网格化人口数据", "正文三。"),
	}
	data := indexData(posts, []*entity.NavigationR{
		{Name: "团队成员登录", URL: "/login"},
		{Name: "加入我们", URL: "/about"},
	}, 3)

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		`class="post-grid" id="latest"`,
		`便利蜂是怎样选址的？`,
		`一千一百人遇难之后`,
		`网格化人口数据`,
		`极海 | GeoHey Blog`,
		`阅读预计需要`, // 阅读时长文案
		`团队成员登录`,
		`hero`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered homepage missing %q", want)
		}
	}

	// 精选卡片及其轮播交互必须彻底移除。
	for _, unwanted := range []string{
		"featured-card", "featured-column", "home-layout", "articles-column",
		"data-featured-card", "data-featured-next", "arrow-btn", "has-featured",
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("homepage should no longer contain %q", unwanted)
		}
	}

	// 每篇内容各渲染一次卡片，数量与文章数一致。
	if n := strings.Count(out, `class="article-card`); n != len(posts) {
		t.Errorf("expected %d article cards, got %d", len(posts), n)
	}
}

// TestCorporateSinglePostLayout 校验仅有一篇内容时也能正常渲染单张卡片。
func TestCorporateSinglePostLayout(t *testing.T) {
	setupCorporateTheme(t)

	data := indexData([]*entity.PostR{corporatePost("only", "唯一的一篇", "正文。")}, []*entity.NavigationR{}, 1)

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "home-layout") {
		t.Error("single post should not render the old two-column layout wrapper")
	}
	if !strings.Contains(out, "唯一的一篇") {
		t.Error("single post should render as a regular card")
	}
}

// TestCorporateSingleLazyLoader 回归测试：主题只能有一个懒加载实现。
// 历史上模板同时引入了共享的 lazy-img.js 与主题自己的 corporate.js，
// 两个观察者抢同一个 data-src，后触发的一方读到 null 并把 src 写成 "null"，
// 表现为"刷新后图片闪一下变成裂图"。
func TestCorporateSingleLazyLoader(t *testing.T) {
	setupCorporateTheme(t)

	// 1) 模板不得再引入共享懒加载脚本
	base, err := fs.ReadFile(ThemesFS, "themes/corporate/template.html")
	if err != nil {
		t.Fatalf("read template.html: %v", err)
	}
	if bytes.Contains(base, []byte("/assets/lazy-img.js")) {
		t.Error(`template.html must not load the shared lazy-img.js: it competes with corporate.js for data-src`)
	}
	if !bytes.Contains(base, []byte("/assets/corporate.js")) {
		t.Error("template.html should load the theme's own corporate.js")
	}

	// 2) 主题不得自带 lazy-img.js 覆盖共享实现
	if _, err := fs.Stat(ThemesFS, "themes/corporate/assets/lazy-img.js"); err == nil {
		t.Error("corporate theme must not ship its own lazy-img.js")
	}

	// 3) 渲染结果里每个 <img> 的图片地址只由 data-src 承载一次
	post := corporatePost("only", "图片文章", "![配图](/uploads/images/a.jpg)")
	data := indexData([]*entity.PostR{post}, []*entity.NavigationR{}, 1)

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "lazy-img.js") {
		t.Error("rendered homepage still references the shared lazy-img.js")
	}
	if n := strings.Count(out, "data-src="); n == 0 {
		t.Error("lazy images should carry data-src for the theme loader")
	}
	if strings.Contains(out, `src="null"`) || strings.Contains(out, `src='null'`) {
		t.Error("rendered markup must not contain a null image source")
	}
}

// TestCorporateEmptyPosts 校验没有任何内容时展示空态而不是报错。
func TestCorporateEmptyPosts(t *testing.T) {
	setupCorporateTheme(t)

	data := indexData([]*entity.PostR{}, []*entity.NavigationR{}, 0)

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}
	if !strings.Contains(buf.String(), "empty-state") {
		t.Error("empty post list should render the empty state")
	}
}

// TestCorporateHeroUsesFirstContentImage 校验缺少封面时回退到正文首图作为头图。
func TestCorporateHeroUsesFirstContentImage(t *testing.T) {
	setupCorporateTheme(t)

	post := corporatePost("only", "带首图的文章", "前言\n\n![配图](/uploads/images/hero.jpg)\n\n正文。")
	data := indexData([]*entity.PostR{post}, []*entity.NavigationR{}, 1)

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}
	if !strings.Contains(buf.String(), "/uploads/images/hero.jpg") {
		t.Error("hero should fall back to the first image in the post content")
	}
}

// TestCorporateSearchState 校验搜索态下的标题与重置入口。
func TestCorporateSearchState(t *testing.T) {
	setupCorporateTheme(t)

	data := indexData([]*entity.PostR{corporatePost("only", "搜索结果", "正文。")}, []*entity.NavigationR{}, 1)
	data["Search"] = "选址"

	var buf bytes.Buffer
	if err := IndexTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute index template: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "选址") {
		t.Error("search keyword should be echoed in the section title")
	}
	if !strings.Contains(out, "重置过滤器") {
		t.Error("search state should offer a reset filter action")
	}
}

// TestCorporateArchiveLayout 校验归档页在文章没有封面时回退到正文首图，
// 该场景曾在 {{ with firstImage }} 内引用外层的 .Title 导致模板执行失败。
func TestCorporateArchiveLayout(t *testing.T) {
	setupCorporateTheme(t)

	post := corporatePost("first", "没有封面但有首图的文章", "前言\n\n![配图](/uploads/images/inline.jpg)\n\n正文。")
	data := indexData([]*entity.PostR{post}, []*entity.NavigationR{}, 1)

	var buf bytes.Buffer
	if err := PostTmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute post template: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "/uploads/images/inline.jpg") {
		t.Error("archive card should fall back to the first image in the post content")
	}
	if !strings.Contains(out, "全部文章") {
		t.Error("archive page should render its page title")
	}
}
