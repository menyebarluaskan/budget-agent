// Package httpapi wires the HTTP surface.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"budget-agent/internal/accounts"
)

const readinessTimeout = 2 * time.Second

type Options struct {
	Logger *slog.Logger
	Ready  func(context.Context) error
	DB     *sql.DB
}

type statusResponse struct {
	Status string `json:"status"`
}

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
	if opts.DB != nil {
		svc := accounts.NewService(opts.DB)
		registerAccountRoutes(mux, svc)
	}
	return withMiddleware(logger, mux)
}

func registerAccountRoutes(mux *http.ServeMux, svc *accounts.Service) {
	mux.HandleFunc("GET /api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		list, err := svc.List(r.Context(), ws)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"accounts": list})
	})
	mux.HandleFunc("POST /api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name                string `json:"name"`
			Type                string `json:"type"`
			Currency            string `json:"currency"`
			OpeningBalanceMinor int64  `json:"opening_balance_minor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ws := workspaceID(r)
		if body.Currency == "" {
			body.Currency = "IDR"
		}
		a, err := svc.Create(r.Context(), accounts.CreateInput{
			WorkspaceID:         ws,
			Name:                body.Name,
			Type:                body.Type,
			Currency:            body.Currency,
			OpeningBalanceMinor: body.OpeningBalanceMinor,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, a)
	})
	mux.HandleFunc("GET /api/v1/accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		a, err := svc.Get(r.Context(), ws, r.PathValue("id"))
		if err != nil {
			if err == accounts.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, a)
	})
	mux.HandleFunc("PATCH /api/v1/accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		var body struct {
			Name                *string `json:"name"`
			Archived            *bool   `json:"archived"`
			OpeningBalanceMinor *int64  `json:"opening_balance_minor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		a, err := svc.Update(r.Context(), ws, r.PathValue("id"), accounts.UpdateInput{
			Name:                body.Name,
			Archived:            body.Archived,
			OpeningBalanceMinor: body.OpeningBalanceMinor,
		})
		if err != nil {
			if err == accounts.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, a)
	})
	mux.HandleFunc("DELETE /api/v1/accounts/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		if err := svc.Delete(r.Context(), ws, r.PathValue("id")); err != nil {
			if err == accounts.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func workspaceID(r *http.Request) string {
	if v := r.Header.Get("X-Workspace-Id"); v != "" {
		return v
	}
	return "default"
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"title": msg, "status": status})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
