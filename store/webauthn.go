package store

import (
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

func createWebAuthnTables() error {
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS webauthn_credentials (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id         TEXT NOT NULL,
		credential_id   BLOB NOT NULL,
		public_key      BLOB NOT NULL,
		attestation_type TEXT NOT NULL,
		transport       TEXT NOT NULL DEFAULT '',
		flags           INTEGER NOT NULL DEFAULT 0,
		aaguid          BLOB,
		sign_count      INTEGER NOT NULL DEFAULT 0,
		clone_warning   INTEGER NOT NULL DEFAULT 0,
		attachment      TEXT NOT NULL DEFAULT '',
		created_at      INTEGER NOT NULL
	)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
	CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_cred_cid ON webauthn_credentials (credential_id)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
	CREATE INDEX IF NOT EXISTS idx_wa_cred_uid ON webauthn_credentials (user_id)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS webauthn_sessions (
		challenge   TEXT NOT NULL PRIMARY KEY,
		user_id     BLOB,
		data        BLOB NOT NULL,
		expires     INTEGER NOT NULL
	)`); err != nil {
		return err
	}
	return nil
}

// ===============================
// Credentials
// ===============================

func SaveWebAuthnCredential(userID string, cred *webauthn.Credential) error {
	transport, _ := json.Marshal(cred.Transport)
	flags := cred.Flags.ProtocolValue()

	_, err := db.Exec(`
		INSERT INTO webauthn_credentials
		(user_id, credential_id, public_key, attestation_type, transport,
		 flags, aaguid, sign_count, clone_warning, attachment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, cred.ID, cred.PublicKey, cred.AttestationType, string(transport),
		int(flags), cred.Authenticator.AAGUID, cred.Authenticator.SignCount,
		boolToInt(cred.Authenticator.CloneWarning), cred.Authenticator.Attachment,
		time.Now().Unix(),
	)
	return err
}

func GetWebAuthnCredentials(userID string) ([]webauthn.Credential, error) {
	rows, err := db.Query(`
		SELECT credential_id, public_key, attestation_type, transport,
			   flags, aaguid, sign_count, clone_warning, attachment
		FROM webauthn_credentials
		WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCredentials(rows)
}

func GetWebAuthnCredentialByID(credentialID []byte) (*webauthn.Credential, string, error) {
	var userID string
	var cred webauthn.Credential
	var transportJSON string
	var flagsRaw int
	var cloneWarnInt int

	err := db.QueryRow(`
		SELECT user_id, credential_id, public_key, attestation_type, transport,
			   flags, aaguid, sign_count, clone_warning, attachment
		FROM webauthn_credentials
		WHERE credential_id = ?`, credentialID).Scan(
		&userID, &cred.ID, &cred.PublicKey, &cred.AttestationType, &transportJSON,
		&flagsRaw, &cred.Authenticator.AAGUID, &cred.Authenticator.SignCount,
		&cloneWarnInt, &cred.Authenticator.Attachment,
	)
	if err != nil {
		return nil, "", err
	}

	cred.Authenticator.CloneWarning = cloneWarnInt != 0
	cred.Flags = webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(flagsRaw))
	json.Unmarshal([]byte(transportJSON), &cred.Transport)

	return &cred, userID, nil
}

type WebAuthnCredRow struct {
	ID           int64
	UserID       string
	CredentialID []byte
	CreatedAt    int64
}

func ListWebAuthnCredentials(userID string) ([]WebAuthnCredRow, error) {
	rows, err := db.Query(`
		SELECT id, user_id, credential_id, created_at
		FROM webauthn_credentials
		WHERE user_id = ?
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []WebAuthnCredRow
	for rows.Next() {
		var r WebAuthnCredRow
		if err := rows.Scan(&r.ID, &r.UserID, &r.CredentialID, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func DeleteWebAuthnCredential(id int64) error {
	_, err := db.Exec(`DELETE FROM webauthn_credentials WHERE id = ?`, id)
	return err
}

func UpdateWebAuthnCredential(userID string, cred *webauthn.Credential) error {
	transport, _ := json.Marshal(cred.Transport)
	flags := cred.Flags.ProtocolValue()

	_, err := db.Exec(`
		UPDATE webauthn_credentials SET
			public_key = ?, attestation_type = ?, transport = ?,
			flags = ?, aaguid = ?, sign_count = ?, clone_warning = ?, attachment = ?
		WHERE user_id = ? AND credential_id = ?`,
		cred.PublicKey, cred.AttestationType, string(transport),
		int(flags), cred.Authenticator.AAGUID, cred.Authenticator.SignCount,
		boolToInt(cred.Authenticator.CloneWarning), cred.Authenticator.Attachment,
		userID, cred.ID,
	)
	return err
}

// ===============================
// Sessions
// ===============================

func SaveWebAuthnSession(session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO webauthn_sessions (challenge, user_id, data, expires)
		VALUES (?, ?, ?, ?)`,
		session.Challenge, session.UserID, data, session.Expires.Unix(),
	)
	return err
}

func GetWebAuthnSession(challenge string) (*webauthn.SessionData, error) {
	var data []byte
	var expires int64
	err := db.QueryRow(`
		SELECT data, expires FROM webauthn_sessions
		WHERE challenge = ?`, challenge).Scan(&data, &expires)
	if err != nil {
		return nil, err
	}
	if expires > 0 && time.Now().Unix() > expires {
		DeleteWebAuthnSession(challenge)
		return nil, ErrSessionExpired
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteWebAuthnSession(challenge string) error {
	_, err := db.Exec(`DELETE FROM webauthn_sessions WHERE challenge = ?`, challenge)
	return err
}

func CleanupExpiredWebAuthnSessions() error {
	_, err := db.Exec(`DELETE FROM webauthn_sessions WHERE expires <= ?`, time.Now().Unix())
	return err
}

// ===============================
// Helpers
// ===============================

var ErrSessionExpired = &expiredError{}

type expiredError struct{}

func (e *expiredError) Error() string {
	return "webauthn session expired"
}

func scanCredentials(rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
}) ([]webauthn.Credential, error) {
	var creds []webauthn.Credential
	for rows.Next() {
		var c webauthn.Credential
		var transportJSON string
		var flagsRaw int
		var cloneWarnInt int
		var aaguid []byte

		err := rows.Scan(
			&c.ID, &c.PublicKey, &c.AttestationType, &transportJSON,
			&flagsRaw, &aaguid, &c.Authenticator.SignCount,
			&cloneWarnInt, &c.Authenticator.Attachment,
		)
		if err != nil {
			return nil, err
		}

		c.Authenticator.AAGUID = aaguid
		c.Authenticator.CloneWarning = cloneWarnInt != 0
		c.Flags = webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(flagsRaw))
		json.Unmarshal([]byte(transportJSON), &c.Transport)

		creds = append(creds, c)
	}
	return creds, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
