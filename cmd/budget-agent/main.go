// Command budget-agent runs the HTTP API and, in production, the embedded web
// UI. Scaffolding serves only the health endpoints until the backend milestones
// wire in storage and domain handlers.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "modernc.org/sqlite"

	"budget-agent/internal/accounts"
	"budget-agent/internal/ai"
	"budget-agent/internal/config"
	"budget-agent/internal/db"
	"budget-agent/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(runHealthcheck())
		case "backup":
			os.Exit(runBackup(os.Args[2:]))
		}
	}
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func runBackup(args []string) int {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	dest := fs.String("dest", "", "destination directory for the backup")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *dest == "" {
		fmt.Fprintln(os.Stderr, "--dest is required")
		return 2
	}
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_PATH is required")
		return 2
	}
	if err := os.MkdirAll(*dest, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	out := filepath.Join(*dest, "budget-"+stamp+".db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer db.Close()
	// VACUUM INTO
	_, err = db.Exec("VACUUM INTO '" + out + "'")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("backup written to", out)
	return 0
}

func runHealthcheck() int {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if err := accounts.EnsureDefaultWorkspace(context.Background(), database); err != nil {
		return fmt.Errorf("bootstrap workspace: %w", err)
	}

	aiProvider, aiEnabled := ai.NewFromConfig(cfg.AIEnabled, cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel, cfg.AIVisionModel, cfg.AITimeout)

	ready := func(ctx context.Context) error { return database.PingContext(ctx) }
	server := &http.Server{
		Addr: cfg.Addr,
		Handler: httpapi.New(httpapi.Options{
			Logger:          logger,
			Ready:           ready,
			DB:              database,
			AuthMode:        cfg.AuthMode,
			TokenPepper:     cfg.TokenPepper,
			DefaultCurrency: cfg.DefaultCurrency,
			ReceiptsDir:     cfg.ReceiptsDir,
			AIEnabled:       aiEnabled,
			AIProvider:      aiProvider,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
