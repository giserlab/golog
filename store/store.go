package store

import (
	"database/sql"
	"errors"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite", "file:golog.sqlite?cache=shared&_journal_mode=WAL&_synchronous=FULL&_busy_timeout=5000")
	if err != nil {
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
