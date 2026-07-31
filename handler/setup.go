package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golog/entity"
	"golog/store"
	"golog/system"
	"golog/util"

	"github.com/urfave/cli/v2"
)

var (
	injection entity.Injection
)

func Start(c *cli.Context, inject *entity.Injection) error {
	if err := store.AutoMigrate(); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	system.SetConfigWriter(store.SaveConfig)
	cfg, err := store.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	system.Config = cfg
	if system.Config != nil {
		if err := system.SaveConfig(); err != nil {
			return fmt.Errorf("failed to apply config: %w", err)
		}
	}

	injection = *inject
	port := c.String("port")
	// Ensure port starts with ":" for the server, strip it for URL display
	addr := port
	if port[0] != ':' {
		addr = ":" + port
	} else {
		port = port[1:]
	}

	// Graceful shutdown: on SIGTERM/SIGINT drain in-flight requests, then
	// checkpoint the WAL into the main DB so recent writes survive even an
	// unclean service stop.
	srv := &http.Server{Addr: addr, Handler: Router}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down, checkpointing WAL...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		if err := store.Checkpoint(); err != nil {
			log.Printf("wal checkpoint failed: %v", err)
		}
	}()

	if c.String("tls-crt") != "" && c.String("tls-key") != "" {
		url := fmt.Sprintf("https://localhost:%s", port)
		fmt.Printf("👋 Visit %s to use Golog\n", url)
		util.OpenBrowser(url)
		if err := srv.ListenAndServeTLS(c.String("tls-crt"), c.String("tls-key")); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	} else {
		url := fmt.Sprintf("http://localhost:%s", port)
		fmt.Printf("👋 Visit %s to use Golog\n", url)
		util.OpenBrowser(url)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}
