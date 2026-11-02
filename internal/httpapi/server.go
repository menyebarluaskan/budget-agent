// Package httpapi wires the HTTP surface. Feature handlers arrive with the
// backend milestones; this package currently exposes liveness and readiness.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

// Options configures the HTTP handler.
type Options struct {
	Logger *slog.Logger
	// Ready reports whether dependencies are usable. A nil check means ready.
	Ready func(context.Context) error
}

type statusResponse struct {
	Status string `json:"status"`
}

// New returns the root HTTP handler.
func New(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ready := opts.Ready
	if ready == nil {
		ready = func(context.Context) error { return nil }
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := ready(ctx); err != nil {
			logger.Debug("readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "not ready"})
			return
		}
		writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
	})

	return withMiddleware(logger, mux)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
