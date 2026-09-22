package handler

import (
	"bytes"
	htmltemplate "html/template"
	"log"
	"strings"
	texttemplate "text/template"

	"golog/entity"
	"golog/mailer"
	"golog/store"
	"golog/system"

	"github.com/gin-gonic/gin"
)

// 留言邮件通知。
//
// 触发时机与收件人：
//   - 新留言/新回复提交后（状态 pending）：通知文章作者，以及可选的管理员邮箱
//     （MailAdminEmail）。这样作者能第一时间知道有待审核内容。
//   - 回复通过审核后：通知“被回复留言”的作者（MailNotifyReply）。第三方只有在
//     内容真正公开可见后才会收到邮件，避免未审核内容被当作垃圾邮件的载体。
//
// 所有发送都在独立 goroutine 中完成，失败只写日志，不影响访客的提交结果。

// MailTemplateData 是自定义邮件模板可用的上下文。
// 模板中可直接使用下列字段，例如 {{ .CommentAuthor }}、{{ .PostTitle }}。
type MailTemplateData struct {
	SiteName   string // 站点名称
	SiteURL    string // 站点根地址，如 https://example.com
	AdminURL   string // 后台留言管理地址
	PostTitle  string
	PostURL    string // 文章地址（带 #comment-<id> 锚点）
	PostAuthor string

	CommentAuthor  string
	CommentEmail   string
	CommentURL     string
	CommentContent string
	CommentDate    string

	IsReply       bool   // 当前邮件对应的是否为回复
	ParentAuthor  string // 被回复留言的作者名
	ParentContent string // 被回复留言的内容
}

// MailTemplateSet 是一组（新留言、回复）邮件模板。
type MailTemplateSet struct {
	NewSubject   string
	NewBody      string
	ReplySubject string
	ReplyBody    string
}

func defaultMailTemplatesZH() MailTemplateSet {
	return MailTemplateSet{
		NewSubject: `[{{ .SiteName }}] {{ if .IsReply }}新回复{{ else }}新留言{{ end }}：{{ .PostTitle }}`,
		NewBody: `<div style="font-family:-apple-system,'Segoe UI',Roboto,'PingFang SC','Microsoft YaHei',sans-serif;line-height:1.7;color:#333;max-width:640px">
  <p>{{ .SiteName }} 的《<a href="{{ .PostURL }}">{{ .PostTitle }}</a>》有一条{{ if .IsReply }}新回复{{ else }}新留言{{ end }}，当前状态为「待审核」。</p>
  {{ if .IsReply }}
  <p style="color:#666;margin-bottom:4px">回复对象：{{ .ParentAuthor }}</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #e2e2e2;color:#666;white-space:pre-wrap">{{ .ParentContent }}</blockquote>
  {{ end }}
  <p style="margin-bottom:4px"><strong>{{ .CommentAuthor }}</strong>{{ if .CommentURL }}（<a href="{{ .CommentURL }}">{{ .CommentURL }}</a>）{{ end }} 于 {{ .CommentDate }} 写道：</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #ddd;color:#444;white-space:pre-wrap">{{ .CommentContent }}</blockquote>
  <p><a href="{{ .AdminURL }}">前往后台审核</a></p>
</div>`,
		ReplySubject: `[{{ .SiteName }}] {{ .CommentAuthor }} 回复了你的留言`,
		ReplyBody: `<div style="font-family:-apple-system,'Segoe UI',Roboto,'PingFang SC','Microsoft YaHei',sans-serif;line-height:1.7;color:#333;max-width:640px">
  <p>你在《<a href="{{ .PostURL }}">{{ .PostTitle }}</a>》中的留言收到了来自 <strong>{{ .CommentAuthor }}</strong> 的回复：</p>
  <p style="color:#666;margin-bottom:4px">你的留言：</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #e2e2e2;color:#666;white-space:pre-wrap">{{ .ParentContent }}</blockquote>
  <p style="color:#666;margin-bottom:4px">回复内容：</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #ddd;color:#444;white-space:pre-wrap">{{ .CommentContent }}</blockquote>
  <p><a href="{{ .PostURL }}">查看完整讨论</a></p>
</div>`,
	}
}

