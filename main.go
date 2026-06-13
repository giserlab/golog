package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/thanhpk/randstr"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"

	"golog/entity"
	"golog/handler"
	"golog/store"

	"github.com/google/uuid"
)

var (
	Version   = ""
	BuildTime = ""
	Commit    = ""
)

func main() {
	fmt.Println(`
                  _                 
   __ _    ___   | |   ___     __ _ 
  / _  |  / _ \  | |  / _ \   / _  |
 | (_) | | (_) | | | | (_) | | (_) |
  \__, |  \___/  |_|  \___/   \__, |
  |___/                       |___/ `)

	app := &cli.App{
		Name:    "golog",
		Version: fmt.Sprintf("Version: %s\tBuild Time: %s\tCommit: %s", Version, BuildTime, Commit),
		Usage:   "A simple blogging system written in Golang ✨",
		Action:  start,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Usage:   "port to listen on",
				Aliases: []string{"p"},
				Value:   "5201",
			},
			&cli.StringFlag{
				Name:  "tls-key",
				Usage: "path to TLS key file",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "tls-crt",
				Usage: "path to TLS certificate file",
				Value: "",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "reset-password",
				Usage:  "reset the password of a user, email address is required ",
				Action: resetUser,
			},
			{
				Name:   "db:migrate",
				Usage:  "Upgrade or downgrade the database schema. Usage: golog db:migrate [version]",
				Action: dbMigrate,
			},
			{
				Name:   "token:create",
				Usage:  "Create a new API token. Usage: golog token:create <user_id> <name>",
				Action: createToken,
			},
			{
				Name:   "token:delete",
				Usage:  "Delete an API token by ID. Usage: golog token:delete <token_id>",
				Action: deleteToken,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func start(c *cli.Context) error {
	return handler.Start(c, &entity.Injection{
		Version:   Version,
		BuildTime: BuildTime,
		Commit:    Commit,
		GoVersion: runtime.Version(),
	})
}

func resetUser(c *cli.Context) error {
	if err := store.AutoMigrate(); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	u, err := store.GetUserByEmail(c.Args().First())
	if err != nil {
		return err
	}
	pwd := randstr.String(16, randstr.Base64Chars)
	b, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := store.UpdateUserPassword(u.ID, string(b)); err != nil {
		return err
	}
	log.Printf(`Password for user %s has been reset to: "%s"`, u.Email, pwd)
	return nil
}

func dbMigrate(c *cli.Context) error {
	target := 0 // 0 = latest
	if c.Args().Len() > 0 {
		v, err := strconv.Atoi(c.Args().First())
		if err != nil {
			return fmt.Errorf("invalid version number: %q (use an integer or omit for latest)", c.Args().First())
		}
		target = v
	}

	if err := store.MigrateTo(target); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	return nil
}

func createToken(c *cli.Context) error {
	if err := store.AutoMigrate(); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	args := c.Args()
	if args.Len() < 2 {
		return fmt.Errorf("usage: golog token:create <user_id> <name>")
	}
	userID := args.Get(0)
	name := args.Get(1)

	// Verify user exists
	if _, err := store.GetUser(userID); err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	plainToken := randstr.String(32, randstr.Base62Chars)
	hash, err := bcrypt.GenerateFromPassword([]byte(plainToken), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	t := &entity.TokenW{
		ID:        uuid.New().String(),
		Name:      name,
		TokenHash: string(hash),
		UserID:    userID,
		CreatedAt: time.Now().Unix(),
	}
	if err := store.CreateToken(t); err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	log.Printf("Token created: %s", plainToken)
	log.Printf("Name: %s", name)
	log.Printf("ID: %s", t.ID)
	log.Printf("User ID: %s", userID)
	log.Println("Store this token securely — it will not be shown again.")
	return nil
}

func deleteToken(c *cli.Context) error {
	if err := store.AutoMigrate(); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	id := c.Args().First()
	if id == "" {
		return fmt.Errorf("usage: golog token:delete <token_id>")
	}

	if err := store.DeleteToken(id); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	log.Printf("Token %s deleted.", id)
	return nil
}
