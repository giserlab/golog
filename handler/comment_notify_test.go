package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// newMailTestContext 构造一个带请求的 gin 上下文，用于验证邮件里的绝对地址。
func newMailTestContext(t *testing.T, target, forwardedProto string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	if forwardedProto != "" {
		c.Request.Header.Set("X-Forwarded-Proto", forwardedProto)
	}
	return c
}

func withTestConfig(t *testing.T, cfg *entity.Config) {
	t.Helper()
	original := system.Config
	system.Config = cfg
	t.Cleanup(func() { system.Config = original })
}

func sampleMailData() MailTemplateData {
	return MailTemplateData{
		SiteName:       "测试站",
		SiteURL:        "https://blog.example.com",
		AdminURL:       "https://blog.example.com/admin/comments?status=pending",
		PostTitle:      "标题",
		PostURL:        "https://blog.example.com/post/hello#comment-c1",
		PostAuthor:     "作者",
		CommentAuthor:  "访客",
		CommentEmail:   "visitor@example.com",
		CommentURL:     "https://visitor.example.com",
		CommentContent: "留言正文",
		CommentDate:    "2024-01-02 03:04",
		IsReply:        true,
		ParentAuthor:   "楼主",
		ParentContent:  "原留言",
	}
}

// TestRenderMailDefaults 断言内置模板能渲染出主题与正文中的关键内容。
func TestRenderMailDefaults(t *testing.T) {
	withTestConfig(t, &entity.Config{Name: "测试站", Locale: "zh-cn"})
	defaults := DefaultMailTemplates()
	data := sampleMailData()

	subject := renderMailSubject("", defaults.NewSubject, data)
	if !strings.Contains(subject, "新回复") || !strings.Contains(subject, "标题") {
		t.Fatalf("unexpected default subject: %q", subject)
	}

	body := renderMailBody("", defaults.NewBody, data)
	for _, want := range []string{"留言正文", "楼主", "原留言", "待审核"} {
		if !strings.Contains(body, want) {
			t.Fatalf("default body missing %q:\n%s", want, body)
		}
	}

	replySubject := renderMailSubject("", defaults.ReplySubject, data)
	if !strings.Contains(replySubject, "访客") {
		t.Fatalf("unexpected default reply subject: %q", replySubject)
	}
}

// TestDefaultMailTemplatesFollowLocale 断言默认模板随站点语言切换。
func TestDefaultMailTemplatesFollowLocale(t *testing.T) {
	withTestConfig(t, &entity.Config{Name: "Site", Locale: "en-us"})
	if got := DefaultMailTemplates().NewSubject; !strings.Contains(got, "New ") || !strings.Contains(got, "reply") {
		t.Fatalf("english subject = %q", got)
	}
	withTestConfig(t, &entity.Config{Name: "站点", Locale: "zh-cn"})
	if got := DefaultMailTemplates().NewSubject; !strings.Contains(got, "新留言") {
		t.Fatalf("chinese subject = %q", got)
	}
}

// TestRenderMailBodyEscapesUserContent 断言留言内容不会作为 HTML 注入邮件正文。
func TestRenderMailBodyEscapesUserContent(t *testing.T) {
	data := sampleMailData()
	data.CommentContent = `<script>alert("xss")</script>`
	body := renderMailBody("{{ .CommentContent }}", "", data)
	if strings.Contains(body, "<script>") {
		t.Fatalf("comment content was not escaped: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("expected escaped content, got: %s", body)
	}
}

// TestRenderMailTemplateFallback 断言模板解析失败时回退到内置模板，
// 而不是整封通知静默丢失。
func TestRenderMailTemplateFallback(t *testing.T) {
	withTestConfig(t, &entity.Config{Name: "测试站", Locale: "zh-cn"})
	data := sampleMailData()
	body := renderMailBody("{{ .Broken", DefaultMailTemplates().NewBody, data)
	if !strings.Contains(body, "留言正文") {
		t.Fatalf("expected fallback body, got: %s", body)
	}
}

// TestRenderMailBodyEmptyWhenNoTemplate 断言没有任何可用模板时返回空串，
// 让调用方放弃发送而不是发出空邮件。
func TestRenderMailBodyEmptyWhenNoTemplate(t *testing.T) {
	if got := renderMailBody("", "", sampleMailData()); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}

func TestBuildMailTemplateData(t *testing.T) {
	withTestConfig(t, &entity.Config{Name: "测试站", Locale: "zh-cn"})
	c := newMailTestContext(t, "https://blog.example.com/post/hello", "https")

	post := &entity.PostR{ID: "p1", Title: "标题", Slug: "hello"}
	post.Author = entity.UserR{ID: "u1", Nickname: "作者", Email: "author@example.com"}
	comment := &entity.CommentR{ID: "c1", PostID: "p1", ParentID: "c0", AuthorName: "访客", Content: "回复内容", CreatedAt: 1700000000}
	parent := &entity.CommentR{ID: "c0", AuthorName: "楼主", Content: "原留言"}

	data := buildMailTemplateData(c, post, comment, parent)
	if data.PostURL != "https://blog.example.com/post/hello#comment-c1" {
		t.Fatalf("PostURL = %q", data.PostURL)
	}
	if data.SiteURL != "https://blog.example.com" {
		t.Fatalf("SiteURL = %q", data.SiteURL)
	}
	if !data.IsReply || data.ParentAuthor != "楼主" || data.ParentContent != "原留言" {
		t.Fatalf("reply context not populated: %+v", data)
	}
	if data.AdminURL != "https://blog.example.com/admin/comments?status=pending" {
		t.Fatalf("AdminURL = %q", data.AdminURL)
	}

	// 顶层留言：不携带回复上下文。
	top := buildMailTemplateData(c, post, &entity.CommentR{ID: "c2"}, nil)
	if top.IsReply || top.ParentAuthor != "" {
		t.Fatalf("top-level comment must not be marked as reply: %+v", top)
	}
}

func TestUniqueEmails(t *testing.T) {
	got := uniqueEmails([]string{" A@Example.com ", "a@example.com", "", "not-an-email", "b@example.com"})
	want := []string{"A@Example.com", "b@example.com"}
	if len(got) != len(want) {
		t.Fatalf("uniqueEmails = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniqueEmails = %v, want %v", got, want)
		}
	}
}

func TestContainsEmail(t *testing.T) {
	if !containsEmail([]string{"A@Example.com"}, " a@example.com ") {
		t.Fatal("containsEmail should match case-insensitively")
	}
	if containsEmail([]string{"a@example.com"}, "") {
		t.Fatal("empty target must not match")
	}
}
