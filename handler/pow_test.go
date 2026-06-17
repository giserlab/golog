package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"golog/entity"
	"golog/system"

	"github.com/gin-gonic/gin"
)

func withPoWConfig(t *testing.T, config *entity.Config) {
	t.Helper()
	oldConfig := system.Config
	system.Config = config
	t.Cleanup(func() {
		system.Config = oldConfig
	})
}

func solveNonceForTest(challenge string, difficulty int) string {
	for nonce := 0; ; nonce++ {
		s := strconv.Itoa(nonce)
		if verifySolution(challenge, s, difficulty) {
			return s
		}
	}
}

func TestPowRejectsUnsignedFutureChallenge(t *testing.T) {
	withPoWConfig(t, &entity.Config{
		IsPublic:      true,
		PoWEnabled:    true,
		PoWDifficulty: 1,
		PoWTTL:        24,
	})

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	challenge := base64.RawURLEncoding.EncodeToString(b) + "." +
		strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10)
	nonce := solveNonceForTest(challenge, 1)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pow/solve", nil)
	PowSolve(nilContext(w, req), &PowSolveRequest{
		Challenge: challenge,
		Nonce:     nonce,
		Redirect:  "/",
	})

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == powCookieName {
			t.Fatalf("unsigned challenge was accepted and received %s cookie", powCookieName)
		}
	}
}

func TestPowCookieRequiresSignedChallenge(t *testing.T) {
	withPoWConfig(t, &entity.Config{
		IsPublic:      true,
		PoWEnabled:    true,
		PoWDifficulty: 1,
		PoWTTL:        24,
	})

	challenge := generateChallenge(1).Challenge
	nonce := solveNonceForTest(challenge, 1)
	if !verifyCookie(issueCookie(challenge, nonce)) {
		t.Fatal("signed challenge cookie was rejected")
	}

	unsignedChallenge := "unsigned." + strconv.FormatInt(time.Now().Unix(), 10)
	unsignedNonce := solveNonceForTest(unsignedChallenge, 1)
	if verifyCookie(issueCookie(unsignedChallenge, unsignedNonce)) {
		t.Fatal("cookie with unsigned challenge was accepted")
	}
}

func TestNoRouteRequiresPowWhenEnabled(t *testing.T) {
	withPoWConfig(t, &entity.Config{
		IsPublic:      true,
		PoWEnabled:    true,
		PoWDifficulty: 1,
		PoWTTL:        24,
	})

	for _, path := range []string{"/missing-page", "/pow-missing-page"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			Router.ServeHTTP(w, req)

			if got := w.Code; got != http.StatusFound {
				t.Fatalf("status = %d, want %d", got, http.StatusFound)
			}
			if got, want := w.Header().Get("Location"), "/pow?redirect=%2F"+path[1:]; got != want {
				t.Fatalf("Location = %q, want %q", got, want)
			}
		})
	}
}

func TestPowMiddlewareKeepsExcludedRoutesOpen(t *testing.T) {
	withPoWConfig(t, &entity.Config{
		IsPublic:      true,
		PoWEnabled:    true,
		PoWDifficulty: 1,
		PoWTTL:        24,
	})

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
		"/uploads/photo.jpg",
		"/assets/pow.js",
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

func TestPowSolveRejectsInvalidNonce(t *testing.T) {
	withPoWConfig(t, &entity.Config{
		IsPublic:      true,
		PoWEnabled:    true,
		PoWDifficulty: 1,
		PoWTTL:        24,
	})

	challenge := generateChallenge(1).Challenge
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pow/solve", nil)
	PowSolve(nilContext(w, req), &PowSolveRequest{
		Challenge: challenge,
		Nonce:     "1;2",
		Redirect:  "/",
	})

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == powCookieName {
			t.Fatalf("invalid nonce was accepted and received %s cookie", powCookieName)
		}
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

func nilContext(w *httptest.ResponseRecorder, req *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c
}
