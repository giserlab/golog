package mailer

import (
	"encoding/base64"
	"strings"
	"testing"
)

// TestConfigNormalize 覆盖发信配置的缺省值补齐。
func TestConfigNormalize(t *testing.T) {
	got := Config{Username: "bot@example.com", Host: "smtp.example.com"}.Normalize()
	if got.Port != 587 {
		t.Fatalf("Port = %d, want 587", got.Port)
	}
	if got.Encryption != EncryptionStartTLS {
		t.Fatalf("Encryption = %q, want starttls", got.Encryption)
	}
	if got.FromEmail != "bot@example.com" {
		t.Fatalf("FromEmail = %q, want the SMTP username", got.FromEmail)
	}
}

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok", Config{Host: "smtp.example.com", Port: 465, Encryption: EncryptionSSL, FromEmail: "a@b.c"}, false},
		{"missing host", Config{FromEmail: "a@b.c"}, true},
		{"bad port", Config{Host: "h", Port: 70000, FromEmail: "a@b.c"}, true},
		{"missing sender", Config{Host: "h", Port: 25, Encryption: EncryptionNone}, true},
		{"bad encryption", Config{Host: "h", Port: 25, Encryption: "pgp", FromEmail: "a@b.c"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestBuildMessageMultipart 断言邮件是 multipart/alternative，且两个部分
// 都是 base64 编码的 UTF-8 正文，中文主题按 RFC 2047 编码。
func TestBuildMessageMultipart(t *testing.T) {
	cfg := Config{Host: "smtp.example.com", FromName: "站点", FromEmail: "noreply@example.com"}
	raw := string(buildMessage(cfg, []string{"to@example.com"}, "新留言：标题", "纯文本正文", "<p>HTML 正文</p>"))

	for _, want := range []string{
		"From: =?UTF-8?",
		"To: to@example.com",
		"MIME-Version: 1.0",
		"multipart/alternative; boundary=",
		`Content-Type: text/plain; charset="UTF-8"`,
		`Content-Type: text/html; charset="UTF-8"`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("message missing %q:\n%s", want, raw)
		}
	}
	if strings.Contains(raw, "Subject: 新留言") {
		t.Fatalf("subject must be RFC 2047 encoded, got:\n%s", raw)
	}

	boundary := boundaryFrom(raw)
	if boundary == "" {
		t.Fatal("could not extract boundary")
	}
	parts := strings.Split(raw, "--"+boundary)
	if len(parts) < 4 {
		t.Fatalf("expected 2 MIME parts plus terminator, got %d", len(parts))
	}
	for i, want := range []string{"纯文本正文", "<p>HTML 正文</p>"} {
		if got := decodeBase64Part(parts[i+1]); got != want {
			t.Fatalf("part %d = %q, want %q", i+1, got, want)
		}
	}
}

func TestBuildMessagePlainTextOnly(t *testing.T) {
	cfg := Config{Host: "h", FromEmail: "a@b.c"}
	raw := string(buildMessage(cfg, []string{"to@example.com"}, "hi", "just text", ""))
	if !strings.Contains(raw, `Content-Type: text/plain; charset="UTF-8"`) {
		t.Fatalf("expected a plain text message, got:\n%s", raw)
	}
	if strings.Contains(raw, "multipart/alternative") {
		t.Fatalf("plain text message must not be multipart:\n%s", raw)
	}
}

func TestHTMLToText(t *testing.T) {
	got := HTMLToText(`<style>p{color:red}</style><p>第一行</p><p>第二行&amp;更多<br>第三行</p>`)
	want := "第一行\n第二行&更多\n第三行"
	if got != want {
		t.Fatalf("HTMLToText = %q, want %q", got, want)
	}
}

func boundaryFrom(raw string) string {
	const marker = `boundary="`
	i := strings.Index(raw, marker)
	if i == -1 {
		return ""
	}
	rest := raw[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j == -1 {
		return ""
	}
	return rest[:j]
}

func decodeBase64Part(part string) string {
	_, body, found := strings.Cut(part, "\r\n\r\n")
	if !found {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(body, "\r\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			break
		}
		b.WriteString(line)
	}
	decoded, err := base64.StdEncoding.DecodeString(b.String())
	if err != nil {
		return ""
	}
	return string(decoded)
}
