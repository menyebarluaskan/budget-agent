// Package config loads and validates the process configuration from the
// environment. It fails closed: a missing or malformed value is an error rather
// than a half-configured process.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr     = "127.0.0.1:8080"
	defaultAuthMode = "tailnet"
	defaultCurrency = "IDR"
	defaultTimezone = "UTC"
)

// Config is the validated process configuration.
type Config struct {
	Addr         string
	DatabasePath string
	ReceiptsDir  string
	BaseURL      string
	AuthMode     string
	TokenPepper  string

	AIEnabled     bool
	AIBaseURL     string
	AIAPIKey      string
	AIModel       string
	AIVisionModel string

	WhatsAppEnabled        bool
	WhatsAppVerifyToken    string
	WhatsAppAppSecret      string
	WhatsAppAccessToken    string
	WhatsAppPhoneNumberID  string
	WhatsAppAllowedNumbers []string

	DefaultCurrency string
	Timezone        string
	LogLevel        slog.Level
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom reads configuration using getenv and validates it. It exists so the
// loader can be tested without changing the process environment.
func LoadFrom(getenv func(string) string) (Config, error) {
	read := func(key string) string { return strings.TrimSpace(getenv(key)) }

	cfg := Config{
		Addr:            valueOrDefault(read("APP_ADDR"), defaultAddr),
		DatabasePath:    read("DATABASE_PATH"),
		ReceiptsDir:     read("RECEIPTS_DIR"),
		BaseURL:         read("BASE_URL"),
		AuthMode:        valueOrDefault(read("AUTH_MODE"), defaultAuthMode),
		TokenPepper:     getenv("TOKEN_PEPPER"),
		AIBaseURL:       read("AI_BASE_URL"),
		AIAPIKey:        getenv("AI_API_KEY"),
		AIModel:         read("AI_MODEL"),
		AIVisionModel:   read("AI_VISION_MODEL"),
		DefaultCurrency: valueOrDefault(read("DEFAULT_CURRENCY"), defaultCurrency),
		Timezone:        valueOrDefault(read("TZ"), defaultTimezone),
		LogLevel:        slog.LevelInfo,
	}

	if err := require("DATABASE_PATH", cfg.DatabasePath); err != nil {
		return Config{}, err
	}
	if err := require("RECEIPTS_DIR", cfg.ReceiptsDir); err != nil {
		return Config{}, err
	}
	if err := require("BASE_URL", cfg.BaseURL); err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.TokenPepper) == "" {
		return Config{}, fmt.Errorf("TOKEN_PEPPER is required")
	}

	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return Config{}, err
	}
	if err := validateAuthMode(cfg.AuthMode); err != nil {
		return Config{}, err
	}
	if err := validateCurrency(cfg.DefaultCurrency); err != nil {
		return Config{}, err
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Config{}, fmt.Errorf("TZ: %w", err)
	}

	level, err := parseLogLevel(read("LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if cfg.AIEnabled, err = parseBool("AI_ENABLED", read("AI_ENABLED")); err != nil {
		return Config{}, err
	}
	if cfg.AIEnabled {
		if cfg.AIBaseURL == "" {
			return Config{}, fmt.Errorf("AI_BASE_URL is required when AI_ENABLED is true")
		}
		if cfg.AIModel == "" {
			return Config{}, fmt.Errorf("AI_MODEL is required when AI_ENABLED is true")
		}
	}

	if cfg.WhatsAppEnabled, err = parseBool("WHATSAPP_ENABLED", read("WHATSAPP_ENABLED")); err != nil {
		return Config{}, err
	}
	if cfg.WhatsAppEnabled {
		cfg.WhatsAppVerifyToken = getenv("WHATSAPP_VERIFY_TOKEN")
		cfg.WhatsAppAppSecret = getenv("WHATSAPP_APP_SECRET")
		cfg.WhatsAppAccessToken = getenv("WHATSAPP_ACCESS_TOKEN")
		cfg.WhatsAppPhoneNumberID = read("WHATSAPP_PHONE_NUMBER_ID")
		cfg.WhatsAppAllowedNumbers = splitCSV(read("WHATSAPP_ALLOWED_NUMBERS"))

		if cfg.WhatsAppVerifyToken == "" {
			return Config{}, fmt.Errorf("WHATSAPP_VERIFY_TOKEN is required when WHATSAPP_ENABLED is true")
		}
		if cfg.WhatsAppAppSecret == "" {
			return Config{}, fmt.Errorf("WHATSAPP_APP_SECRET is required when WHATSAPP_ENABLED is true")
		}
		if cfg.WhatsAppAccessToken == "" {
			return Config{}, fmt.Errorf("WHATSAPP_ACCESS_TOKEN is required when WHATSAPP_ENABLED is true")
		}
		if cfg.WhatsAppPhoneNumberID == "" {
			return Config{}, fmt.Errorf("WHATSAPP_PHONE_NUMBER_ID is required when WHATSAPP_ENABLED is true")
		}
		if len(cfg.WhatsAppAllowedNumbers) == 0 {
			return Config{}, fmt.Errorf("WHATSAPP_ALLOWED_NUMBERS is required when WHATSAPP_ENABLED is true")
		}
	}

	return cfg, nil
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func require(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("BASE_URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("BASE_URL must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("BASE_URL must include a host")
	}
	return nil
}

func validateAuthMode(mode string) error {
	switch mode {
	case "tailnet", "dev":
		return nil
	default:
		return fmt.Errorf("AUTH_MODE must be tailnet or dev")
	}
}

func validateCurrency(code string) error {
	if len(code) != 3 {
		return fmt.Errorf("DEFAULT_CURRENCY must be a three-letter ISO 4217 code")
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("DEFAULT_CURRENCY must be a three-letter ISO 4217 code")
		}
	}
	return nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(valueOrDefault(strings.TrimSpace(raw), "info")) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

func parseBool(name, raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", name, err)
	}
	return value, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}
