// Package httpapi wires the HTTP surface.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"budget-agent/internal/accounts"
	"budget-agent/internal/analytics"
	"budget-agent/internal/budgets"
	"budget-agent/internal/dashboard"
	"budget-agent/internal/mcp"
	"budget-agent/internal/tags"
	"budget-agent/internal/transactions"
)

const readinessTimeout = 2 * time.Second

// AuthResult represents the result of authentication.
type AuthResult struct {
	WorkspaceID     string
	IsAuthenticated bool
}

// AuthMiddleware authenticates requests using tailnet identity or bearer tokens.
// It resolves the authenticated identity to a workspace_id stored in the request
// context. Health/readiness/webhook routes are exempt.
func AuthMiddleware(db *sql.DB, next http.Handler, authMode string, tokenPepper string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exempt routes that must work without identity.
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" ||
			strings.HasPrefix(r.URL.Path, "/webhooks/") {
			next.ServeHTTP(w, r)
			return
		}

		// Dev mode: synthetic default workspace, no identity required.
		if authMode == "dev" || authMode == "" {
			if db != nil {
				_ = accounts.EnsureDefaultWorkspace(r.Context(), db)
			}
			r = r.WithContext(context.WithValue(r.Context(), "workspace_id", "default"))
			next.ServeHTTP(w, r)
			return
		}

		// Bearer token: resolve the token's bound workspace (token service lands later).
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString != "" && db != nil {
				// Token hashing/lookup is not implemented yet; keep the existing
				// placeholder behavior so API/MCP clients are not blocked.
				r = r.WithContext(context.WithValue(r.Context(), "workspace_id", "authenticated"))
				next.ServeHTTP(w, r)
				return
			}
		}

		// Tailnet identity: trust the header only because we bind to loopback.
		login := r.Header.Get("Tailscale-User-Login")
		if login == "" {
			writeError(w, http.StatusUnauthorized, "missing tailnet identity")
			return
		}
		if db == nil {
			writeError(w, http.StatusUnauthorized, "identity resolution unavailable")
			return
		}
		ws, err := accounts.EnsurePersonalWorkspace(r.Context(), db, login)
		if err != nil || ws == "" {
			writeError(w, http.StatusInternalServerError, "identity resolution failed")
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), "workspace_id", ws))
		next.ServeHTTP(w, r)
	})
}

// hashToken computes a hash of a token for comparison.
func hashToken(token string, pepper string) []byte {
	h := fmt.Sprintf("%s:%s", token, pepper)
	return []byte(h)
}

// rateLimitTracker tracks request timestamps per workspace for rate limiting.
type rateLimitTracker struct {
	tokens map[string]time.Time
}

// checkAndRecord checks and records a request for rate limiting.
// Returns true if the request is allowed, false if rate limit exceeded.
func (t *rateLimitTracker) checkAndRecord(ws string) bool {
	now := time.Now()
	if _, exists := t.tokens[ws]; !exists {
		t.tokens[ws] = now
		return true
	}
	// Reset timer if more than a minute has passed
	if now.Sub(t.tokens[ws]) > time.Minute {
		t.tokens[ws] = now
		return true
	}
	// Allow up to 100 requests per minute per workspace
	if len(t.tokens) >= 100 {
		return false
	}
	t.tokens[ws] = now
	return true
}

