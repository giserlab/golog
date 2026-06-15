package store

import (
	"errors"

	"golog/entity"

	"golang.org/x/crypto/bcrypt"
)

var ErrTokenNotFound = errors.New("token not found")

func CreateToken(t *entity.TokenW) error {
	if _, err := db.Exec(`INSERT INTO tokens (id, name, token_hash, user_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.TokenHash, t.UserID, t.CreatedAt); err != nil {
		return err
	}
	return nil
}

func GetTokenByHash(tokenHash string) (*entity.TokenR, error) {
	tokens, err := ListTokens()
	if err != nil {
		return nil, err
	}
	for _, t := range tokens {
		if err := bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(tokenHash)); err == nil {
			return t, nil
		}
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