func defaultMailTemplatesEN() MailTemplateSet {
	return MailTemplateSet{
		NewSubject: `[{{ .SiteName }}] New {{ if .IsReply }}reply{{ else }}comment{{ end }} on "{{ .PostTitle }}"`,
		NewBody: `<div style="font-family:-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;line-height:1.7;color:#333;max-width:640px">
  <p>There is a new {{ if .IsReply }}reply{{ else }}comment{{ end }} awaiting moderation on <a href="{{ .PostURL }}">{{ .PostTitle }}</a> at {{ .SiteName }}.</p>
  {{ if .IsReply }}
  <p style="color:#666;margin-bottom:4px">In reply to {{ .ParentAuthor }}:</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #e2e2e2;color:#666;white-space:pre-wrap">{{ .ParentContent }}</blockquote>
  {{ end }}
  <p style="margin-bottom:4px"><strong>{{ .CommentAuthor }}</strong>{{ if .CommentURL }} (<a href="{{ .CommentURL }}">{{ .CommentURL }}</a>){{ end }} wrote on {{ .CommentDate }}:</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #ddd;color:#444;white-space:pre-wrap">{{ .CommentContent }}</blockquote>
  <p><a href="{{ .AdminURL }}">Review in the admin panel</a></p>
</div>`,
		ReplySubject: `[{{ .SiteName }}] {{ .CommentAuthor }} replied to your comment`,
		ReplyBody: `<div style="font-family:-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;line-height:1.7;color:#333;max-width:640px">
  <p>Your comment on <a href="{{ .PostURL }}">{{ .PostTitle }}</a> received a reply from <strong>{{ .CommentAuthor }}</strong>:</p>
  <p style="color:#666;margin-bottom:4px">Your comment:</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #e2e2e2;color:#666;white-space:pre-wrap">{{ .ParentContent }}</blockquote>
  <p style="color:#666;margin-bottom:4px">Reply:</p>
  <blockquote style="margin:0 0 16px;padding:8px 12px;border-left:3px solid #ddd;color:#444;white-space:pre-wrap">{{ .CommentContent }}</blockquote>
  <p><a href="{{ .PostURL }}">View the discussion</a></p>
</div>`,
	}
}

// DefaultMailTemplates 返回当前站点语言下内置的默认邮件模板，
// 供后台设置界面作为占位提示，并在自定义模板缺失时兜底。
func DefaultMailTemplates() MailTemplateSet {
	locale := ""
	if system.Config != nil {
		locale = system.Config.Locale
	}
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return defaultMailTemplatesEN()
	}
	return defaultMailTemplatesZH()
}

// mailerConfig 把站点配置转换为 mailer 的发信配置。
func mailerConfig(cfg *entity.Config) mailer.Config {
	return mailer.Config{
		Host:       cfg.MailHost,
		Port:       cfg.MailPortOrDefault(),
		Username:   cfg.MailUsername,
		Password:   cfg.MailPassword,
		Encryption: mailer.Encryption(cfg.MailEncryptionOrDefault()),
		FromName:   cfg.MailSenderName(),
		FromEmail:  cfg.MailSenderEmail(),
	}
}

// notifyNewComment 在留言/回复提交后异步通知文章作者与管理员邮箱。
func notifyNewComment(c *gin.Context, post *entity.PostR, comment *entity.CommentR, parent *entity.CommentR) {
	cfg := system.Config
	if cfg == nil || !cfg.MailConfigured() || !cfg.MailNotifyAuthor {
		return
	}

	recipients := uniqueEmails(append([]string{post.Author.Email}, cfg.MailAdminEmail))
	if len(recipients) == 0 {
		return
	}

	defaults := DefaultMailTemplates()
	data := buildMailTemplateData(c, post, comment, parent)
	subject := renderMailSubject(cfg.MailAuthorSubject, defaults.NewSubject, data)
	body := renderMailBody(cfg.MailAuthorBody, defaults.NewBody, data)
	if body == "" {
		return
	}
	sendMailAsync(mailerConfig(cfg), recipients, subject, body, "notify author about comment "+comment.ID)
}

// notifyReplyApproved 在回复通过审核后异步通知被回复留言的作者。
// 收件人为空、与回复作者相同（自己回复自己）时静默跳过。
func notifyReplyApproved(c *gin.Context, comment *entity.CommentR) {
	cfg := system.Config
	if cfg == nil || !cfg.MailConfigured() || !cfg.MailNotifyReply {
		return
	}
	if comment.ParentID == "" {
		return
	}
	parent, err := store.GetComment(comment.ParentID)
	if err != nil {
		log.Printf("[mail] reply %s: cannot load parent %s: %v", comment.ID, comment.ParentID, err)
		return
	}
	recipients := uniqueEmails([]string{parent.AuthorEmail})
	if len(recipients) == 0 || containsEmail(recipients, comment.AuthorEmail) {
		return
	}
	post, err := store.GetPost(comment.PostID)
	if err != nil {
		log.Printf("[mail] reply %s: cannot load post %s: %v", comment.ID, comment.PostID, err)
		return
	}

	defaults := DefaultMailTemplates()
	data := buildMailTemplateData(c, post, comment, parent)
	subject := renderMailSubject(cfg.MailReplySubject, defaults.ReplySubject, data)
	body := renderMailBody(cfg.MailReplyBody, defaults.ReplyBody, data)
	if body == "" {
		return
	}
	sendMailAsync(mailerConfig(cfg), recipients, subject, body, "notify reply recipient about comment "+comment.ID)
}

