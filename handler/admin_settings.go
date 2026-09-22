package handler

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"golog/entity"
	"golog/mailer"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// ============================
//  SettingsView
// ============================

func SettingsView(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_settings", data(c, gin.H{
		"Version":            injection.Version,
		"RuntimeVersion":     runtime.Version(),
		"BuildTime":          injection.BuildTime,
		"Commit":             injection.Commit,
		"GitHub":             `<a href="https://github.com/giserlab/golog" target="_blank">https://github.com/giserlab/golog</a>`,
		"Timezones":          entity.Timezones,
		"Locales":            entity.Locales,
		"IsCustomTimeFormat": system.Config.IsCustomTimeFormat(),
		"IsCustomDateFormat": system.Config.IsCustomDateFormat(),
		"Year":               time.Now().Format("2006"),
		"Month":              time.Now().Format("01"),
		"Day":                time.Now().Format("02"),
		"Hour":               time.Now().Format("03"),
		"Hour24":             time.Now().Format("15"),
		"Minute":             time.Now().Format("04"),
		"Clock":              time.Now().Format("PM"),
		"PoWEnabled":         system.Config.PoWEnabled,
		"PoWMaxNumber":       system.Config.PoWMaxNumber,
		"PoWTTL":             system.Config.PoWTTL,
		"PoWBotBypass":       system.Config.PoWBotBypass,
		"PoWBotUserAgents":   strings.Join(system.Config.PoWBotUserAgents, "\n"),
		"CommentsEnabled":    system.Config.CommentsEnabled,
		"MailTemplates":      DefaultMailTemplates(),
	}))
}

// ============================
//  SettingsEdit
// ============================

type SettingsEditRequest struct {
	Name              string `form:"name" binding:"required,max=64" conform:"trim"`
	Description       string `form:"description" binding:"required,max=128" conform:"trim"`
	About             string `form:"about" binding:"required" conform:"trim"`
	IsPublic          bool   `form:"is_public"`
	Timezone          int    `form:"timezone" binding:"min=-43200,max=50400"`
	DateFormat        string `form:"date_format" binding:"required"`
	DateFormatCustom  string `form:"date_format_custom" conform:"trim"`
	TimeFormat        string `form:"time_format" binding:"required"`
	TimeFormatCustom  string `form:"time_format_custom" conform:"trim"`
	Locale            string `form:"locale" binding:"required"`
	PoWEnabled        bool   `form:"pow_enabled"`
	PoWMaxNumber      int64  `form:"pow_max_number" binding:"min=1000,max=10000000"`
	PoWTTL            int    `form:"pow_ttl" binding:"min=1,max=168"`
	PoWBotBypass      bool   `form:"pow_bot_bypass"`
	PoWBotUserAgents  string `form:"pow_bot_user_agents" conform:"trim"`
	CommentsEnabled   bool   `form:"comments_enabled"`

	// 邮件通知：字段校验保持宽松（omitempty），否则邮件未启用时会连带
	// 阻止基础设置的保存。
	MailEnabled       bool   `form:"mail_enabled"`
	MailHost          string `form:"mail_host" binding:"omitempty,max=255" conform:"trim"`
	MailPort          int    `form:"mail_port" binding:"omitempty,min=1,max=65535"`
	MailUsername      string `form:"mail_username" binding:"omitempty,max=255" conform:"trim"`
	MailPassword      string `form:"mail_password"`
	MailPasswordClear bool   `form:"mail_password_clear"`
	MailEncryption    string `form:"mail_encryption" binding:"omitempty,oneof=none ssl starttls"`
	MailFromName      string `form:"mail_from_name" binding:"omitempty,max=128" conform:"trim"`
	MailFromEmail     string `form:"mail_from_email" binding:"omitempty,email,max=255" conform:"trim"`
	MailAdminEmail    string `form:"mail_admin_email" binding:"omitempty,email,max=255" conform:"trim"`
	MailNotifyAuthor  bool   `form:"mail_notify_author"`
	MailNotifyReply   bool   `form:"mail_notify_reply"`
	MailAuthorSubject string `form:"mail_author_subject" binding:"omitempty,max=255" conform:"trim"`
	MailAuthorBody    string `form:"mail_author_body"`
	MailReplySubject  string `form:"mail_reply_subject" binding:"omitempty,max=255" conform:"trim"`
	MailReplyBody     string `form:"mail_reply_body"`
}

