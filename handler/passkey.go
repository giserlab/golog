package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// newWebAuthn creates a WebAuthn instance using configured RP settings when
// present, falling back to the current request for older installations.
func newWebAuthn(c *gin.Context) (*webauthn.WebAuthn, error) {
	rpid, origins, err := webAuthnRelyingParty(c)
	if err != nil {
		return nil, err
	}

	return webauthn.New(&webauthn.Config{
		RPID:          rpid,
		RPDisplayName: system.Config.Name,
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationRequired,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute},
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute},
		},
	})
}

func webAuthnRelyingParty(c *gin.Context) (string, []string, error) {
	origins, err := configuredWebAuthnOrigins()
	if err != nil {
		return "", nil, err
	}

	rpid := strings.TrimSpace(system.Config.WebAuthnRPID)
	if rpid == "" && len(origins) > 0 {
		rpid, err = originRPID(origins[0])
		if err != nil {
			return "", nil, err
		}
	}
	if rpid == "" {
		rpid, err = hostWithoutPort(c.Request.Host)
		if err != nil {
			return "", nil, err
		}
	}
	if len(origins) == 0 {
		origin, err := requestOrigin(c)
		if err != nil {
			return "", nil, err
		}
		origins = []string{origin}
	}
	return rpid, origins, nil
}

func configuredWebAuthnOrigins() ([]string, error) {
	var origins []string
	for _, raw := range system.Config.WebAuthnOrigins {
		origin, err := normalizeWebAuthnOrigin(raw)
		if err != nil {
			return nil, err
		}
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins, nil
}

func normalizeWebAuthnOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("invalid webauthn origin scheme")
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid webauthn origin host")
	}
	return u.Scheme + "://" + u.Host, nil
}

func originRPID(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	return hostWithoutPort(u.Host)
}

func requestOrigin(c *gin.Context) (string, error) {
	if _, err := hostWithoutPort(c.Request.Host); err != nil {
		return "", err
	}
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
		if fwd := c.GetHeader("X-Forwarded-Proto"); fwd == "https" {
			scheme = "https"
		}
	}
	return scheme + "://" + c.Request.Host, nil
}

func hostWithoutPort(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("missing host")
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.Trim(host, "[]"), nil
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(h, "[]"), nil
	}
	if strings.Contains(host, ":") {
		return "", fmt.Errorf("invalid host")
	}
	return host, nil
}

// ===============================
// Passkey Login (Discoverable / Usernameless)
// ===============================

func PasskeyLoginBegin(c *gin.Context) {
	webAuthn, err := newWebAuthn(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	options, session, err := webAuthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := store.SaveWebAuthnSession(session); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge": session.Challenge,
		"publicKey": options.Response,
	})
}

func PasskeyLoginFinish(c *gin.Context) {
	challenge := c.Query("challenge")
	if challenge == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing session"})
		return
	}

	session, err := store.ConsumeWebAuthnSession(challenge)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid or expired session"})
		return
	}

	webAuthn, err := newWebAuthn(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Discoverable login: look up user by the user handle from the authenticator.
	discoverHandler := func(rawID, userHandle []byte) (webauthn.User, error) {
		u, err := store.GetUser(string(userHandle))
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		creds, err := store.GetWebAuthnCredentials(u.ID)
		if err != nil {
			return nil, err
		}
		return &entity.WebAuthnUser{
			UserR:       u,
			Credentials: creds,
		}, nil
	}

	validatedUser, cred, err := webAuthn.FinishPasskeyLogin(discoverHandler, *session, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wu := validatedUser.(*entity.WebAuthnUser)

	// Update credential (sign count, etc.)
	if err := store.UpdateWebAuthnCredential(wu.ID, cred); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Establish the user session
	setUserID(c, wu.ID)
	c.JSON(http.StatusOK, gin.H{"redirect": "../admin/posts"})
}

// ===============================
// Passkey Registration (admin only)
// ===============================

func PasskeyRegisterBegin(c *gin.Context) {
	u, err := resolveUser(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}

	creds, err := store.GetWebAuthnCredentials(u.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	webAuthn, err := newWebAuthn(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wu := &entity.WebAuthnUser{
		UserR:       u,
		Credentials: creds,
	}

	options, session, err := webAuthn.BeginRegistration(wu)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := store.SaveWebAuthnSession(session); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge": session.Challenge,
		"publicKey": options.Response,
	})
}

func PasskeyRegisterFinish(c *gin.Context) {
	challenge := c.Query("challenge")
	if challenge == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing session"})
		return
	}

	session, err := store.ConsumeWebAuthnSession(challenge)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid or expired session"})
		return
	}

	u, err := resolveUser(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	if !bytes.Equal(session.UserID, []byte(u.ID)) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid session user"})
		return
	}

	creds, err := store.GetWebAuthnCredentials(u.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	webAuthn, err := newWebAuthn(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wu := &entity.WebAuthnUser{
		UserR:       u,
		Credentials: creds,
	}

	cred, err := webAuthn.FinishRegistration(wu, *session, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := store.SaveWebAuthnCredential(u.ID, cred); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ===============================
// Passkey Management (admin)
// ===============================

type passkeyItem struct {
	ID            int64  `json:"id"`
	CredentialID  string `json:"credential_id"`
	CreatedAt     int64  `json:"created_at"`
	CreatedAtForm string `json:"created_at_form"`
}

func PasskeyList(c *gin.Context) {
	u, err := resolveUser(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}

	rows, err := store.ListWebAuthnCredentials(u.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]passkeyItem, 0, len(rows))
	for _, r := range rows {
		credentialID := base64.RawURLEncoding.EncodeToString(r.CredentialID)
		if len(credentialID) > 12 {
			credentialID = credentialID[:12] + "..."
		}
		items = append(items, passkeyItem{
			ID:            r.ID,
			CredentialID:  credentialID,
			CreatedAt:     r.CreatedAt,
			CreatedAtForm: time.Unix(r.CreatedAt, 0).Format("2006-01-02 15:04"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"passkeys": items})
}

type PasskeyDeleteRequest struct {
	ID int64 `form:"id" binding:"required"`
}

func PasskeyDelete(c *gin.Context) {
	u, err := resolveUser(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}

	var req PasskeyDeleteRequest
	if err := c.ShouldBind(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure the credential belongs to the current user
	rows, err := store.ListWebAuthnCredentials(u.ID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	found := false
	for _, r := range rows {
		if r.ID == req.ID {
			found = true
			break
		}
	}
	if !found {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "credential not found"})
		return
	}

	if err := store.DeleteWebAuthnCredential(req.ID); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if isAjax(c) {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	setMessage(c, "passkey_delete_success")
	c.Redirect(http.StatusFound, "../user/"+u.ID)
}

// resolveUser returns the user specified by the "user_id" query param,
// or falls back to the currently logged-in user.
func resolveUser(c *gin.Context) (*entity.UserR, error) {
	if uid := c.Query("user_id"); uid != "" {
		return store.GetUser(uid)
	}
	return self(c)
}

// isAjax checks if the request is an AJAX/XHR request.
func isAjax(c *gin.Context) bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest"
}
