package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// Migration represents a single database schema migration.
// Each version has a forward (Up) and backward (Down) function.
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.Tx) error
	Down        func(*sql.Tx) error
}

// All registered migrations, ordered by version.
// Append new migrations at the end — never modify or reorder existing entries.
var migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema: users, posts, navigations, tags, webauthn",
		Up:          migrationV1Up,
		Down:        migrationV1Down,
	},
}

// ─── Migration engine ───────────────────────────────────────────────────────

// ensureMigrationsTable creates the _migrations tracking table if it doesn't exist.
func ensureMigrationsTable() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			version     INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at  INTEGER NOT NULL
		)
	`)
	return err
}

// appliedVersions returns the set of already-applied migration versions.
func appliedVersions() (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM _migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions[v] = true
	}
	return versions, rows.Err()
}

// latestVersion returns the highest version number registered.
func latestVersion() int {
	maxV := 0
	for _, m := range migrations {
		if m.Version > maxV {
			maxV = m.Version
		}
	}
	return maxV
}

// CurrentVersion returns the highest applied migration version.
// Returns 0 if no migrations have been applied.
func CurrentVersion() (int, error) {
	if err := ensureMigrationsTable(); err != nil {
		return 0, err
	}
	versions, err := appliedVersions()
	if err != nil {
		return 0, err
	}
	maxV := 0
	for v := range versions {
		if v > maxV {
			maxV = v
		}
	}
	return maxV, nil
}

// AutoMigrate runs all pending migrations up to the latest version.
// Safe to call repeatedly — only unapplied migrations will execute.
// Called automatically during server startup.
func AutoMigrate() error {
	return MigrateTo(0) // 0 means "latest"
}

// MigrateTo migrates the database schema to version target.
//   - target == 0 → migrate to the latest version
//   - target > current → applies Up migrations (upgrade)
//   - target < current → applies Down migrations (downgrade)
//
// Each migration runs inside a transaction. If any step fails,
// only that individual migration is rolled back; prior migrations are preserved.
func MigrateTo(target int) error {
	if err := ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	applied, err := appliedVersions()
	if err != nil {
		return fmt.Errorf("failed to read applied versions: %w", err)
	}

	if target == 0 {
		target = latestVersion()
	}

	current := 0
	for v := range applied {
		if v > current {
			current = v
		}
	}

	if current == target {
		log.Printf("[migrate] database already at version %d, nothing to do", target)
		return nil
	}

	if target > current {
		// Forward migration: apply Up from current+1 → target
		for _, m := range migrations {
			if m.Version > current && m.Version <= target {
				log.Printf("[migrate] ↑ upgrading to version %d: %s", m.Version, m.Description)
				if err := applyUp(m); err != nil {
					return fmt.Errorf("migration %d failed: %w", m.Version, err)
				}
			}
		}
	} else {
		// Backward migration: apply Down from current → target+1 (reverse order)
		for i := len(migrations) - 1; i >= 0; i-- {
			m := migrations[i]
			if m.Version <= current && m.Version > target {
				log.Printf("[migrate] ↓ downgrading from version %d: %s", m.Version, m.Description)
				if err := applyDown(m); err != nil {
					return fmt.Errorf("rollback of migration %d failed: %w", m.Version, err)
				}
			}
		}
	}

	newVer, _ := CurrentVersion()
	log.Printf("[migrate] complete — database at version %d", newVer)
	return nil
}

// applyUp runs a single forward migration in a transaction.
func applyUp(m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := m.Up(tx); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO _migrations (version, description, applied_at) VALUES (?, ?, ?)`,
		m.Version, m.Description, time.Now().Unix(),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// applyDown runs a single backward migration in a transaction.
func applyDown(m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := m.Down(tx); err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM _migrations WHERE version = ?`, m.Version); err != nil {
		return err
	}

	return tx.Commit()
}

// ─── Migration v1: Initial schema ───────────────────────────────────────────

func migrationV1Up(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id             TEXT NOT NULL PRIMARY KEY,
			email          TEXT NOT NULL UNIQUE,
			nickname       TEXT NOT NULL UNIQUE,
			password       TEXT NOT NULL,
			bio            TEXT NOT NULL,
			created_at     INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id             TEXT NOT NULL PRIMARY KEY,
			type           TEXT NOT NULL,
			title          TEXT NOT NULL,
			slug           TEXT NOT NULL,
			excerpt        TEXT NOT NULL,
			author_id      TEXT NOT NULL,
			password       TEXT NOT NULL,
			visibility     TEXT NOT NULL,
			content        TEXT NOT NULL,
			pinned_at      INTEGER NOT NULL,
			published_at   INTEGER NOT NULL,
			created_at     INTEGER NOT NULL,
			updated_at     INTEGER NOT NULL,
			trashed_at     INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS navigations (
			id         TEXT NOT NULL PRIMARY KEY,
			url        TEXT NOT NULL,
			name       TEXT NOT NULL,
			sequence   INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id          TEXT NOT NULL PRIMARY KEY,
			slug        TEXT NOT NULL,
			name        TEXT NOT NULL,
			description TEXT NOT NULL,
			created_at  INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS post_tags (
			tag_id  TEXT NOT NULL,
			post_id TEXT NOT NULL,
			PRIMARY KEY (tag_id, post_id)
		)`,
		`CREATE TABLE IF NOT EXISTS webauthn_credentials (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id          TEXT NOT NULL,
			credential_id    BLOB NOT NULL,
			public_key       BLOB NOT NULL,
			attestation_type TEXT NOT NULL,
			transport        TEXT NOT NULL DEFAULT '',
			flags            INTEGER NOT NULL DEFAULT 0,
			aaguid           BLOB,
			sign_count       INTEGER NOT NULL DEFAULT 0,
			clone_warning    INTEGER NOT NULL DEFAULT 0,
			attachment       TEXT NOT NULL DEFAULT '',
			created_at       INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_cred_cid ON webauthn_credentials (credential_id)`,
		`CREATE INDEX IF NOT EXISTS idx_wa_cred_uid ON webauthn_credentials (user_id)`,
		`CREATE TABLE IF NOT EXISTS webauthn_sessions (
			challenge   TEXT NOT NULL PRIMARY KEY,
			user_id     BLOB,
			data        BLOB NOT NULL,
			expires     INTEGER NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrationV1Down(tx *sql.Tx) error {
	stmts := []string{
		`DROP TABLE IF EXISTS webauthn_sessions`,
		`DROP INDEX IF EXISTS idx_wa_cred_uid`,
		`DROP INDEX IF EXISTS idx_wa_cred_cid`,
		`DROP TABLE IF EXISTS webauthn_credentials`,
		`DROP TABLE IF EXISTS post_tags`,
		`DROP TABLE IF EXISTS tags`,
		`DROP TABLE IF EXISTS navigations`,
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS users`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
