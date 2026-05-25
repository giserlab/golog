package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/thanhpk/randstr"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"

	"golog/entity"
	"golog/handler"
	"golog/store"
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
