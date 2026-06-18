package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golog/entity"
	"golog/system"

	"github.com/altcha-org/altcha-lib-go"
	"github.com/gin-gonic/gin"
)

func withAltchaConfig(t *testing.T) {
	t.Helper()
	oldConfig := system.Config
	system.Config = &entity.Config{
		IsPublic:     true,
		PoWEnabled:   true,
		PoWMaxNumber: 5000,
		PoWTTL:       24,
		PoWHMACKey:   "test-hmac-key-abcdefghijklmnopqrstuvwxyz",
	}
	t.Cleanup(func() {
		system.Config = oldConfig
	})
}

func TestAltchaChallengeReturnsValidChallenge(t *testing.T) {
	withAltchaConfig(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/altcha/challenge", nil)
	Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var ch altcha.Challenge
	if err := json.Unmarshal(w.Body.Bytes(), &ch); err != nil {
		t.Fatalf("invalid challenge JSON: %v", err)
	}
	if ch.Algorithm == "" || ch.Challenge == "" || ch.Salt == "" || ch.Signature == "" {
		t.Fatal("challenge fields missing")
	}
}

func TestPowSolveAcceptsValidPayload(t *testing.T) {
	withAltchaConfig(t)

	challenge := createTestChallenge(t)
	solution := solveTestChallenge(t, challenge)
	payload := buildTestPayload(challenge, solution)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pow/solve", nil)
	c := nilContext(w, req)
	PowSolve(c, &PowSolveRequest{
		Altcha:   payload,
		Redirect: "/",
	})

	// When calling the handler directly via a test context gin writes the
	// Location header but does not flush the status code, so we assert the
	// header instead of the response code.
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Fatalf("Location = %q, want %q", loc, "/")
	}
	if !hasPowCookie(w) {
		t.Fatal("expected pow cookie to be issued")
	}
}

func TestPowSolveRejectsInvalidPayload(t *testing.T) {
	withAltchaConfig(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pow/solve", nil)
	c := nilContext(w, req)
	PowSolve(c, &PowSolveRequest{
		Altcha:   "invalid-payload",
		Redirect: "/",
	})

	if loc := w.Header().Get("Location"); loc != "/pow?redirect=%2F" {
		t.Fatalf("Location = %q, want %q", loc, "/pow?redirect=%2F")
	}
	if hasPowCookie(w) {
		t.Fatal("unexpected pow cookie issued for invalid payload")
	}
}

func TestPowMiddlewareRequiresVerification(t *testing.T) {
	withAltchaConfig(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	Router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if got, want := w.Header().Get("Location"), "/pow?redirect=%2Fmissing-page"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestPowMiddlewareAllowsVerifiedCookie(t *testing.T) {
	withAltchaConfig(t)

	// Issue a verification cookie directly.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	c1 := nilContext(w1, req1)
	issuePowCookie(c1)
	cookie := findPowCookie(w1)
	if cookie == "" {
		t.Fatal("failed to issue pow cookie")
	}

	// Use the cookie to access a protected route.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	req2.Header.Set("Cookie", powCookieName+"="+cookie)
	Router.ServeHTTP(w2, req2)

	// NoRoute handler returns 404 for missing pages when the middleware passes.
	if w2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusNotFound)
	}
}

func TestPowMiddlewareKeepsExcludedRoutesOpen(t *testing.T) {
	withAltchaConfig(t)

	router := gin.New()
	router.Use(powMiddleware)
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, path := range []string{
		"/admin",
		"/admin/assets/app.css",
		"/wizard",
		"/login",
		"/pow",
		"/altcha/challenge",
		"/uploads/photo.jpg",
		"/assets/pow.css",
		"/rss.xml",
		"/feed.xml",
		"/sitemap.xml",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)

			if got := w.Code; got != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", got, http.StatusNoContent)
			}
		})
	}
}

