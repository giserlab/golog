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
	db, err = sql.Open("sqlite", "file:db.sqlite?cache=shared")
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
