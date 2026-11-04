// Command budget-agent runs the HTTP API and, in production, the embedded web
// UI. Scaffolding serves only the health endpoints until the backend milestones
// wire in storage and domain handlers.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"budget-agent/internal/accounts"
	"budget-agent/internal/ai"
	"budget-agent/internal/config"
	"budget-agent/internal/db"
	"budget-agent/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
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
