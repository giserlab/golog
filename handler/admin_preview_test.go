package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"
	"golog/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestPreviewRouteRequiresLogin 验证作者预览路由 /preview/:slug 必须登录：
// 未登录访问应被重定向到 /login，而不是渲染文章或走公开路由逻辑。
func TestPreviewRouteRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restoreConfig := false
	if system.Config == nil {
		system.Config = &entity.Config{Theme: "default"}
		restoreConfig = true
	}
	defer func() {
		if restoreConfig {
			system.Config = nil
		}
	}()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/preview/some-slug", nil)
	Router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("got status %d, want 302 (redirect to login)", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Fatalf("got redirect location %q, want /login", loc)
	}
}

// TestPublicPostRouteHidesDraft 验证公开路由 /post/:slug 不能查看草稿：
// 无论是否登录，访问草稿 slug 一律 404；草稿只能通过作者预览路由
// /preview/:slug（登录且为作者/管理员）查看。
func TestPublicPostRouteHidesDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// --- 独立的临时数据库（与真实数据隔离） ---
	tmpDir := t.TempDir()
	if err := store.Open(filepath.Join(tmpDir, "test.sqlite")); err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	defer store.Open("golog.sqlite") // 恢复 init() 默认连接

	if err := store.AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// --- 种子数据：管理员作者 + 一篇草稿 + 一篇公开文章 ---
	now := time.Now().Unix()
	passHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	author := &entity.UserW{
		ID:        uuid.New().String(),
		Email:     "author@example.com",
		Password:  string(passHash),
		Nickname:  "Author",
		Role:      "admin",
		CreatedAt: now,
	}
	if err := store.CreateUser(author); err != nil {
		t.Fatalf("create user: %v", err)
	}
	seedPost := func(slug, title string, vis entity.Visibility) {
		p := &entity.PostW{
			ID:          uuid.New().String(),
			Type:        util.BlogType,
			Title:       title,
			Slug:        slug,
			AuthorID:    author.ID,
			Visibility:  vis,
			Content:     "body of " + title,
			PublishedAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := store.CreatePost(p); err != nil {
			t.Fatalf("create post %s: %v", slug, err)
		}
	}
	seedPost("draft-post", "Draft Post", entity.VisibilityDraft)
	seedPost("public-post", "Public Post", entity.VisibilityPublic)

	// --- 站点配置 + 主题模板 + 语言包（备份/恢复） ---
	// 注意：SaveConfig 会加载 NotFoundTmpl 等主题模板，必须在清理时还原，
	// 否则后续依赖“NotFoundTmpl 为 nil 时 noRoute 直接 404”的测试会受影响。
	oldConfig := system.Config
	oldNotFoundTmpl := system.NotFoundTmpl
	system.Config = &entity.Config{
		Theme:    "default",
		Locale:   "zh-cn",
		Name:     "Test Blog",
		IsPublic: true, // 匿名可访问公开路由
	}
	system.SetConfigWriter(func(*entity.Config) error { return nil })
	if err := system.SaveConfig(); err != nil {
		t.Fatalf("save config: %v", err)
	}
	t.Cleanup(func() {
		system.Config = oldConfig
		system.NotFoundTmpl = oldNotFoundTmpl
		// 清空共享的按 IP 限流器：POST /login 经过 throttle（1 次/秒），
		// 避免 -count>1 或重复运行时同一 IP 在 1 秒内再次登录被 429。
		cleanupThrottleLimiters()
	})

	// --- 通过真实路由 + 会话/CSRF 中间件走完整登录流程 ---
	jar := map[string]string{}
	captureCookies := func(w *httptest.ResponseRecorder) {
		for _, c := range w.Result().Cookies() {
			jar[c.Name] = c.Value
		}
	}
	do := func(method, path string, form url.Values) *httptest.ResponseRecorder {
		var body io.Reader
		if form != nil {
			body = strings.NewReader(form.Encode())
		}
		req := httptest.NewRequest(method, path, body)
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if len(jar) > 0 {
			parts := make([]string, 0, len(jar))
			for k, v := range jar {
				parts = append(parts, k+"="+v)
			}
			req.Header.Set("Cookie", strings.Join(parts, "; "))
		}
		w := httptest.NewRecorder()
		Router.ServeHTTP(w, req)
		captureCookies(w)
		return w
	}

	// 1) 未登录访问草稿的公开路由 → 404
	if w := do(http.MethodGet, "/post/draft-post", nil); w.Code != http.StatusNotFound {
		t.Fatalf("anonymous GET /post/draft-post = %d, want 404", w.Code)
	}

	// 2) 未登录访问公开文章的公开路由 → 200（对照：公开文章不受影响）
	if w := do(http.MethodGet, "/post/public-post", nil); w.Code != http.StatusOK {
		t.Fatalf("anonymous GET /post/public-post = %d, want 200", w.Code)
	}

	// 3) 登录作者
	w := do(http.MethodGet, "/login", nil)
	csrfRe := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
	m := csrfRe.FindStringSubmatch(w.Body.String())
	if len(m) != 2 {
		t.Fatalf("could not extract csrf token from login page")
	}
	w = do(http.MethodPost, "/login", url.Values{
		"email":    {"author@example.com"},
		"password": {"password123"},
		"_csrf":    {m[1]},
	})
	if w.Code != http.StatusFound {
		t.Fatalf("POST /login = %d, want 302", w.Code)
	}

	// 4) 登录后访问草稿的公开路由 → 仍然 404
	if w := do(http.MethodGet, "/post/draft-post", nil); w.Code != http.StatusNotFound {
		t.Fatalf("logged-in GET /post/draft-post = %d, want 404", w.Code)
	}

	// 5) 作者预览路由可查看草稿 → 200
	if w := do(http.MethodGet, "/preview/draft-post", nil); w.Code != http.StatusOK {
		t.Fatalf("logged-in GET /preview/draft-post = %d, want 200", w.Code)
	}
}
