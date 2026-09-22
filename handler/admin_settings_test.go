package handler

import (
	"bytes"
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
)

func newSettingsTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(sessions.Sessions("golog", cookie.NewStore([]byte("test-secret"))))
	r.POST("/admin/settings", handleForm(SettingsEdit))
	return r
}

func settingsForm() url.Values {
	form := url.Values{}
	form.Set("name", "测试站")
	form.Set("description", "描述")
	form.Set("about", "关于")
	form.Set("timezone", "0")
	form.Set("date_format", "2006-01-02")
	form.Set("time_format", "15:04")
	form.Set("locale", "zh-cn")
	form.Set("pow_max_number", "200000")
	form.Set("pow_ttl", "24")
	return form
}

func postSettings(t *testing.T, form url.Values) {
	t.Helper()
	r := newSettingsTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("unexpected status %d, body=%s", w.Code, w.Body.String())
	}
}

// TestSettingsEditPersistsMailConfig 覆盖邮件配置的保存，包括
// “密码留空 = 保持原值”与显式清除密码的语义。
func TestSettingsEditPersistsMailConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restoreSystemState(t)

	system.Config = &entity.Config{
		Theme: "default", Locale: "zh-cn",
		DateFormat: "2006-01-02", TimeFormat: "15:04", PostsPerPage: 10,
		MailPassword: "old-secret",
	}
	system.ReloadLocale("zh-cn")

	var saved *entity.Config
	writeConfig := func(c *entity.Config) error {
		cp := *c
		saved = &cp
		return nil
	}
	system.SetConfigWriter(writeConfig)

	// 1) 修改 SMTP 设置但密码留空 → 保留原密码。
	form := settingsForm()
	form.Set("mail_enabled", "true")
	form.Set("mail_host", "smtp.example.com")
	form.Set("mail_port", "465")
	form.Set("mail_encryption", "ssl")
	form.Set("mail_username", "bot@example.com")
	form.Set("mail_from_name", "测试站")
	form.Set("mail_from_email", "noreply@example.com")
	form.Set("mail_admin_email", "admin@example.com")
	form.Set("mail_notify_author", "true")
	form.Set("mail_notify_reply", "true")
	form.Set("mail_author_subject", "主题 {{ .PostTitle }}")
	form.Set("mail_reply_body", "<p>{{ .CommentContent }}</p>")
	postSettings(t, form)

	if saved == nil {
		t.Fatal("configWriter was not called")
	}
	if !saved.MailEnabled || saved.MailHost != "smtp.example.com" || saved.MailPort != 465 {
		t.Fatalf("mail server settings not saved: %+v", saved)
	}
	if saved.MailEncryption != entity.MailEncryptionSSL {
		t.Fatalf("MailEncryption = %q, want ssl", saved.MailEncryption)
	}
	if saved.MailPassword != "old-secret" {
		t.Fatalf("empty password field must keep the stored password, got %q", saved.MailPassword)
	}
	if !saved.MailNotifyAuthor || !saved.MailNotifyReply {
		t.Fatalf("notification toggles not saved: %+v", saved)
	}
	if saved.MailAuthorSubject != "主题 {{ .PostTitle }}" {
		t.Fatalf("MailAuthorSubject = %q", saved.MailAuthorSubject)
	}

	// 2) 显式清除密码。
	form = settingsForm()
	form.Set("mail_password_clear", "true")
	postSettings(t, form)
	if saved.MailPassword != "" {
		t.Fatalf("password should have been cleared, got %q", saved.MailPassword)
	}

	// 3) 填写新密码。
	form = settingsForm()
	form.Set("mail_password", "new-secret")
	postSettings(t, form)
	if saved.MailPassword != "new-secret" {
		t.Fatalf("MailPassword = %q, want new-secret", saved.MailPassword)
	}

	// 4) 未填端口/加密方式时保留已有值（输入框始终渲染，清空即“不改”）。
	form = settingsForm()
	form.Set("mail_host", "smtp.example.com")
	postSettings(t, form)
	if saved.MailPort != 465 {
		t.Fatalf("MailPort = %d, want the previously saved 465", saved.MailPort)
	}
	if saved.MailEncryption != entity.MailEncryptionSSL {
		t.Fatalf("MailEncryption = %q, want the previously saved ssl", saved.MailEncryption)
	}
}

// TestSettingsEditMailDefaults 覆盖从未配置过邮件时的缺省值补齐。
func TestSettingsEditMailDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restoreSystemState(t)

	system.Config = &entity.Config{
		Theme: "default", Locale: "zh-cn",
		DateFormat: "2006-01-02", TimeFormat: "15:04", PostsPerPage: 10,
	}
	system.ReloadLocale("zh-cn")

	var saved *entity.Config
	system.SetConfigWriter(func(c *entity.Config) error {
		cp := *c
		saved = &cp
		return nil
	})

	form := settingsForm()
	form.Set("mail_enabled", "true")
	form.Set("mail_host", "smtp.example.com")
	postSettings(t, form)

	if saved == nil {
		t.Fatal("configWriter was not called")
	}
	if saved.MailPort != 587 {
		t.Fatalf("MailPort = %d, want 587", saved.MailPort)
	}
	if saved.MailEncryption != entity.MailEncryptionStartTLS {
		t.Fatalf("MailEncryption = %q, want starttls", saved.MailEncryption)
	}
}

