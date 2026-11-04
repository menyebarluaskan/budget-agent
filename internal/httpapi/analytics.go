package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"budget-agent/internal/analytics"
	"budget-agent/internal/budgets"
)

func registerAnalyticsRoutes(mux *http.ServeMux, svc *analytics.Service) {
	mux.HandleFunc("GET /api/v1/analytics/spending", func(w http.ResponseWriter, r *http.Request) {
		from, to, err := dashboardRange(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		months, err := parseBoundedQuery(r, "months", analytics.DefaultTrendMonths, analytics.MaxTrendMonths)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := svc.Spending(r.Context(), workspaceID(r), from, to, months)
		if err != nil {
			if errors.Is(err, analytics.ErrInvalidRange) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /api/v1/budgets/{id}/trend", func(w http.ResponseWriter, r *http.Request) {
		asOf := time.Now().UTC()
		if raw := r.URL.Query().Get("to"); raw != "" {
			parsed, err := time.Parse("2006-01-02", raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
				return
			}
			asOf = parsed
		}
		periods, err := parseBoundedQuery(r, "periods", analytics.DefaultTrendMonths, analytics.MaxTrendMonths)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		result, err := svc.BudgetTrend(r.Context(), workspaceID(r), r.PathValue("id"), asOf, periods)
		if err != nil {
			if errors.Is(err, budgets.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

// parseBoundedQuery parses an optional positive integer query parameter.
func parseBoundedQuery(r *http.Request, name string, fallback, max int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > max {
		return 0, errors.New(name + " must be between 1 and " + strconv.Itoa(max))
	}
	return value, nil
}
