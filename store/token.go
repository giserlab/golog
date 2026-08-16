package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"

	"golog/entity"

	"golang.org/x/crypto/bcrypt"
)

var ErrTokenNotFound = errors.New("token not found")

// TokenDigest returns the hex SHA-256 digest of a plaintext token.
// Stored alongside the bcrypt hash so authentication can do an indexed
// lookup instead of bcrypt-comparing against every token (which is an
// O(N) CPU amplification / DoS vector).
func TokenDigest(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func CreateToken(t *entity.TokenW) error {
	if _, err := db.Exec(`INSERT INTO tokens (id, name, token_hash, token_hash_sha256, user_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.TokenHash, t.TokenHashSHA256, t.UserID, t.CreatedAt); err != nil {
		return err
	}
	return nil
}

// GetTokenByHash looks up a token by its plaintext value.
// Fast path: SHA-256 index lookup + one bcrypt confirmation.
// Fallback: tokens created before migration v8 (empty token_hash_sha256)
// are verified with a bounded legacy scan; on a match the digest is
// back-filled so subsequent lookups take the fast path.
func GetTokenByHash(tokenHash string) (*entity.TokenR, error) {
	digest := TokenDigest(tokenHash)

	var t entity.TokenR
	err := db.QueryRow(`SELECT id, name, token_hash, user_id, created_at FROM tokens WHERE token_hash_sha256 = ?`, digest).
		Scan(&t.ID, &t.Name, &t.TokenHash, &t.UserID, &t.CreatedAt)
	if err == nil {
		// bcrypt 复核：防止任何（理论上的）SHA-256 碰撞被误认为有效 token。
		if bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(tokenHash)) == nil {
			return &t, nil
		}
		return nil, ErrTokenNotFound
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 旧 token 回退：仅当库里存在未迁移（sha256 为空）的 token 时才扫描，
	// 攻击者无法让空 sha256 token 出现，因此扫描范围受控。
	var legacy int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tokens WHERE token_hash_sha256 = ''`).Scan(&legacy); err != nil {
		return nil, err
	}
	if legacy == 0 {
		return nil, ErrTokenNotFound
	}

	rows, err := db.Query(`SELECT id, name, token_hash, user_id, created_at FROM tokens WHERE token_hash_sha256 = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lt entity.TokenR
		if err := rows.Scan(&lt.ID, &lt.Name, &lt.TokenHash, &lt.UserID, &lt.CreatedAt); err != nil {
			return nil, err
		}
		if bcrypt.CompareHashAndPassword([]byte(lt.TokenHash), []byte(tokenHash)) != nil {
			continue
		}
		// 命中：回写摘要，后续请求走索引快速路径。
		if _, err := db.Exec(`UPDATE tokens SET token_hash_sha256 = ? WHERE id = ?`, digest, lt.ID); err != nil {
			return nil, err
		}
		return &lt, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nil, ErrTokenNotFound
}

func ListTokens() ([]*entity.TokenR, error) {
	rows, err := db.Query(`SELECT id, name, token_hash, user_id, created_at FROM tokens ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*entity.TokenR
	for rows.Next() {
		var t entity.TokenR
		if err := rows.Scan(&t.ID, &t.Name, &t.TokenHash, &t.UserID, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &t)
	}
	return tokens, nil
}

func ListTokensByUser(userID string) ([]*entity.TokenR, error) {
	rows, err := db.Query(`SELECT id, name, token_hash, user_id, created_at FROM tokens WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*entity.TokenR
	for rows.Next() {
		var t entity.TokenR
		if err := rows.Scan(&t.ID, &t.Name, &t.TokenHash, &t.UserID, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &t)
	}
	return tokens, nil
}

func GetToken(id string) (*entity.TokenR, error) {
	row := db.QueryRow(`SELECT id, name, token_hash, user_id, created_at FROM tokens WHERE id = ?`, id)
	var t entity.TokenR
	if err := row.Scan(&t.ID, &t.Name, &t.TokenHash, &t.UserID, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func DeleteToken(id string) error {
	if _, err := db.Exec(`DELETE FROM tokens WHERE id = ?`, id); err != nil {
		return err
	}
	return nil
}