func TestPowCookieRejectsTampering(t *testing.T) {
	withAltchaConfig(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	req.Header.Set("Cookie", powCookieName+"=tampered.value")
	Router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPowCookieExpires(t *testing.T) {
	withAltchaConfig(t)

	value := issueExpiredPowCookie()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing-page", nil)
	req.Header.Set("Cookie", powCookieName+"="+value)
	Router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestPowRedirectURLSanitizesRedirect(t *testing.T) {
	if got, want := powRedirectURL("/post/test?x=1"), "/pow?redirect=%2Fpost%2Ftest%3Fx%3D1"; got != want {
		t.Fatalf("powRedirectURL = %q, want %q", got, want)
	}
	if got, want := powRedirectURL("/post/test?next=https://example.com"), "/pow?redirect=%2Fpost%2Ftest%3Fnext%3Dhttps%3A%2F%2Fexample.com"; got != want {
		t.Fatalf("powRedirectURL = %q, want %q", got, want)
	}
	if got, want := powRedirectURL("//example.com"), "/pow?redirect=%2F"; got != want {
		t.Fatalf("powRedirectURL = %q, want %q", got, want)
	}
	if got, want := powRedirectURL("https://example.com"), "/pow?redirect=%2F"; got != want {
		t.Fatalf("powRedirectURL = %q, want %q", got, want)
	}
}

func TestMatchBotAgent(t *testing.T) {
	cases := []struct {
		ua      string
		pattern string
		want    bool
	}{
		// plain substring matching (default behavior)
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "googlebot", true},
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "bingbot", false},
		// wildcard matching against the full UA
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "*bot*", true},
		{"Mozilla/5.0 (compatible; Bingbot/2.1)", "*bot*", true},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "*bot*", false},
		{"SomeBot", "*bot", true},
		{"SomeBot/1.0", "*bot*", true},
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "*googlebot*", true},
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "*?ooglebot*", true},
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "mozilla/5.0 *googlebot*", true},
		// empty pattern never matches
		{"Mozilla/5.0", "", false},
		// case insensitivity
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "GOOGLEBOT", true},
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", "*GOOGLEBOT*", true},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+"_"+tc.ua, func(t *testing.T) {
			if got := matchBotAgent(strings.ToLower(tc.ua), tc.pattern); got != tc.want {
				t.Fatalf("matchBotAgent(%q, %q) = %v, want %v", tc.ua, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestPowMiddlewareAllowsConfiguredBots(t *testing.T) {
	withAltchaConfig(t)
	system.Config.PoWBotBypass = true

	router := gin.New()
	router.Use(powMiddleware)
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	cases := []struct {
		ua   string
		want int
	}{
		{"Mozilla/5.0 (compatible; Googlebot/2.1)", http.StatusNoContent},
		{"Mozilla/5.0 (compatible; Bingbot/2.1)", http.StatusNoContent},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", http.StatusFound},
	}
	for _, tc := range cases {
		t.Run(tc.ua, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/some-page", nil)
			req.Header.Set("User-Agent", tc.ua)
			router.ServeHTTP(w, req)
			if got := w.Code; got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func createTestChallenge(t *testing.T) altcha.Challenge {
	t.Helper()
	expires := time.Now().Add(15 * time.Minute)
	challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
		HMACKey:   system.Config.PoWHMACKey,
		MaxNumber: system.Config.PoWMaxNumber,
		Expires:   &expires,
	})
	if err != nil {
		t.Fatalf("create challenge: %v", err)
	}
	return challenge
}

func solveTestChallenge(t *testing.T, challenge altcha.Challenge) int {
	t.Helper()
	stop := make(chan struct{})
	solution, err := altcha.SolveChallenge(challenge.Challenge, challenge.Salt, altcha.Algorithm(challenge.Algorithm), int(challenge.MaxNumber), 0, stop)
	if err != nil {
		t.Fatalf("solve challenge: %v", err)
	}
	if solution == nil {
		t.Fatal("no solution found")
	}
	return solution.Number
}

func buildTestPayload(challenge altcha.Challenge, number int) string {
	p := altcha.Payload{
		Algorithm: challenge.Algorithm,
		Challenge: challenge.Challenge,
		Number:    int64(number),
		Salt:      challenge.Salt,
		Signature: challenge.Signature,
	}
	b, _ := json.Marshal(p)
	return base64.StdEncoding.EncodeToString(b)
}

func hasPowCookie(w *httptest.ResponseRecorder) bool {
	return findPowCookie(w) != ""
}

func findPowCookie(w *httptest.ResponseRecorder) string {
	for _, c := range w.Result().Cookies() {
		if c.Name == powCookieName {
			return c.Value
		}
	}
	return ""
}

func issueExpiredPowCookie() string {
	expiresAt := time.Now().Add(-time.Hour)
	random := base64.RawURLEncoding.EncodeToString([]byte("expirednonce"))
	payload := "v1." + strconv.FormatInt(expiresAt.Unix(), 10) + "." + random
	sig := signPowCookiePayload(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

func nilContext(w *httptest.ResponseRecorder, req *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}
