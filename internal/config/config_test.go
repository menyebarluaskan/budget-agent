package config

import (
	"log/slog"
	"strings"
	"testing"
)

func testEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_PATH": "/data/budget.db",
		"RECEIPTS_DIR":  "/data/receipts",
		"BASE_URL":      "https://budget-agent.tailnet.ts.net",
		"TOKEN_PEPPER":  "test-pepper",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom(testEnv(validEnv()))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Addr != "127.0.0.1:8080" {
		t.Errorf("Addr = %q, want default", cfg.Addr)
	}
	if cfg.AuthMode != "tailnet" {
		t.Errorf("AuthMode = %q, want tailnet", cfg.AuthMode)
	}
	if cfg.DefaultCurrency != "IDR" {
		t.Errorf("DefaultCurrency = %q, want IDR", cfg.DefaultCurrency)
	}
	if cfg.Timezone != "UTC" {
		t.Errorf("Timezone = %q, want UTC", cfg.Timezone)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.AIEnabled {
		t.Error("AIEnabled = true, want false")
	}
	if cfg.WhatsAppEnabled {
		t.Error("WhatsAppEnabled = true, want false")
	}
}

func TestLoadAcceptsExplicitValues(t *testing.T) {
	values := validEnv()
	values["APP_ADDR"] = "0.0.0.0:9090"
	values["AUTH_MODE"] = "dev"
	values["DEFAULT_CURRENCY"] = "USD"
	values["TZ"] = "Asia/Jakarta"
	values["LOG_LEVEL"] = "debug"

	cfg, err := LoadFrom(testEnv(values))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Addr != "0.0.0.0:9090" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.AuthMode != "dev" {
		t.Errorf("AuthMode = %q", cfg.AuthMode)
	}
	if cfg.DefaultCurrency != "USD" {
		t.Errorf("DefaultCurrency = %q", cfg.DefaultCurrency)
	}
	if cfg.Timezone != "Asia/Jakarta" {
		t.Errorf("Timezone = %q", cfg.Timezone)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v", cfg.LogLevel)
	}
}

func TestLoadRequiresCoreValues(t *testing.T) {
	for _, key := range []string{"DATABASE_PATH", "RECEIPTS_DIR", "BASE_URL", "TOKEN_PEPPER"} {
		t.Run(key, func(t *testing.T) {
			values := validEnv()
			delete(values, key)

			_, err := LoadFrom(testEnv(values))
			if err == nil {
				t.Fatal("LoadFrom() error = nil, want error")
			}
			if !strings.Contains(err.Error(), key) {
				t.Fatalf("error = %q, want mention of %s", err, key)
			}
		})
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		errPart string
	}{
		{name: "base URL scheme", key: "BASE_URL", value: "ftp://example.com", errPart: "http or https"},
		{name: "base URL host", key: "BASE_URL", value: "https://", errPart: "host"},
		{name: "auth mode", key: "AUTH_MODE", value: "password", errPart: "AUTH_MODE"},
		{name: "currency length", key: "DEFAULT_CURRENCY", value: "ID", errPart: "DEFAULT_CURRENCY"},
		{name: "currency case", key: "DEFAULT_CURRENCY", value: "idr", errPart: "DEFAULT_CURRENCY"},
		{name: "timezone", key: "TZ", value: "Mars/Olympus", errPart: "TZ"},
		{name: "log level", key: "LOG_LEVEL", value: "verbose", errPart: "LOG_LEVEL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := validEnv()
			values[tt.key] = tt.value

			_, err := LoadFrom(testEnv(values))
			if err == nil {
				t.Fatalf("LoadFrom() error = nil, want %q", tt.errPart)
			}
			if !strings.Contains(err.Error(), tt.errPart) {
				t.Fatalf("error = %q, want mention of %q", err, tt.errPart)
			}
		})
	}
}

func TestLoadValidatesAIWhenEnabled(t *testing.T) {
	values := validEnv()
	values["AI_ENABLED"] = "true"

	_, err := LoadFrom(testEnv(values))
	if err == nil || !strings.Contains(err.Error(), "AI_BASE_URL") {
		t.Fatalf("error = %v, want AI_BASE_URL requirement", err)
	}

	values["AI_BASE_URL"] = "https://ai.example.com/v1"
	_, err = LoadFrom(testEnv(values))
	if err == nil || !strings.Contains(err.Error(), "AI_MODEL") {
		t.Fatalf("error = %v, want AI_MODEL requirement", err)
	}

	values["AI_MODEL"] = "test-model"
	cfg, err := LoadFrom(testEnv(values))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if !cfg.AIEnabled {
		t.Error("AIEnabled = false, want true")
	}
}

func TestLoadValidatesWhatsAppWhenEnabled(t *testing.T) {
	values := validEnv()
	values["WHATSAPP_ENABLED"] = "true"

	required := []string{
		"WHATSAPP_VERIFY_TOKEN",
		"WHATSAPP_APP_SECRET",
		"WHATSAPP_ACCESS_TOKEN",
		"WHATSAPP_PHONE_NUMBER_ID",
		"WHATSAPP_ALLOWED_NUMBERS",
	}
	for _, key := range required {
		t.Run(key, func(t *testing.T) {
			_, err := LoadFrom(testEnv(values))
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("error = %v, want %s requirement", err, key)
			}
			values[key] = "test-value"
		})
	}

	values["WHATSAPP_ALLOWED_NUMBERS"] = "+628111111111, +628222222222"
	cfg, err := LoadFrom(testEnv(values))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if !cfg.WhatsAppEnabled {
		t.Error("WhatsAppEnabled = false, want true")
	}
	if len(cfg.WhatsAppAllowedNumbers) != 2 {
		t.Fatalf("allowed numbers = %v, want 2 entries", cfg.WhatsAppAllowedNumbers)
	}
}

func TestLoadRejectsInvalidBoolean(t *testing.T) {
	values := validEnv()
	values["AI_ENABLED"] = "sometimes"

	_, err := LoadFrom(testEnv(values))
	if err == nil || !strings.Contains(err.Error(), "AI_ENABLED") {
		t.Fatalf("error = %v, want AI_ENABLED boolean requirement", err)
	}
}