// TestSettingsTemplateMailSection 结构性检查：邮件设置项存在、密码不回显、
// 测试邮件是独立表单（不能嵌在设置主表单里）。
func TestSettingsTemplateMailSection(t *testing.T) {
	b, err := view.Templates.ReadFile("templates/admin_settings.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	page := string(b)

	for _, want := range []string{
		`name="mail_enabled"`,
		`name="mail_host"`,
		`name="mail_port"`,
		`name="mail_username"`,
		`name="mail_password"`,
		`name="mail_password_clear"`,
		`name="mail_encryption"`,
		`name="mail_from_email"`,
		`name="mail_admin_email"`,
		`name="mail_notify_author"`,
		`name="mail_notify_reply"`,
		`name="mail_author_subject"`,
		`name="mail_author_body"`,
		`name="mail_reply_subject"`,
		`name="mail_reply_body"`,
		`admin/settings/test-mail`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("admin settings page missing %q", want)
		}
	}

	// 已保存的 SMTP 密码绝不能回显到 HTML 中：密码框必须始终为空，
	// 仅用占位符提示“已设置密码”。
	if !strings.Contains(page, `name="mail_password" value=""`) {
		t.Error("the SMTP password input must be rendered empty")
	}
	if strings.Contains(page, `value="{{ .Config.MailPassword }}"`) {
		t.Error("the stored SMTP password must not be rendered into the settings page")
	}
}

// TestSettingsTemplateRendersMailSection 实际执行模板，确保新增的邮件设置
// 区域在运行时不会因为字段缺失或函数用法错误而渲染失败。
func TestSettingsTemplateRendersMailSection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restoreSystemState(t)

	system.Config = &entity.Config{
		Name: "测试站", Theme: "default", Locale: "zh-cn",
		DateFormat: "2006-01-02", TimeFormat: "15:04", PostsPerPage: 10,
		MailEnabled: true, MailHost: "smtp.example.com", MailPort: 465,
		MailEncryption: entity.MailEncryptionSSL, MailUsername: "bot@example.com",
		MailPassword: "secret", MailFromEmail: "noreply@example.com",
		MailNotifyAuthor: true,
	}
	system.ReloadLocale("zh-cn")

	tmpl, err := template.New("admin_base.html").Funcs(funcs).ParseFS(
		view.Templates, "templates/admin_base.html", "templates/admin_settings.html")
	if err != nil {
		t.Fatalf("parse admin settings templates: %v", err)
	}

	stats := map[[2]string]int{}
	momentStats := map[string]int{}
	tagMap := map[[2]string]int{}
	data := map[string]any{
		"Config":              system.Config,
		"Self":                &entity.UserR{ID: "u1", Nickname: "管理员", Role: "admin", Email: "admin@example.com"},
		"Message":             "",
		"CSRF":                "csrf-token",
		"PendingCommentCount": 2,
		"URL": map[string]string{
			"Root": "http://localhost/", "Absolute": "http://localhost/admin/settings",
			"AbsoluteHost": "http://localhost/", "RelativeRoot": "../../", "PageType": "settings",
		},
		"Stats": &stats, "MomentStats": &momentStats, "TagMap": &tagMap,
		"Timezones": entity.Timezones, "Locales": entity.Locales,
		"IsCustomTimeFormat": false, "IsCustomDateFormat": false,
		"Year": "2024", "Month": "01", "Day": "02",
		"Hour": "03", "Hour24": "15", "Minute": "04", "Clock": "PM",
		"PoWEnabled": false, "PoWMaxNumber": 200000, "PoWTTL": 24,
		"PoWBotBypass": false, "PoWBotUserAgents": "",
		"CommentsEnabled": true,
		"MailTemplates":   DefaultMailTemplates(),
		"Version":         "dev", "RuntimeVersion": "go1.25", "BuildTime": "now", "Commit": "abc",
		"GitHub": `<a href="https://github.com/giserlab/golog">repo</a>`,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "admin_base.html", data); err != nil {
		t.Fatalf("execute admin settings template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"smtp.example.com",
		`name="mail_password" value=""`,
		`placeholder="••••••••"`, // 已设置密码时给出提示
		`name="mail_encryption"`,
		"admin/settings/test-mail",
		`{{ .SiteName }}`, // 变量说明
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered settings page missing %q", want)
		}
	}
}
