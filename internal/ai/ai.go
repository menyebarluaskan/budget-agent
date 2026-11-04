package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ParseInput is the input to ParseTransaction.
type ParseInput struct {
	Text      string   `json:"text"`
	Accounts  []string `json:"accounts,omitempty"`
	TagNames  []string `json:"tag_names,omitempty"`
	Currency  string   `json:"currency,omitempty"`
}

// TransactionDraft is the structured output of ParseTransaction.
type TransactionDraft struct {
	AccountName  string   `json:"account_name"`
	AmountMinor  int64    `json:"amount_minor"`
	OccurredAt   string   `json:"occurred_at,omitempty"`
	Description  string   `json:"description,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	TagNames     []string `json:"tag_names,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
}

// ReceiptDraft is the structured output of ExtractReceipt.
type ReceiptDraft struct {
	Merchant   string   `json:"merchant,omitempty"`
	OccurredAt string   `json:"occurred_at,omitempty"`
	TotalMinor int64    `json:"total_minor"`
	TaxMinor   int64    `json:"tax_minor,omitempty"`
	TagNames   []string `json:"tag_names,omitempty"`
	LineItems  []string `json:"line_items,omitempty"`
}

// Provider is the AI abstraction.
type Provider interface {
	ParseTransaction(ctx context.Context, in ParseInput) (TransactionDraft, error)
	ExtractReceipt(ctx context.Context, image []byte, mime string) (ReceiptDraft, error)
}

// OpenAIConfig configures the OpenAI-compatible adapter.
type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// OpenAI is an OpenAI-compatible provider.
type OpenAI struct {
	cfg    OpenAIConfig
	client *http.Client
}

// NewOpenAI constructs an OpenAI-compatible provider.
func NewOpenAI(cfg OpenAIConfig) *OpenAI {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &OpenAI{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const parseSystemPrompt = `You are a budgeting assistant. Given a short text describing a purchase or income, extract a single transaction. Respond ONLY with a JSON object matching this schema: {"account_name": string, "amount_minor": integer (signed: negative for expense, positive for income), "occurred_at": ISO8601 string or "", "description": string, "notes": string, "tag_names": [string], "confidence": number 0-1}. Do not include any other text.`

func (o *OpenAI) ParseTransaction(ctx context.Context, in ParseInput) (TransactionDraft, error) {
	userMsg := in.Text
	if len(in.Accounts) > 0 {
		userMsg += "\nAvailable accounts: " + join(in.Accounts)
	}
	if len(in.TagNames) > 0 {
		userMsg += "\nAvailable tags: " + join(in.TagNames)
	}
	if in.Currency != "" {
		userMsg += "\nCurrency: " + in.Currency
	}

	reqBody := chatRequest{
		Model: o.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: parseSystemPrompt},
			{Role: "user", Content: userMsg},
		},
		ResponseFormat: map[string]any{
			"type": "json_object",
		},
	}
	b, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.BaseURL+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return TransactionDraft{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return TransactionDraft{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return TransactionDraft{}, fmt.Errorf("ai request failed: %s: %s", resp.Status, string(body))
	}
	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return TransactionDraft{}, err
	}
	if cr.Error != nil {
		return TransactionDraft{}, fmt.Errorf("ai error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return TransactionDraft{}, fmt.Errorf("no choices returned")
	}
	var draft TransactionDraft
	if err := json.Unmarshal([]byte(cr.Choices[0].Message.Content), &draft); err != nil {
		return TransactionDraft{}, fmt.Errorf("invalid ai output: %w", err)
	}
	return draft, nil
}

func (o *OpenAI) ExtractReceipt(ctx context.Context, image []byte, mime string) (ReceiptDraft, error) {
	// Deferred to T-304; return not-implemented for now.
	return ReceiptDraft{}, fmt.Errorf("receipt extraction not implemented")
}

// NewFromConfig builds a provider from config. Returns nil,false if AI is disabled.
func NewFromConfig(enabled bool, baseURL, apiKey, model string, timeout time.Duration) (Provider, bool) {
	if !enabled || baseURL == "" || model == "" {
		return nil, false
	}
	return NewOpenAI(OpenAIConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Timeout: timeout,
	}), true
}


func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