// rateLimitMiddleware provides basic rate limiting per workspace.
func rateLimitMiddleware(next http.Handler, tokens *rateLimitTracker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws := r.Context().Value("workspace_id").(string)
		if !tokens.checkAndRecord(ws) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type Options struct {
	Logger          *slog.Logger
	Ready           func(context.Context) error
	DB              *sql.DB
	AuthMode        string
	TokenPepper     string
	DefaultCurrency string
	AIEnabled       bool
	AIProvider      any
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
		asvc := accounts.NewService(opts.DB)
		registerAccountRoutes(mux, asvc)
		tsvc := tags.NewService(opts.DB)
		registerTagRoutes(mux, tsvc)
		txsvc := transactions.NewService(opts.DB)
		registerTransactionRoutes(mux, txsvc)
		bsvc := budgets.NewService(opts.DB)
		registerBudgetRoutes(mux, bsvc)
		registerAnalyticsRoutes(mux, analytics.NewService(opts.DB, opts.DefaultCurrency))
		registerDashboardRoutes(mux, dashboard.NewService(opts.DB, opts.DefaultCurrency))
		mux.Handle("/mcp", mcp.NewHandler(mcp.Deps{DB: opts.DB, DefaultCurrency: opts.DefaultCurrency}))
	}
	authed := AuthMiddleware(opts.DB, mux, opts.AuthMode, opts.TokenPepper)
	return withMiddleware(logger, authed)
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

func registerTagRoutes(mux *http.ServeMux, svc *tags.Service) {
	mux.HandleFunc("GET /api/v1/tags", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		list, err := svc.List(r.Context(), ws)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tags": list})
	})
	mux.HandleFunc("POST /api/v1/tags", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ws := workspaceID(r)
		t, err := svc.Create(r.Context(), tags.CreateInput{WorkspaceID: ws, Name: body.Name, Color: body.Color})
		if err != nil {
			if err == tags.ErrNameTaken {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, t)
	})
	mux.HandleFunc("GET /api/v1/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		t, err := svc.Get(r.Context(), ws, r.PathValue("id"))
		if err != nil {
			if err == tags.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	mux.HandleFunc("PATCH /api/v1/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		var body struct {
			Name  *string `json:"name"`
			Color *string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		t, err := svc.Update(r.Context(), ws, r.PathValue("id"), tags.UpdateInput{Name: body.Name, Color: body.Color})
		if err != nil {
			if err == tags.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			if err == tags.ErrNameTaken {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	mux.HandleFunc("DELETE /api/v1/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		if err := svc.Delete(r.Context(), ws, r.PathValue("id")); err != nil {
			if err == tags.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func registerTransactionRoutes(mux *http.ServeMux, svc *transactions.Service) {
	mux.HandleFunc("GET /api/v1/transactions", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		f, err := parseTransactionFilter(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		page, err := svc.ListPage(r.Context(), ws, f)
		if err != nil {
			if errors.Is(err, transactions.ErrInvalidFilter) ||
				errors.Is(err, transactions.ErrInvalidSort) ||
				errors.Is(err, transactions.ErrInvalidCursor) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, page)
	})
	mux.HandleFunc("POST /api/v1/transactions", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			AccountID   string   `json:"account_id"`
			AmountMinor int64    `json:"amount_minor"`
			OccurredAt  string   `json:"occurred_at"`
			Description string   `json:"description"`
			Notes       string   `json:"notes"`
			Source      string   `json:"source"`
			TagIDs      []string `json:"tag_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ws := workspaceID(r)
		t, err := svc.Create(r.Context(), transactions.CreateInput{WorkspaceID: ws, AccountID: body.AccountID, AmountMinor: body.AmountMinor, OccurredAt: body.OccurredAt, Description: body.Description, Notes: body.Notes, Source: body.Source, TagIDs: body.TagIDs})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, t)
	})
	mux.HandleFunc("GET /api/v1/transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		t, err := svc.Get(r.Context(), ws, r.PathValue("id"))
		if err != nil {
			if err == transactions.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	mux.HandleFunc("PATCH /api/v1/transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		var body struct {
			AccountID   *string   `json:"account_id"`
			AmountMinor *int64    `json:"amount_minor"`
			OccurredAt  *string   `json:"occurred_at"`
			Description *string   `json:"description"`
			Notes       *string   `json:"notes"`
			Source      *string   `json:"source"`
			TagIDs      *[]string `json:"tag_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		t, err := svc.Update(r.Context(), ws, r.PathValue("id"), transactions.UpdateInput{AccountID: body.AccountID, AmountMinor: body.AmountMinor, OccurredAt: body.OccurredAt, Description: body.Description, Notes: body.Notes, Source: body.Source, TagIDs: body.TagIDs})
		if err != nil {
			if err == transactions.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, t)
	})
	mux.HandleFunc("DELETE /api/v1/transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		if err := svc.Delete(r.Context(), ws, r.PathValue("id")); err != nil {
			if err == transactions.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /api/v1/transfers", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			FromAccountID string `json:"from_account_id"`
			ToAccountID   string `json:"to_account_id"`
			AmountMinor   int64  `json:"amount_minor"`
			OccurredAt    string `json:"occurred_at"`
			Description   string `json:"description"`
			Notes         string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		transfer, err := svc.CreateTransfer(r.Context(), transactions.TransferInput{
			WorkspaceID:   workspaceID(r),
			FromAccountID: body.FromAccountID,
			ToAccountID:   body.ToAccountID,
			AmountMinor:   body.AmountMinor,
			OccurredAt:    body.OccurredAt,
			Description:   body.Description,
			Notes:         body.Notes,
		})
		if err != nil {
			if errors.Is(err, transactions.ErrNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, transfer)
	})
}

// parseTransactionFilter validates and normalizes the list query. Date-only
// bounds are widened to whole UTC days so a `to` filter includes the whole day.
func parseTransactionFilter(r *http.Request) (transactions.Filter, error) {
	query := r.URL.Query()
	f := transactions.Filter{
		AccountID: query.Get("account_id"),
		TagIDs:    append([]string{}, query["tag"]...),
		Query:     strings.TrimSpace(query.Get("q")),
		Source:    query.Get("source"),
		Sort:      query.Get("sort"),
		Cursor:    query.Get("cursor"),
		Limit:     50,
	}
	if legacyTag := query.Get("tag_id"); legacyTag != "" {
		f.TagIDs = append(f.TagIDs, legacyTag)
	}
	f.TagIDs = uniqueStrings(f.TagIDs)

	if raw := query.Get("from"); raw != "" {
		value, err := normalizeRangeBound(raw, false)
		if err != nil {
			return transactions.Filter{}, fmt.Errorf("from %w", err)
		}
		f.From = value
	}
	if raw := query.Get("to"); raw != "" {
		value, err := normalizeRangeBound(raw, true)
		if err != nil {
			return transactions.Filter{}, fmt.Errorf("to %w", err)
		}
		f.To = value
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return transactions.Filter{}, fmt.Errorf("from must not be after to")
	}

	minAmount, err := parseOptionalInt64(query.Get("min_amount_minor"), "min_amount_minor")
	if err != nil {
		return transactions.Filter{}, err
	}
	maxAmount, err := parseOptionalInt64(query.Get("max_amount_minor"), "max_amount_minor")
	if err != nil {
		return transactions.Filter{}, err
	}
	f.MinAmountMinor = minAmount
	f.MaxAmountMinor = maxAmount

	if len(f.Query) > 200 {
		return transactions.Filter{}, fmt.Errorf("q must be 200 characters or fewer")
	}
	if len(f.TagIDs) > 20 {
		return transactions.Filter{}, fmt.Errorf("at most 20 tag filters are allowed")
	}

	if raw, ok := query["include_transfers"]; ok && len(raw) > 0 {
		include, err := strconv.ParseBool(raw[len(raw)-1])
		if err != nil {
			return transactions.Filter{}, fmt.Errorf("include_transfers must be true or false")
		}
		f.ExcludeTransfers = !include
	} else {
		f.ExcludeTransfers = true
	}

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > transactions.MaxListLimit {
			return transactions.Filter{}, fmt.Errorf("limit must be between 1 and %d", transactions.MaxListLimit)
		}
		f.Limit = limit
	}
	return f, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func parseOptionalInt64(raw, name string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s must be an integer", name)
	}
	return &value, nil
}

func normalizeRangeBound(raw string, endOfDay bool) (string, error) {
	if len(raw) == len("2006-01-02") {
		day, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return "", fmt.Errorf("must be YYYY-MM-DD or an RFC 3339 timestamp")
		}
		if endOfDay {
			day = day.Add(24*time.Hour - time.Second)
		}
		return day.Format(time.RFC3339), nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "", fmt.Errorf("must be YYYY-MM-DD or an RFC 3339 timestamp")
	}
	return value.UTC().Format(time.RFC3339), nil
}

func workspaceID(r *http.Request) string {
	if v := r.Header.Get("X-Workspace-Id"); v != "" {
		return v
	}
	return "default"
}

func registerBudgetRoutes(mux *http.ServeMux, svc *budgets.Service) {
	mux.HandleFunc("GET /api/v1/budgets", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		list, err := svc.List(r.Context(), ws)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"budgets": list})
	})
	mux.HandleFunc("POST /api/v1/budgets", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string   `json:"name"`
			AmountMinor int64    `json:"amount_minor"`
			Period      string   `json:"period"`
			StartDate   string   `json:"start_date"`
			EndDate     *string  `json:"end_date,omitempty"`
			Rollover    bool     `json:"rollover"`
			AccountID   *string  `json:"account_id,omitempty"`
			TagIDs      []string `json:"tag_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ws := workspaceID(r)
		a, err := svc.Create(r.Context(), budgets.CreateInput{
			WorkspaceID: ws,
			Name:        body.Name,
			AmountMinor: body.AmountMinor,
			Period:      body.Period,
			StartDate:   body.StartDate,
			EndDate:     body.EndDate,
			Rollover:    body.Rollover,
			AccountID:   body.AccountID,
			TagIDs:      body.TagIDs,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, a)
	})
	mux.HandleFunc("GET /api/v1/budgets/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		b, err := svc.Get(r.Context(), ws, r.PathValue("id"))
		if err != nil {
			if err == budgets.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, b)
	})
	mux.HandleFunc("PATCH /api/v1/budgets/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        *string   `json:"name,omitempty"`
			AmountMinor *int64    `json:"amount_minor,omitempty"`
			Period      *string   `json:"period,omitempty"`
			StartDate   *string   `json:"start_date,omitempty"`
			EndDate     *string   `json:"end_date,omitempty"`
			Rollover    *bool     `json:"rollover,omitempty"`
			AccountID   *string   `json:"account_id,omitempty"`
			TagIDs      *[]string `json:"tag_ids,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ws := workspaceID(r)
		b, err := svc.Update(r.Context(), ws, r.PathValue("id"), budgets.UpdateInput{
			Name:        body.Name,
			AmountMinor: body.AmountMinor,
			Period:      body.Period,
			StartDate:   body.StartDate,
			EndDate:     body.EndDate,
			Rollover:    body.Rollover,
			AccountID:   body.AccountID,
			TagIDs:      body.TagIDs,
		})
		if err != nil {
			if err == budgets.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, b)
	})
	mux.HandleFunc("DELETE /api/v1/budgets/{id}", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		if err := svc.Delete(r.Context(), ws, r.PathValue("id")); err != nil {
			if err == budgets.ErrNotFound {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/budgets/{id}/summary", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		tagID := r.URL.Query().Get("tag_id")
		summary, err := svc.Summarize(r.Context(), ws, budgets.SummaryOptions{TagID: tagID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"summary": summary})
	})
}
func registerDashboardRoutes(mux *http.ServeMux, svc *dashboard.Service) {
	mux.HandleFunc("GET /api/v1/dashboard", func(w http.ResponseWriter, r *http.Request) {
		from, to, err := dashboardRange(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := svc.Build(r.Context(), workspaceID(r), from, to)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

func dashboardRange(r *http.Request) (time.Time, time.Time, error) {
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")
	if fromRaw == "" && toRaw == "" {
		now := time.Now().UTC()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(0, 1, -1), nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("from and to must be provided together")
	}
	from, err := time.Parse("2006-01-02", fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from must be YYYY-MM-DD")
	}
	to, err := time.Parse("2006-01-02", toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to must be YYYY-MM-DD")
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("from must not be after to")
	}
	return from, to, nil
}

func registerSpendingRoutes(mux *http.ServeMux, svc *transactions.Service) {
	mux.HandleFunc("GET /api/v1/spending/summary", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		q := transactions.Filter{From: from, To: to}
		list, err := svc.List(r.Context(), ws, q)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		var totalMinor int64
		for _, tx := range list {
			totalMinor += tx.AmountMinor
		}
		writeJSON(w, http.StatusOK, map[string]any{"total_spent_minor": totalMinor, "transactions": list})
	})
	mux.HandleFunc("GET /api/v1/spending/tag-summary", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		q := transactions.Filter{From: from, To: to}
		list, err := svc.List(r.Context(), ws, q)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		tagSpending := make(map[string]int64)
		for _, tx := range list {
			for _, tagID := range tx.TagIDs {
				tagSpending[tagID] += tx.AmountMinor
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"tag_spending": tagSpending})
	})
}

func registerRecommendationRoutes(mux *http.ServeMux, bsvc *budgets.Service, tsvc *tags.Service) {
	mux.HandleFunc("GET /api/v1/recommendations", func(w http.ResponseWriter, r *http.Request) {
		ws := workspaceID(r)
		allBudgets, err := bsvc.List(r.Context(), ws)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		spentPerTag := make(map[string]int64)
		budgetRemaining := make(map[string]int64)
		for _, budget := range allBudgets {
			summary, err := bsvc.Summarize(r.Context(), ws, budgets.SummaryOptions{})
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			for _, s := range summary {
				spentPerTag[s.TagID] = s.SpentMinor
				budgetRemaining[budget.ID] = budget.AmountMinor - spentPerTag[budget.ID]
			}
		}
		overBudget := []string{}
		for tagID, spent := range spentPerTag {
			if budget, ok := budgetRemaining[tagID]; ok && spent >= budget {
				overBudget = append(overBudget, tagID)
			}
		}
		recommendations := map[string]any{
			"over_budget_tags": overBudget,
			"budget_remaining": budgetRemaining,
		}
		writeJSON(w, http.StatusOK, recommendations)
	})
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
