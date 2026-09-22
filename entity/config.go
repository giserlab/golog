package entity

type ColorScheme string

const (
	ColorSchemeLight ColorScheme = "light"
	ColorSchemeDark  ColorScheme = "dark"
	ColorSchemeAuto  ColorScheme = ""
)

type FontFamily string

const (
	FontFamilyNotoSans  FontFamily = "sans"
	FontFamilyNotoSerif FontFamily = "serif"
)

type AuthorBlock string

const (
	AuthorBlockNone  AuthorBlock = "none"
	AuthorBlockStart AuthorBlock = "start"
	AuthorBlockEnd   AuthorBlock = "end"
)

// MailEncryption 是 SMTP 连接的加密方式。
type MailEncryption string

const (
	MailEncryptionNone     MailEncryption = "none"     // 明文（通常在受信内网或本地中继使用）
	MailEncryptionSSL      MailEncryption = "ssl"      // 隐式 TLS（SMTPS，常见端口 465）
	MailEncryptionStartTLS MailEncryption = "starttls" // 显式 TLS（常见端口 587）
)

type Config struct {
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	About             string      `json:"about"`
	IsPublic          bool        `json:"is_public"`
	DateFormat        string      `json:"date_format"`
	TimeFormat        string      `json:"time_format"`
	Timezone          int         `json:"timezone"`
	InjectedHead      string      `json:"injected_head"`
	InjectedFoot      string      `json:"injected_foot"`
	InjectedPostStart string      `json:"injected_post_start"`
	InjectedPostEnd   string      `json:"injected_post_end"`
	FooterText        string      `json:"footer_text"`
	ColorScheme       ColorScheme `json:"color_scheme"`
	ContainerWidth    string      `json:"container_width"`
	FontFamily        FontFamily  `json:"font_family"`
	FontSize          string      `json:"font_size"`
	HighlightJS       bool        `json:"highlight_js"`
	AuthorBlock       AuthorBlock `json:"author_block"`
	PostsPerPage      int         `json:"posts_per_page"`
	Theme             string      `json:"theme"`
	Locale            string      `json:"locale"`
	Favicon           string      `json:"favicon"`
	CustomCSS         string      `json:"custom_css"`
	APIKey            string      `json:"api_key"`
	WebAuthnRPID      string      `json:"webauthn_rp_id"`
	WebAuthnOrigins   []string    `json:"webauthn_origins"`

	PoWEnabled       bool     `json:"pow_enabled"`
	PoWMaxNumber     int64    `json:"pow_max_number"`
	PoWTTL           int      `json:"pow_ttl"`
	PoWHMACKey       string   `json:"pow_hmac_key"`
	PoWBotBypass     bool     `json:"pow_bot_bypass"`
	PoWBotUserAgents []string `json:"pow_bot_user_agents"`

	CommentsEnabled bool `json:"comments_enabled"`

	// ── 留言邮件通知（SMTP） ──────────────────────────────────────────────
	// MailEnabled 是总开关：关闭时不建立任何 SMTP 连接。
	MailEnabled    bool           `json:"mail_enabled"`
	MailHost       string         `json:"mail_host"`
	MailPort       int            `json:"mail_port"`
	MailUsername   string         `json:"mail_username"`
	MailPassword   string         `json:"mail_password"`
	MailEncryption MailEncryption `json:"mail_encryption"`
	MailFromName   string         `json:"mail_from_name"`
	MailFromEmail  string         `json:"mail_from_email"`
	// MailAdminEmail 是额外的收件人（例如站点管理员），留空则只通知文章作者。
	MailAdminEmail string `json:"mail_admin_email"`
	// MailNotifyAuthor 控制新留言/新回复提交时是否通知文章作者及管理员。
	MailNotifyAuthor bool `json:"mail_notify_author"`
	// MailNotifyReply 控制回复通过审核后是否通知“被回复留言”的作者。
	MailNotifyReply bool `json:"mail_notify_reply"`
	// 自定义邮件模板（Go template 语法）；留空时使用内置默认模板。
	MailAuthorSubject string `json:"mail_author_subject"`
	MailAuthorBody    string `json:"mail_author_body"`
	MailReplySubject  string `json:"mail_reply_subject"`
	MailReplyBody     string `json:"mail_reply_body"`
}

// MailSenderEmail 返回发件人地址：优先使用显式配置，缺省回退到 SMTP 用户名。
func (c *Config) MailSenderEmail() string {
	if c.MailFromEmail != "" {
		return c.MailFromEmail
	}
	return c.MailUsername
}

// MailSenderName 返回发件人显示名，缺省回退到站点名称。
func (c *Config) MailSenderName() string {
	if c.MailFromName != "" {
		return c.MailFromName
	}
	return c.Name
}

// MailPortOrDefault 返回 SMTP 端口，未配置时使用 587。
func (c *Config) MailPortOrDefault() int {
	if c.MailPort > 0 {
		return c.MailPort
	}
	return 587
}

// MailEncryptionOrDefault 返回 SMTP 加密方式，未配置时使用 STARTTLS。
func (c *Config) MailEncryptionOrDefault() MailEncryption {
	switch c.MailEncryption {
	case MailEncryptionNone, MailEncryptionSSL, MailEncryptionStartTLS:
		return c.MailEncryption
	default:
		return MailEncryptionStartTLS
	}
}

// MailConfigured 报告邮件通知是否已具备发送条件（总开关、服务器与发件人）。
func (c *Config) MailConfigured() bool {
	return c.MailEnabled && c.MailHost != "" && c.MailSenderEmail() != ""
}

func (c *Config) IsCustomTimeFormat() bool {
	return c.TimeFormat != "PM 03:04" && c.TimeFormat != "15:04" && c.TimeFormat != "03:04 PM"
}

func (c *Config) IsCustomDateFormat() bool {
	return c.DateFormat != "2006-01-02" && c.DateFormat != "01/02/2006" && c.DateFormat != "02/01/2006"
}
