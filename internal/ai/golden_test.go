package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseTransactionGolden(t *testing.T) {
	fix, err := os.ReadFile("testdata/coffee.json")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fix)
	}))
	defer srv.Close()

	p := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, Model: "test"})
	d, err := p.ParseTransaction(context.Background(), ParseInput{Text: "coffee 150"})
	if err != nil {
		t.Fatal(err)
	}
	if d.AmountMinor != -15000 || d.Description != "coffee" {
		t.Fatalf("unexpected: %+v", d)
	}
	if len(d.TagNames) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(d.TagNames))
	}
	b, _ := json.Marshal(d)
	if len(b) == 0 {
		t.Fatal("empty marshal")
	}
}