// sendMailAsync 在后台发送邮件：SMTP 往返可能耗时数秒，不应阻塞 HTTP 响应。
// 这里只捕获值，绝不把 gin.Context 带进 goroutine。
func sendMailAsync(cfg mailer.Config, to []string, subject, html, context string) {
	go func() {
		if err := mailer.Send(cfg, mailer.Message{To: to, Subject: subject, HTML: html}); err != nil {
			log.Printf("[mail] failed to %s (%v): %v", context, to, err)
		}
	}()
}

// buildMailTemplateData 组装邮件模板上下文。
// 文章地址带 #comment-<id> 锚点，方便收件人直接定位到该条留言。
func buildMailTemplateData(c *gin.Context, post *entity.PostR, comment *entity.CommentR, parent *entity.CommentR) MailTemplateData {
	base := siteBaseURL(c)
	postURL := base + "/post/" + post.Slug
	if comment != nil && comment.ID != "" {
		postURL += "#comment-" + comment.ID
	}
	data := MailTemplateData{
		SiteName:   system.Config.Name,
		SiteURL:    base,
		AdminURL:   base + "/admin/comments?status=pending",
		PostTitle:  post.Title,
		PostURL:    postURL,
		PostAuthor: post.Author.Nickname,
	}
	if comment != nil {
		data.CommentAuthor = comment.AuthorName
		data.CommentEmail = comment.AuthorEmail
		data.CommentURL = comment.AuthorURL
		data.CommentContent = comment.Content
		data.CommentDate = comment.CreatedDate()
	}
	if parent != nil {
		data.IsReply = true
		data.ParentAuthor = parent.AuthorName
		data.ParentContent = parent.Content
	}
	return data
}

// renderMailSubject 渲染主题行：优先自定义模板，解析/执行失败或结果为空时回退到默认模板。
func renderMailSubject(custom, fallback string, data MailTemplateData) string {
	for _, text := range []string{custom, fallback} {
		if strings.TrimSpace(text) == "" {
			continue
		}
		tmpl, err := texttemplate.New("subject").Option("missingkey=zero").Parse(text)
		if err != nil {
			log.Printf("[mail] invalid subject template: %v", err)
			continue
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("[mail] failed to render subject template: %v", err)
			continue
		}
		if out := strings.TrimSpace(buf.String()); out != "" {
			return out
		}
	}
	return ""
}

// renderMailBody 渲染 HTML 正文。使用 html/template，模板里写 HTML、
// {{ .CommentContent }} 等用户输入会被自动转义，避免留言内容注入邮件 HTML。
func renderMailBody(custom, fallback string, data MailTemplateData) string {
	for _, text := range []string{custom, fallback} {
		if strings.TrimSpace(text) == "" {
			continue
		}
		tmpl, err := htmltemplate.New("body").Option("missingkey=zero").Parse(text)
		if err != nil {
			log.Printf("[mail] invalid body template: %v", err)
			continue
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Printf("[mail] failed to render body template: %v", err)
			continue
		}
		if out := strings.TrimSpace(buf.String()); out != "" {
			return out
		}
	}
	return ""
}

// uniqueEmails 去除空白、非法（不含 @）与重复的地址，保持原有顺序。
func uniqueEmails(addrs []string) []string {
	seen := make(map[string]bool, len(addrs))
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" || !strings.Contains(addr, "@") || seen[strings.ToLower(addr)] {
			continue
		}
		seen[strings.ToLower(addr)] = true
		out = append(out, addr)
	}
	return out
}

func containsEmail(addrs []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, addr := range addrs {
		if strings.ToLower(strings.TrimSpace(addr)) == target {
			return true
		}
	}
	return false
}

// siteBaseURL 返回当前请求对应的站点根地址（含 scheme）。
func siteBaseURL(c *gin.Context) string {
	return requestScheme(c) + "://" + c.Request.Host
}
