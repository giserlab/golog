// Package mailer 提供最小化的 SMTP 发信能力，仅依赖标准库。
//
// 支持三种连接方式：
//   - ssl      隐式 TLS（SMTPS，常见端口 465）
//   - starttls 显式 TLS（常见端口 587）
//   - none     明文（仅供本地中继或受信内网使用）
//
// 邮件正文以 multipart/alternative 发送：纯文本部分缺失时由 HTML 自动降级生成，
// 保证纯文本客户端（以及部分垃圾邮件过滤器）仍能阅读。邮件内容模板的渲染由
// 调用方负责，本包只关心“把一段 HTML/纯文本可靠地投递出去”。
package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// dialTimeout 限制单次 SMTP 操作的连接建立耗时，避免后台 goroutine 永久阻塞。
const dialTimeout = 15 * time.Second

// Encryption 指定 SMTP 连接的加密方式。
type Encryption string

const (
	EncryptionNone     Encryption = "none"
	EncryptionSSL      Encryption = "ssl"
	EncryptionStartTLS Encryption = "starttls"
)

// Config 是一次发信所需的 SMTP 服务器与发件人信息。
type Config struct {
	Host       string
	Port       int
	Username   string
	Password   string
	Encryption Encryption
	FromName   string
	FromEmail  string
}

// Normalize 补齐缺省值并返回规范化后的副本：端口默认 587，
// 加密方式默认 STARTTLS，发件人邮箱缺省回退到 SMTP 用户名。
func (c Config) Normalize() Config {
	if c.Port == 0 {
		c.Port = 587
	}
	if c.Encryption == "" {
		c.Encryption = EncryptionStartTLS
	}
	if c.FromEmail == "" {
		c.FromEmail = c.Username
	}
	return c
}

// Validate 检查发信必需字段，配置不完整时返回可读的错误。
func (c Config) Validate() error {
	c = c.Normalize()
	switch {
	case strings.TrimSpace(c.Host) == "":
		return errors.New("smtp host is empty")
	case c.Port < 1 || c.Port > 65535:
		return fmt.Errorf("smtp port %d out of range", c.Port)
	case strings.TrimSpace(c.FromEmail) == "":
		return errors.New("sender email is empty")
	}
	switch c.Encryption {
	case EncryptionNone, EncryptionSSL, EncryptionStartTLS:
	default:
		return fmt.Errorf("unsupported encryption mode %q", c.Encryption)
	}
	return nil
}

// Message 是一封待发送的邮件。
type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

// Send 通过 SMTP 同步发送一封邮件。调用方若不想阻塞请求，
// 应自行在 goroutine 中调用并处理返回的错误。
func Send(cfg Config, msg Message) error {
	cfg = cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return err
	}
	recipients := normalizeRecipients(msg.To)
	if len(recipients) == 0 {
		return errors.New("no recipients")
	}
	subject := strings.TrimSpace(msg.Subject)
	if subject == "" {
		subject = "(no subject)"
	}
	text := msg.Text
	if strings.TrimSpace(text) == "" {
		text = HTMLToText(msg.HTML)
	}

	client, err := dial(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := authenticate(client, cfg); err != nil {
		return err
	}
	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(buildMessage(cfg, recipients, subject, text, msg.HTML)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("finish message: %w", err)
	}
	return client.Quit()
}

// dial 按加密方式建立连接并返回已握手的 SMTP 客户端。
func dial(cfg Config) (*smtp.Client, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	dialer := &net.Dialer{Timeout: dialTimeout}
	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}

	if cfg.Encryption == EncryptionSSL {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("smtp ssl dial %s: %w", addr, err)
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("smtp ssl handshake: %w", err)
		}
		return client, nil
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("smtp handshake: %w", err)
	}
	if cfg.Encryption == EncryptionStartTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			_ = client.Close()
			return nil, errors.New("smtp server does not advertise STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("smtp starttls: %w", err)
		}
	}
	return client, nil
}

// authenticate 在配置了用户名时执行 SMTP AUTH；服务器未声明 AUTH 支持则报错，
// 避免“静默地以匿名身份发信”导致难以排查的投递失败。
func authenticate(client *smtp.Client, cfg Config) error {
	if cfg.Username == "" {
		return nil
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("smtp server does not support AUTH but a username is configured")
	}
	if err := client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	return nil
}