func SettingsEdit(c *gin.Context, req *SettingsEditRequest) {
	system.Config.Name = req.Name
	system.Config.About = req.About
	system.Config.Description = req.Description
	system.Config.IsPublic = req.IsPublic
	system.Config.Timezone = req.Timezone
	system.Config.Locale = req.Locale
	system.Config.PoWEnabled = req.PoWEnabled
	if req.PoWMaxNumber >= 1000 && req.PoWMaxNumber <= 10000000 {
		system.Config.PoWMaxNumber = req.PoWMaxNumber
	} else if system.Config.PoWMaxNumber == 0 {
		system.Config.PoWMaxNumber = 200000 // default
	}
	if req.PoWTTL >= 1 && req.PoWTTL <= 168 {
		system.Config.PoWTTL = req.PoWTTL
	} else if system.Config.PoWTTL == 0 {
		system.Config.PoWTTL = 24 // default
	}
	system.Config.PoWBotBypass = req.PoWBotBypass
	system.Config.PoWBotUserAgents = parsePowBotUserAgents(req.PoWBotUserAgents)
	system.Config.CommentsEnabled = req.CommentsEnabled

	applyMailSettings(req)

	if req.DateFormat == "custom" {
		system.Config.DateFormat = req.DateFormatCustom
	} else {
		system.Config.DateFormat = req.DateFormat
	}
	if req.TimeFormat == "custom" {
		system.Config.TimeFormat = req.TimeFormatCustom
	} else {
		system.Config.TimeFormat = req.TimeFormat
	}
	if err := system.SaveConfig(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	setMessage(c, "notice_settings_updated")
	c.Redirect(http.StatusFound, "/admin/settings")
}

// parsePowBotUserAgents splits a newline-separated list of user-agent strings,
// trimming whitespace and filtering out empty lines.
func parsePowBotUserAgents(raw string) []string {
	if raw == "" {
		return nil
	}
	var agents []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			agents = append(agents, line)
		}
	}
	return agents
}

// applyMailSettings 把表单中的邮件配置写入 system.Config。
//
// 密码留空表示“保持原值”：设置页不回显密码，管理员只改其它字段时
// 不必重新输入；勾选“清除密码”可显式置空（例如改用免认证的本地中继）。
func applyMailSettings(req *SettingsEditRequest) {
	system.Config.MailEnabled = req.MailEnabled
	system.Config.MailHost = req.MailHost
	if req.MailPort > 0 {
		system.Config.MailPort = req.MailPort
	} else if system.Config.MailPort == 0 {
		system.Config.MailPort = 587
	}
	system.Config.MailUsername = req.MailUsername
	switch {
	case req.MailPasswordClear:
		system.Config.MailPassword = ""
	case req.MailPassword != "":
		system.Config.MailPassword = req.MailPassword
	}
	if req.MailEncryption != "" {
		system.Config.MailEncryption = entity.MailEncryption(req.MailEncryption)
	} else if system.Config.MailEncryption == "" {
		system.Config.MailEncryption = entity.MailEncryptionStartTLS
	}
	system.Config.MailFromName = req.MailFromName
	system.Config.MailFromEmail = req.MailFromEmail
	system.Config.MailAdminEmail = req.MailAdminEmail
	system.Config.MailNotifyAuthor = req.MailNotifyAuthor
	system.Config.MailNotifyReply = req.MailNotifyReply
	system.Config.MailAuthorSubject = req.MailAuthorSubject
	system.Config.MailAuthorBody = req.MailAuthorBody
	system.Config.MailReplySubject = req.MailReplySubject
	system.Config.MailReplyBody = req.MailReplyBody
}

// ============================
//  SettingsTestMail
// ============================

type SettingsTestMailRequest struct {
	Recipient string `form:"recipient" binding:"omitempty,email,max=255" conform:"trim"`
}

// SettingsTestMail 使用当前（已保存的）SMTP 配置同步发送一封测试邮件。
// 它按“新留言通知”模板渲染示例内容，因此管理员既能验证 SMTP 连通性，
// 也能直观看到自定义模板的最终效果。
func SettingsTestMail(c *gin.Context, req *SettingsTestMailRequest) {
	cfg := system.Config
	if cfg == nil || !cfg.MailConfigured() {
		setMessage(c, "notice_mail_not_configured")
		c.Redirect(http.StatusFound, "/admin/settings")
		return
	}

	recipient := req.Recipient
	if recipient == "" {
		if me, err := self(c); err == nil && me != nil {
			recipient = me.Email
		}
	}
	if recipient == "" {
		recipient = cfg.MailSenderEmail()
	}

	base := siteBaseURL(c)
	defaults := DefaultMailTemplates()
	data := MailTemplateData{
		SiteName:       cfg.Name,
		SiteURL:        base,
		AdminURL:       base + "/admin/comments",
		PostTitle:      system.Locale.String("settings_mail_test_post"),
		PostURL:        base + "/",
		PostAuthor:     cfg.Name,
		CommentAuthor:  system.Locale.String("settings_mail_test_author"),
		CommentEmail:   "visitor@example.com",
		CommentContent: system.Locale.String("settings_mail_test_content"),
		CommentDate:    time.Now().Format("2006-01-02 15:04"),
	}
	subject := renderMailSubject(cfg.MailAuthorSubject, defaults.NewSubject, data)
	body := renderMailBody(cfg.MailAuthorBody, defaults.NewBody, data)
	if body == "" {
		setMessage(c, "notice_mail_not_configured")
		c.Redirect(http.StatusFound, "/admin/settings")
		return
	}

	if err := mailer.Send(mailerConfig(cfg), mailer.Message{
		To:      []string{recipient},
		Subject: system.Locale.String("settings_mail_test_prefix") + subject,
		HTML:    body,
	}); err != nil {
		setMessage(c, fmt.Sprintf("%s: %v", system.Locale.String("notice_mail_failed"), err))
	} else {
		setMessage(c, fmt.Sprintf(system.Locale.String("notice_mail_sent"), recipient))
	}
	c.Redirect(http.StatusFound, "/admin/settings")
}
