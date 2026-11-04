package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIParseTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"account_name":"Wallet","amount_minor":-15000,"occurred_at":"","description":"coffee","notes":"","tag_names":["food"],"confidence":0.9}`}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, Model: "test", Timeout: 0})
	d, err := p.ParseTransaction(context.Background(), ParseInput{Text: "coffee 150"})
	if err != nil {
		t.Fatal(err)
	}
	if d.AmountMinor != -15000 || d.AccountName != "Wallet" {
		t.Fatalf("unexpected draft: %+v", d)
	}
}

func TestNewFromConfigDisabled(t *testing.T) {
	if p, ok := NewFromConfig(false, "http://x", "k", "m", 0); ok || p != nil {
		t.Fatal("expected disabled")
	}
	if p, ok := NewFromConfig(true, "", "k", "m", 0); ok || p != nil {
		t.Fatal("expected disabled without base url")
	}
}
