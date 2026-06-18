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
	"regexp"
	"strconv"
	"strings"
	"time"

	"golog/entity"
	"golog/system"

	"github.com/altcha-org/altcha-lib-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	csrf "github.com/utrack/gin-csrf"
)

// powCookieName is the name of the long-lived verification cookie.
const powCookieName = "golog_pow"

const (
	powCookieVersion = "v1"
)

// Routes excluded from PoW verification.
var powExcludedPrefixes = []string{
	"/admin",
	"/wizard",
	"/login",
	"/pow",
	"/uploads",
	"/assets",
}

var powExcludedPaths = map[string]bool{
	"/rss.xml":          true,
	"/feed.xml":         true,
	"/sitemap.xml":      true,
	"/altcha/challenge": true,
}

// defaultPowBotUserAgents lists common search-engine crawlers that may be
// allowed to bypass PoW verification when PoWBotBypass is enabled.
var defaultPowBotUserAgents = []string{
	"googlebot",
	"bingbot",
	"slurp",
	"duckduckbot",
	"baiduspider",
	"yandexbot",
	"sogou",
	"applebot",
	"bytespider",
}

// isPowBotRequest reports whether the request User-Agent matches a configured
// search-engine crawler and bot bypass is enabled.
func isPowBotRequest(c *gin.Context) bool {
	if system.Config == nil || !system.Config.PoWBotBypass {
		return false
	}
	ua := strings.ToLower(c.Request.UserAgent())
	if ua == "" {
		return false
	}
	agents := system.Config.PoWBotUserAgents
	if len(agents) == 0 {
		agents = defaultPowBotUserAgents
	}
	for _, agent := range agents {
		if matchBotAgent(ua, agent) {
			return true
		}
	}
	return false
}

// matchBotAgent matches a lowercased user-agent against a pattern.
// Plain patterns perform substring matching (backwards compatible with the
// default list). Patterns containing * or ? are treated as shell-style
// wildcards against the full user-agent string:
//   * matches any sequence of characters, ? matches any single character.
func matchBotAgent(ua, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		return false
	}
	if strings.ContainsAny(pattern, "*?") {
		rePattern := regexp.QuoteMeta(pattern)
		rePattern = strings.ReplaceAll(rePattern, "\\*", ".*")
		rePattern = strings.ReplaceAll(rePattern, "\\?", ".")
		return regexp.MustCompile("^" + rePattern + "$").MatchString(ua)
	}
	return strings.Contains(ua, pattern)
}

// powHMACKey returns the configured ALTCHA HMAC key, generating one if missing.
func powHMACKey() string {
	if system.Config != nil && system.Config.PoWHMACKey != "" {
		return system.Config.PoWHMACKey
	}
	return ""
}

// powMaxNumber returns the configured ALTCHA MaxNumber with a safe default.
func powMaxNumber() int64 {
	if system.Config != nil && system.Config.PoWMaxNumber > 0 {
		return system.Config.PoWMaxNumber
	}
	return 200000
}

// powTTL returns the configured verification cookie TTL.
func powTTL() time.Duration {
	if system.Config != nil && system.Config.PoWTTL > 0 {
		return time.Duration(system.Config.PoWTTL) * time.Hour
	}
	return 24 * time.Hour
}

// AltchaChallenge serves a fresh ALTCHA challenge to the widget.
func AltchaChallenge(c *gin.Context) {
	key := powHMACKey()
	if key == "" {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	expires := time.Now().Add(15 * time.Minute)
	challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
		HMACKey:   key,
		MaxNumber: powMaxNumber(),
		Expires:   &expires,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, challenge)
}

