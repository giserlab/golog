package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func init() {
	if err := Open("golog.sqlite"); err != nil {
		log.Fatalln(err)
	}
	go func() {
		for {
			if err := ClearExpiredTrashPosts(); err != nil {
				log.Println(err)
			}
			if err := CleanupExpiredWebAuthnSessions(); err != nil {
				log.Println(err)
			}
			<-time.After(24 * time.Hour)
		}
	}()

	// ponytail: every minute, merge WAL frames into the main DB so recent
	// writes survive even a hard kill; PASSIVE never blocks traffic.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := db.Exec(`PRAGMA wal_checkpoint(PASSIVE)`); err != nil {
				log.Println(err)
			}
		}
	}()
}

// Open connects to the SQLite database at path, replacing any previous
// connection. Called with the default path in init(); the CLI --db flag calls
// it again with a custom path before any command runs.
func Open(path string) error {
	dsn := fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL&_synchronous=FULL&_busy_timeout=5000", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	if db != nil {
		db.Close()
	}
	db = d
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// Checkpoint merges the WAL into the main database file. Call it during
// graceful shutdown so recent writes are persisted even if the process is
// killed before SQLite runs its own checkpoint.
func Checkpoint() error {
	var busy, logCount, ckpt int
	if err := db.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &logCount, &ckpt); err != nil {
		return err
	}
	return nil
}
