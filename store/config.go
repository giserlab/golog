package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"golog/entity"
)

const configID = "default"

// LoadConfig loads the site configuration from the config table.
// Returns nil when no configuration has been saved yet.
func LoadConfig() (*entity.Config, error) {
	var data string
	err := db.QueryRow(`SELECT data FROM config WHERE id = ?`, configID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg entity.Config
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig persists the site configuration to the config table.
func SaveConfig(cfg *entity.Config) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO config (id, data) VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET data = excluded.data
	`, configID, string(b))
	return err
}
