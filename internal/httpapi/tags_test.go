package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTagRejectsDuplicateName(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tags", strings.NewReader(`{"name":" food ","color":"#ef4444"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestUpdateTagRejectsDuplicateName(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tags/transport", strings.NewReader(`{"name":"FOOD"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}
