package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	csrf "github.com/utrack/gin-csrf"
)

// powSecretKey is an ephemeral HMAC key regenerated on every server restart.
// Cookies from a previous process instance are automatically invalidated.
var powSecretKey string

func init() {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate PoW secret key: " + err.Error())
	}
	powSecretKey = base64.RawURLEncoding.EncodeToString(b)
}

// ─── Cookie name ──────────────────────────────────────────────────────────────

const powCookieName = "golog_pow"

const (
	powChallengeVersion      = "v1"
	powChallengeSolveTTL     = 15 * time.Minute
	powChallengeMaxClockSkew = 5 * time.Minute
	powMaxChallengeLength    = 256
	powMaxNonceLength        = 20
)

// ─── Routes excluded from PoW ─────────────────────────────────────────────────

var powExcludedPrefixes = []string{
	"/admin",
	"/wizard",
	"/login",
	"/pow",
	"/uploads",
	"/assets",
}

// powExcludedPaths are exact paths excluded from PoW.
var powExcludedPaths = map[string]bool{
	"/rss.xml":     true,
	"/feed.xml":    true,
	"/sitemap.xml": true,
}

// ─── Core types ───────────────────────────────────────────────────────────────

type PowChallenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
}

// ─── Challenge generation ─────────────────────────────────────────────────────

// generateChallenge creates a signed random challenge with the given
// difficulty and issue timestamp.
func generateChallenge(difficulty int) PowChallenge {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate challenge: " + err.Error())
	}
	payload := strings.Join([]string{
		powChallengeVersion,
		base64.RawURLEncoding.EncodeToString(b),
		strconv.FormatInt(time.Now().Unix(), 10),
		strconv.Itoa(difficulty),
	}, ".")
	challenge := payload + "." + signChallengePayload(payload)
	return PowChallenge{
		Challenge:  challenge,
		Difficulty: difficulty,
	}
}

// ─── Solution verification ────────────────────────────────────────────────────

// verifySolution checks that SHA256(challenge + ":" + nonce) has at least
// difficulty leading zero bits.
func verifySolution(challenge, nonce string, difficulty int) bool {
	if difficulty < 1 {
		return true // zero difficulty trivially passes
	}
	data := challenge + ":" + nonce
	sum := sha256.Sum256([]byte(data))

	// Count leading zero bits
	leadingZeros := 0
	for _, b := range sum {
		if b == 0 {
			leadingZeros += 8
			continue
		}
		// Count leading zero bits in this byte
		mask := byte(0x80)
		for mask > 0 {
			if b&mask == 0 {
				leadingZeros++
				mask >>= 1
			} else {
				break
			}
		}
		break
	}
	return leadingZeros >= difficulty
}

// ─── Cookie management ────────────────────────────────────────────────────────

// hmacSign returns HMAC-SHA256(message) encoded in base64.
func hmacSign(message string) string {
	mac := hmac.New(sha256.New, []byte(powSecretKey))
	io.WriteString(mac, message)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func signChallengePayload(payload string) string {
	return hmacSign("pow-challenge:" + payload)
}

func signCookiePayload(payload string) string {
	return hmacSign("pow-cookie:" + payload)
}

func configuredPoWDifficulty() int {
	if system.Config == nil {
		return 20
	}
	difficulty := system.Config.PoWDifficulty
	if difficulty < 1 {
		difficulty = 20
	}
	return difficulty
}

func configuredPoWTTL() time.Duration {
	if system.Config == nil {
		return 24 * time.Hour
	}
	ttl := system.Config.PoWTTL
	if ttl <= 0 {
		ttl = 24
	}
	return time.Duration(ttl) * time.Hour
}

func validateChallenge(challenge string, expectedDifficulty int, maxAge time.Duration) bool {
	parts := strings.Split(challenge, ".")
	if len(parts) != 5 || parts[0] != powChallengeVersion {
		return false
	}

	randomB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(randomB) != 16 {
		return false
	}

	issuedAt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return false
	}

	challengeDifficulty, err := strconv.Atoi(parts[3])
	if err != nil || challengeDifficulty != expectedDifficulty {
		return false
	}

	payload := strings.Join(parts[:4], ".")
	expectedSig := signChallengePayload(payload)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[4])) {
		return false
	}

	issued := time.Unix(issuedAt, 0)
	now := time.Now()
	if issued.After(now.Add(powChallengeMaxClockSkew)) {
		return false
	}
	if now.Sub(issued) > maxAge {
		return false
	}
	return true
}

func matchesPoWExcludedPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func validNonce(nonce string) bool {
	if nonce == "" || len(nonce) > powMaxNonceLength {
		return false
	}
	for _, r := range nonce {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func powRedirectURL(redirect string) string {
	return "/pow?redirect=" + urlpkg.QueryEscape(safeRedirect(redirect))
}

// issueCookie creates an HMAC-signed cookie value for a verified challenge.
// Format: base64(challenge) + "." + base64(nonce) + "." + base64(hmac_signature)
func issueCookie(challenge, nonce string) string {
	payload := challenge + ":" + nonce
	sig := signCookiePayload(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(challenge)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(nonce)) + "." +
		sig
}

// verifyCookie parses and validates an HMAC-signed PoW cookie.
// Returns true if the cookie is valid and not expired.
func verifyCookie(cookieValue string) bool {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 3 {
		return false
	}

	challengeB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	nonceB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	challenge := string(challengeB)
	nonce := string(nonceB)
	givenSig := parts[2]

	// Recompute the expected signature and compare
	payload := challenge + ":" + nonce
	expectedSig := signCookiePayload(payload)
	if !hmac.Equal([]byte(expectedSig), []byte(givenSig)) {
		return false
	}

	return validateChallenge(challenge, configuredPoWDifficulty(), configuredPoWTTL())
}

// ─── Middleware ────────────────────────────────────────────────────────────────

// powMiddleware checks whether the visitor has a valid PoW cookie.
// If PoW is disabled in config, the check is skipped entirely.
func powMiddleware(c *gin.Context) {
	// Skip if PoW is disabled
	if system.Config == nil || !system.Config.PoWEnabled {
		c.Next()
		return
	}

	// Skip excluded routes
	path := c.Request.URL.Path
	for _, prefix := range powExcludedPrefixes {
		if matchesPoWExcludedPrefix(path, prefix) {
			c.Next()
			return
		}
	}
	if powExcludedPaths[path] {
		c.Next()
		return
	}

	// Check for a valid PoW cookie
	cookie, err := c.Cookie(powCookieName)
	if err == nil && verifyCookie(cookie) {
		c.Next()
		return
	}

	// Redirect to PoW challenge page
	redirect := c.Request.URL.RequestURI()
	if redirect == "" {
		redirect = "/"
	}
	c.Redirect(http.StatusFound, powRedirectURL(redirect))
	c.Abort()
}

// ─── Page handlers ────────────────────────────────────────────────────────────

// PowPage renders the PoW challenge page. If the visitor already has a valid
// cookie, it redirects them to the intended destination immediately.
func PowPage(c *gin.Context) {
	if system.Config == nil {
		c.Redirect(http.StatusFound, "/wizard")
		return
	}

	// If PoW is disabled, redirect to the intended page or home
	if !system.Config.PoWEnabled {
		redirect := safeRedirect(c.DefaultQuery("redirect", "/"))
		c.Redirect(http.StatusFound, redirect)
		return
	}

	redirect := safeRedirect(c.DefaultQuery("redirect", "/"))

	// If they already have a valid cookie, skip the challenge
	cookie, err := c.Cookie(powCookieName)
	if err == nil && verifyCookie(cookie) {
		c.Redirect(http.StatusFound, redirect)
		return
	}

	difficulty := configuredPoWDifficulty()
	challenge := generateChallenge(difficulty)
	var routes = []entity.Route{}
	routes = append(routes, entity.Route{
		Name: "人机验证",
		Path: "",
	})
	var tpl bytes.Buffer
	if err := system.PowTmpl.Execute(&tpl, powData(c, gin.H{
		"Routes":        routes,
		"PowChallenge":  challenge.Challenge,
		"PowDifficulty": challenge.Difficulty,
		"PowRedirect":   redirect,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

// ─── Solve handler ────────────────────────────────────────────────────────────

// PowSolveRequest is the form payload submitted by the PoW solver.
type PowSolveRequest struct {
	Challenge string `form:"challenge" binding:"required,max=256"`
	Nonce     string `form:"nonce" binding:"required,max=20"`
	Redirect  string `form:"redirect"`
}

// PowSolve verifies the submitted solution and issues a PoW cookie.
func PowSolve(c *gin.Context, req *PowSolveRequest) {
	if system.Config == nil || !system.Config.PoWEnabled {
		c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
		return
	}

	difficulty := configuredPoWDifficulty()

	if len(req.Challenge) > powMaxChallengeLength ||
		!validNonce(req.Nonce) ||
		!validateChallenge(req.Challenge, difficulty, powChallengeSolveTTL) ||
		!verifySolution(req.Challenge, req.Nonce, difficulty) {
		// Verification failed — redirect back to the challenge page
		c.Redirect(http.StatusFound, powRedirectURL(req.Redirect))
		return
	}

	// Issue the PoW cookie
	c.SetCookie(powCookieName, issueCookie(req.Challenge, req.Nonce),
		int(configuredPoWTTL().Seconds()), "/", "", false, true)

	c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
}

func powData(c *gin.Context, data gin.H) gin.H {
	suffix := "https://"
	if c.Request.TLS == nil {
		suffix = "http://"
	}
	fullPath := c.FullPath()
	relativeRoot := entity.RelativeRoots[fullPath]
	stats := map[[2]string]int{}
	momentStats := map[string]int{}
	tagMap := map[[2]string]int{}
	data["QUID"] = uuid.New().String()
	data["Self"] = nil
	data["Stats"] = &stats
	data["MomentStats"] = &momentStats
	data["TagMap"] = &tagMap
	data["BlogTypes"] = map[string]string{}
	data["Config"] = system.Config
	data["Message"] = message(c)
	data["CSRF"] = csrf.GetToken(c)
	data["URL"] = map[string]string{
		"Root":         filepath.Clean(suffix + c.Request.Host + c.Request.URL.Path + relativeRoot),
		"Absolute":     suffix + c.Request.Host + c.Request.URL.Path,
		"RelativeRoot": relativeRoot,
		"AbsoluteHost": suffix + c.Request.Host + "/",
		"PageType":     entity.PageTypes[fullPath],
	}
	return data
}

// safeRedirect prevents open redirect vulnerabilities.
func safeRedirect(rawURL string) string {
	if rawURL == "" || rawURL[0] != '/' {
		return "/"
	}
	parsed, err := urlpkg.Parse(rawURL)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/"
	}
	return rawURL
}
