package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// newWebAuthn creates a webauthn.WebAuthn instance from the request's Host header.
func newWebAuthn(c *gin.Context) (*webauthn.WebAuthn, error) {
	rpid := c.Request.Host
	if idx := strings.Index(rpid, ":"); idx != -1 {
		rpid = rpid[:idx]
	}
	scheme := "https://"
	if c.Request.TLS == nil {
		scheme = "http://"
		if fwd := c.GetHeader("X-Forwarded-Proto"); fwd == "https" {
			scheme = "https://"
		}
	}
	origin := scheme + c.Request.Host

	return webauthn.New(&webauthn.Config{
		RPID:          rpid,
		RPDisplayName: system.Config.Name,
		RPOrigins:     []string{origin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationPreferred,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute},
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute},
		},
	})
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

	options, session, err := webAuthn.BeginDiscoverableLogin()
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

	session, err := store.GetWebAuthnSession(challenge)
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

	// Cleanup session
	store.DeleteWebAuthnSession(challenge)

	// Establish the user session
	setUserID(c, wu.ID)
	c.JSON(http.StatusOK, gin.H{"redirect": "../admin/posts"})
}

// ===============================
// Passkey Registration (admin only)
// ===============================

func PasskeyRegisterBegin(c *gin.Context) {
	u, err := self(c)
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

	session, err := store.GetWebAuthnSession(challenge)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid or expired session"})
		return
	}

	u, err := self(c)
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

	cred, err := webAuthn.FinishRegistration(wu, *session, c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := store.SaveWebAuthnCredential(u.ID, cred); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	store.DeleteWebAuthnSession(challenge)
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
	u, err := self(c)
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
		items = append(items, passkeyItem{
			ID:            r.ID,
			CredentialID:  base64.RawURLEncoding.EncodeToString(r.CredentialID)[:12] + "...",
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
	u, err := self(c)
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

// isAjax checks if the request is an AJAX/XHR request.
func isAjax(c *gin.Context) bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest"
}
