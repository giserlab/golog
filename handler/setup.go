package handler

import (
	"fmt"

	"golog/entity"
	"golog/util"

	"github.com/urfave/cli/v2"
)

var (
	injection entity.Injection
)

func Start(c *cli.Context, inject *entity.Injection) error {
	injection = *inject
	port := c.String("port")
	// Ensure port starts with ":" for Router.Run(), strip it for URL display
	addr := port
	if port[0] != ':' {
		addr = ":" + port
	} else {
		port = port[1:]
	}
	if c.String("tls-crt") != "" && c.String("tls-key") != "" {
		url := fmt.Sprintf("https://localhost:%s", port)
		fmt.Printf("👋 Visit %s to use Golog\n", url)
		util.OpenBrowser(url)
		return Router.RunTLS(addr, c.String("tls-crt"), c.String("tls-key"))
	} else {
		url := fmt.Sprintf("http://localhost:%s", port)
		fmt.Printf("👋 Visit %s to use Golog\n", url)
		util.OpenBrowser(url)
		return Router.Run(addr)
	}
}
