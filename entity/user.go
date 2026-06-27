package entity

import (
	"crypto/md5"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type UserW struct {
	ID        string
	Email     string
	Password  string
	Nickname  string
	Bio       string
	Role      string
	CreatedAt int64
}

type UserR struct {
	ID        string
	Email     string
	Password  string
	Nickname  string
	Bio       string
	Role      string
	CreatedAt int64
}

func (u *UserR) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *UserR) Gravatar() string {
	data := []byte(u.Email)
	return fmt.Sprintf("https://api.dicebear.com/7.x/identicon/svg?seed=%x", md5.Sum(data))
}

// WebAuthnUser wraps UserR with WebAuthn credentials, implementing webauthn.User.
type WebAuthnUser struct {
	*UserR
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.ID)
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.Email
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.Nickname
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

func (u *WebAuthnUser) WebAuthnCredentialExcludeList() []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, len(u.Credentials))
	for i, cred := range u.Credentials {
		descriptors[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
		}
	}
	return descriptors
}