// PowPage renders the ALTCHA verification page.
func PowPage(c *gin.Context) {
	if system.Config == nil {
		c.Redirect(http.StatusFound, "/wizard")
		return
	}

	if !system.Config.PoWEnabled {
		redirect := safeRedirect(c.DefaultQuery("redirect", "/"))
		c.Redirect(http.StatusFound, redirect)
		return
	}

	redirect := safeRedirect(c.DefaultQuery("redirect", "/"))

	// Already verified?
	if hasValidPowCookie(c) {
		c.Redirect(http.StatusFound, redirect)
		return
	}

	var routes = []entity.Route{{
		Name: "人机验证",
		Path: "",
	}}
	var tpl bytes.Buffer
	if err := system.PowTmpl.Execute(&tpl, powData(c, gin.H{
		"Routes":      routes,
		"PowRedirect": redirect,
	})); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
}

// PowSolveRequest is the form payload submitted by the ALTCHA widget.
type PowSolveRequest struct {
	Altcha   string `form:"altcha" binding:"required"`
	Redirect string `form:"redirect"`
}

// PowSolve verifies the ALTCHA payload and issues a long-lived cookie.
func PowSolve(c *gin.Context, req *PowSolveRequest) {
	if system.Config == nil || !system.Config.PoWEnabled {
		c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
		return
	}

	key := powHMACKey()
	if key == "" {
		c.Redirect(http.StatusFound, powRedirectURL(req.Redirect))
		return
	}

	ok, err := altcha.VerifySolution(req.Altcha, key, true)
	if err != nil || !ok {
		c.Redirect(http.StatusFound, powRedirectURL(req.Redirect))
		return
	}

	issuePowCookie(c)
	c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
}

// powMiddleware checks whether the visitor has a valid verification cookie.
func powMiddleware(c *gin.Context) {
	if system.Config == nil || !system.Config.PoWEnabled {
		c.Next()
		return
	}

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

	if isPowBotRequest(c) {
		c.Next()
		return
	}

	if hasValidPowCookie(c) {
		c.Next()
		return
	}

	redirect := c.Request.URL.RequestURI()
	if redirect == "" {
		redirect = "/"
	}
	c.Redirect(http.StatusFound, powRedirectURL(redirect))
	c.Abort()
}

// hasValidPowCookie reports whether the request carries a valid verification cookie.
func hasValidPowCookie(c *gin.Context) bool {
	cookie, err := c.Cookie(powCookieName)
	if err != nil {
		return false
	}
	return verifyPowCookie(cookie)
}

// issuePowCookie sets the long-lived verification cookie.
func issuePowCookie(c *gin.Context) {
	expiresAt := time.Now().Add(powTTL())
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		// Fall back to uuid if rand fails.
		random = []byte(uuid.New().String())
	}
	payload := strings.Join([]string{
		powCookieVersion,
		strconv.FormatInt(expiresAt.Unix(), 10),
		base64.RawURLEncoding.EncodeToString(random),
	}, ".")
	sig := signPowCookiePayload(payload)
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig

	c.SetCookie(powCookieName, value,
		int(powTTL().Seconds()), "/", "", false, true)
}

// verifyPowCookie validates the HMAC-signed verification cookie and expiry.
func verifyPowCookie(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payload := string(payloadB)
	expectedSig := signPowCookiePayload(payload)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[1])) {
		return false
	}

	pparts := strings.Split(payload, ".")
	if len(pparts) != 3 || pparts[0] != powCookieVersion {
		return false
	}
	expiresAt, err := strconv.ParseInt(pparts[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expiresAt
}

// signPowCookiePayload returns the HMAC-SHA256 signature of the cookie payload.
func signPowCookiePayload(payload string) string {
	mac := hmac.New(sha256.New, []byte(powCookieSecret()))
	io.WriteString(mac, payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// powCookieSecret derives the cookie signing secret from the ALTCHA HMAC key.
func powCookieSecret() string {
	if system.Config != nil && system.Config.PoWHMACKey != "" {
		sum := sha256.Sum256([]byte("golog-pow-cookie:" + system.Config.PoWHMACKey))
		return base64.RawURLEncoding.EncodeToString(sum[:])
	}
	return ""
}

func matchesPoWExcludedPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func powRedirectURL(redirect string) string {
	return "/pow?redirect=" + urlpkg.QueryEscape(safeRedirect(redirect))
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