// normalizeRecipients 去除空白与重复地址，保持原有顺序。
func normalizeRecipients(to []string) []string {
	seen := make(map[string]bool, len(to))
	out := make([]string, 0, len(to))
	for _, addr := range to {
		addr = strings.TrimSpace(addr)
		if addr == "" || seen[strings.ToLower(addr)] {
			continue
		}
		seen[strings.ToLower(addr)] = true
		out = append(out, addr)
	}
	return out
}

// buildMessage 组装完整的 RFC 5322 报文（头部 + multipart/alternative 正文）。
// 正文统一使用 base64 编码，规避 SMTP 的点填充与行长限制问题。
func buildMessage(cfg Config, recipients []string, subject, text, html string) []byte {
	var b bytes.Buffer
	writeHeader(&b, "From", formatAddress(cfg.FromName, cfg.FromEmail))
	writeHeader(&b, "To", strings.Join(recipients, ", "))
	writeHeader(&b, "Subject", mime.QEncoding.Encode("UTF-8", subject))
	writeHeader(&b, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&b, "Message-ID", fmt.Sprintf("<%s@%s>", randomHex(12), messageIDHost(cfg.FromEmail)))
	writeHeader(&b, "MIME-Version", "1.0")

	if strings.TrimSpace(html) == "" {
		writeHeader(&b, "Content-Type", `text/plain; charset="UTF-8"`)
		writeHeader(&b, "Content-Transfer-Encoding", "base64")
		b.WriteString("\r\n")
		b.WriteString(wrapBase64(text))
		return b.Bytes()
	}

	boundary := "golog-" + randomHex(16)
	writeHeader(&b, "Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(wrapBase64(text))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(wrapBase64(html))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "--\r\n")
	return b.Bytes()
}

func writeHeader(b *bytes.Buffer, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\r\n")
}

// formatAddress 生成 `显示名 <邮箱>` 形式的地址；显示名按 RFC 2047 编码，
// 以便中文发件人名称在各类邮件客户端中正常显示。
func formatAddress(name, email string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return email
	}
	return mime.QEncoding.Encode("UTF-8", name) + " <" + email + ">"
}

func messageIDHost(email string) string {
	if i := strings.LastIndex(email, "@"); i != -1 && i+1 < len(email) {
		return email[i+1:]
	}
	return "localhost"
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// wrapBase64 按 76 字符折行输出 base64，符合 MIME 行长要求。
func wrapBase64(v string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(v))
	var b strings.Builder
	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString("\r\n")
		encoded = encoded[76:]
	}
	b.WriteString(encoded)
	b.WriteString("\r\n")
	return b.String()
}

// HTMLToText 把一段 HTML 粗略转换为可读的纯文本，用作缺失纯文本部分时的降级内容。
// 它不做完整的 HTML 解析，只处理通知邮件会用到的常见标签与实体。
func HTMLToText(html string) string {
	var out strings.Builder
	skipDepth := 0
	for i := 0; i < len(html); {
		if html[i] == '<' {
			end := strings.IndexByte(html[i:], '>')
			if end == -1 {
				break
			}
			tag := strings.ToLower(strings.TrimSpace(html[i+1 : i+end]))
			i += end + 1
			closing := strings.HasPrefix(tag, "/")
			name := strings.TrimPrefix(tag, "/")
			if j := strings.IndexAny(name, " \t\r\n/"); j != -1 {
				name = name[:j]
			}
			switch name {
			case "script", "style", "head":
				if closing {
					if skipDepth > 0 {
						skipDepth--
					}
				} else {
					skipDepth++
				}
			case "br", "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				out.WriteString("\n")
			}
			continue
		}
		if skipDepth > 0 {
			i++
			continue
		}
		out.WriteByte(html[i])
		i++
	}

	text := out.String()
	replacer := strings.NewReplacer(
		"&nbsp;", " ",
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&#x27;", "'",
	)
	text = replacer.Replace(text)

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if blank {
				continue
			}
			blank = true
			continue
		}
		blank = false
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
