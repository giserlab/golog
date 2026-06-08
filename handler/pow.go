package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golog/system"

	"github.com/gin-gonic/gin"
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
	"/rss.xml":   true,
	"/feed.xml":  true,
	"/sitemap.xml": true,
}

// ─── Core types ───────────────────────────────────────────────────────────────

type PowChallenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
}

// ─── Challenge generation ─────────────────────────────────────────────────────

// generateChallenge creates a random challenge with the given difficulty.
// The challenge includes a timestamp so the server can verify age: base64(random) + "." + unixTimestamp
func generateChallenge(difficulty int) PowChallenge {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate challenge: " + err.Error())
	}
	challenge := base64.RawURLEncoding.EncodeToString(b) + "." +
		strconv.FormatInt(time.Now().Unix(), 10)
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

// issueCookie creates an HMAC-signed cookie value for a verified challenge.
// Format: base64(challenge) + "." + base64(nonce) + "." + base64(hmac_signature)
func issueCookie(challenge, nonce string) string {
	payload := challenge + ":" + nonce
	sig := hmacSign(payload)
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
	expectedSig := hmacSign(payload)
	if !hmac.Equal([]byte(expectedSig), []byte(givenSig)) {
		return false
	}

	// Verify the challenge is not too old (within PoW TTL window)
	// The challenge itself contains a timestamp appended by generateChallenge.
	// Parse it out: format is base64(random) + "." + unixTimestamp
	if lastDot := strings.LastIndex(challenge, "."); lastDot > 0 {
		tsStr := challenge[lastDot+1:]
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err == nil {
			maxAge := system.Config.PoWTTL
			if maxAge <= 0 {
				maxAge = 24
			}
			if time.Now().Unix()-ts > int64(maxAge)*3600 {
				return false // expired
			}
		}
	}

	return true
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
		if strings.HasPrefix(path, prefix) {
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
	c.Redirect(http.StatusFound, "/pow?redirect="+redirect)
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
		redirect := c.DefaultQuery("redirect", "/")
		c.Redirect(http.StatusFound, redirect)
		return
	}

	redirect := c.DefaultQuery("redirect", "/")

	// If they already have a valid cookie, skip the challenge
	cookie, err := c.Cookie(powCookieName)
	if err == nil && verifyCookie(cookie) {
		c.Redirect(http.StatusFound, redirect)
		return
	}

	difficulty := system.Config.PoWDifficulty
	if difficulty < 1 {
		difficulty = 20
	}
	challenge := generateChallenge(difficulty)

	var tpl bytes.Buffer
	if err := system.PowTmpl.Execute(&tpl, data(c, gin.H{
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
	Challenge string `form:"challenge" binding:"required"`
	Nonce     string `form:"nonce" binding:"required"`
	Redirect  string `form:"redirect"`
}

// PowSolve verifies the submitted solution and issues a PoW cookie.
func PowSolve(c *gin.Context, req *PowSolveRequest) {
	if system.Config == nil || !system.Config.PoWEnabled {
		c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
		return
	}

	difficulty := system.Config.PoWDifficulty
	if difficulty < 1 {
		difficulty = 20
	}

	if !verifySolution(req.Challenge, req.Nonce, difficulty) {
		// Verification failed — redirect back to the challenge page
		c.Redirect(http.StatusFound, "/pow?redirect="+req.Redirect)
		return
	}

	// Issue the PoW cookie
	ttl := system.Config.PoWTTL
	if ttl <= 0 {
		ttl = 24
	}
	c.SetCookie(powCookieName, issueCookie(req.Challenge, req.Nonce),
		ttl*3600, "/", "", false, true)

	c.Redirect(http.StatusFound, safeRedirect(req.Redirect))
}

// safeRedirect prevents open redirect vulnerabilities.
func safeRedirect(url string) string {
	if url == "" || url[0] != '/' {
		return "/"
	}
	// Block protocol-relative and external URLs
	if strings.HasPrefix(url, "//") || strings.Contains(url, "://") {
		return "/"
	}
	return url
}
