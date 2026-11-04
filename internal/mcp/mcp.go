package mcp

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"budget-agent/internal/accounts"
	"budget-agent/internal/audit"
	"budget-agent/internal/budgets"
	"budget-agent/internal/dashboard"
	"budget-agent/internal/tags"
	"budget-agent/internal/transactions"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Deps holds the shared dependencies for the MCP server.
type Deps struct {
	DB              *sql.DB
	DefaultCurrency string
}

// NewHandler returns an http.Handler serving the MCP Streamable HTTP transport.
func NewHandler(deps Deps) http.Handler {
	return mcpSDK.NewStreamableHTTPHandler(func(r *http.Request) *mcpSDK.Server {
		ws := workspaceFromRequest(r)
		server := mcpSDK.NewServer(&mcpSDK.Implementation{Name: "budget-agent", Version: "v1"}, nil)
		registerTools(server, deps, ws)
		return server
	}, &mcpSDK.StreamableHTTPOptions{JSONResponse: true})
}

func workspaceFromRequest(r *http.Request) string {
	if v := r.Context().Value("workspace_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if v := r.Header.Get("X-Workspace-Id"); v != "" {
		return v
	}
	return "default"
}

func registerTools(server *mcpSDK.Server, deps Deps, ws string) {
	asvc := accounts.NewService(deps.DB)
	tsvc := tags.NewService(deps.DB)
	txsvc := transactions.NewService(deps.DB)
	bsvc := budgets.NewService(deps.DB)

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "list_accounts",
		Description: "List accounts with balances for the active workspace.",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in struct{}) (*mcpSDK.CallToolResult, map[string]any, error) {
		list, err := asvc.List(ctx, ws)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"accounts": list}, nil
	})

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "list_tags",
		Description: "List tags for the active workspace.",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in struct{}) (*mcpSDK.CallToolResult, map[string]any, error) {
		list, err := tsvc.List(ctx, ws)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"tags": list}, nil
	})

	type queryTxInput struct {
		AccountID string `json:"account_id,omitempty"`
		From      string `json:"from,omitempty"`
		To        string `json:"to,omitempty"`
		Tag       string `json:"tag,omitempty"`
		Query     string `json:"q,omitempty"`
		Source    string `json:"source,omitempty"`
		Limit     int    `json:"limit,omitempty"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "query_transactions",
		Description: "Query transactions with filters (bounded).",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in queryTxInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		f := transactions.Filter{
			AccountID: in.AccountID,
			From:      in.From,
			To:        in.To,
			Query:     in.Query,
			Source:    in.Source,
			Limit:     in.Limit,
		}
		if in.Tag != "" {
			f.TagIDs = []string{in.Tag}
		}
		if f.Limit <= 0 {
			f.Limit = 50
		}
		if f.Limit > 200 {
			f.Limit = 200
		}
		page, err := txsvc.ListPage(ctx, ws, f)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"transactions": page.Transactions, "next_cursor": page.NextCursor}, nil
	})

	type budgetSummaryInput struct {
		BudgetID string `json:"budget_id,omitempty"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "budget_summary",
		Description: "Budget progress for a period (all budgets if no id given).",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in budgetSummaryInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		if in.BudgetID != "" {
			b, err := bsvc.Get(ctx, ws, in.BudgetID)
			if err != nil {
				return nil, nil, err
			}
			sums, err := bsvc.Summarize(ctx, ws, budgets.SummaryOptions{})
			if err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{"budget": b, "summaries": sums}, nil
		}
		list, err := bsvc.List(ctx, ws)
		if err != nil {
			return nil, nil, err
		}
		sums, err := bsvc.Summarize(ctx, ws, budgets.SummaryOptions{})
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"budgets": list, "summaries": sums}, nil
	})

	type spendingSummaryInput struct {
		From string `json:"from,omitempty"`
		To   string `json:"to,omitempty"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "spending_summary",
		Description: "Dashboard summary (balance, spending, budget progress) over a range.",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in spendingSummaryInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		dsvc := dashboard.NewService(deps.DB, deps.DefaultCurrency)
		from, to := spendingRange(in.From, in.To)
		res, err := dsvc.Build(ctx, ws, from, to)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"dashboard": res}, nil
	})

	// Draft flow
	type createDraftInput struct {
		AccountID   string   `json:"account_id"`
		AmountMinor int64    `json:"amount_minor"`
		OccurredAt  string   `json:"occurred_at,omitempty"`
		Description string   `json:"description,omitempty"`
		Notes       string   `json:"notes,omitempty"`
		TagIDs      []string `json:"tag_ids,omitempty"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "create_transaction_draft",
		Description: "Validate and store a pending transaction draft (no write yet).",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in createDraftInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		payload, _ := json.Marshal(in)
		id := uuid.NewString()
		now := time.Now().UTC()
		expires := now.Add(30 * time.Minute)
		_, err := deps.DB.ExecContext(ctx, `
			INSERT INTO drafts (id, workspace_id, source, payload_json, status, created_at, expires_at)
			VALUES (?,?,?,?,?,?,?)`,
			id, ws, "mcp", string(payload), "pending", now.Format(time.RFC3339), expires.Format(time.RFC3339))
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"draft_id": id, "expires_at": expires.Format(time.RFC3339)}, nil
	})

	type draftActionInput struct {
		DraftID string `json:"draft_id"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "confirm_transaction_draft",
		Description: "Persist a pending draft as a transaction.",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in draftActionInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		var payload string
		var status string
		err := deps.DB.QueryRowContext(ctx, `SELECT payload_json, status FROM drafts WHERE id = ? AND workspace_id = ?`, in.DraftID, ws).Scan(&payload, &status)
		if err != nil {
			return nil, nil, fmt.Errorf("draft not found")
		}
		if status != "pending" {
			return nil, nil, fmt.Errorf("draft not pending")
		}
		var d createDraftInput
		if err := json.Unmarshal([]byte(payload), &d); err != nil {
			return nil, nil, err
		}
		if d.OccurredAt == "" {
			d.OccurredAt = time.Now().UTC().Format(time.RFC3339)
		}
		t, err := txsvc.Create(ctx, transactions.CreateInput{
			WorkspaceID: ws, AccountID: d.AccountID, AmountMinor: d.AmountMinor,
			OccurredAt: d.OccurredAt, Description: d.Description, Notes: d.Notes, TagIDs: d.TagIDs,
		})
		if err != nil {
			return nil, nil, err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		_, _ = deps.DB.ExecContext(ctx, `UPDATE drafts SET status='confirmed', confirmed_at=? WHERE id=?`, now, in.DraftID)
		_ = audit.Log(ctx, deps.DB, audit.Entry{WorkspaceID: ws, Actor: "mcp", Action: "draft.confirmed", EntityType: "transaction", EntityID: t.ID})
		return nil, map[string]any{"transaction": t}, nil
	})

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "reject_transaction_draft",
		Description: "Discard a pending draft.",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in draftActionInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		res, err := deps.DB.ExecContext(ctx, `UPDATE drafts SET status='rejected' WHERE id = ? AND workspace_id = ? AND status='pending'`, in.DraftID, ws)
		if err != nil {
			return nil, nil, err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return nil, nil, fmt.Errorf("draft not found or not pending")
		}
		_ = audit.Log(ctx, deps.DB, audit.Entry{WorkspaceID: ws, Actor: "mcp", Action: "draft.rejected", EntityType: "draft", EntityID: in.DraftID})
		return nil, map[string]any{"rejected": true}, nil
	})

	type extractReceiptInput struct {
		MimeType string `json:"mime_type"`
		DataB64  string `json:"data_base64"`
	}
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "extract_receipt",
		Description: "Store a receipt image and create a pending receipt draft (OCR deferred).",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, in extractReceiptInput) (*mcpSDK.CallToolResult, map[string]any, error) {
		raw, err := base64.StdEncoding.DecodeString(in.DataB64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid base64")
		}
		if len(raw) == 0 || len(raw) > 10*1024*1024 {
			return nil, nil, fmt.Errorf("invalid size")
		}
		id := uuid.NewString()
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = deps.DB.ExecContext(ctx, `
			INSERT INTO receipts (id, workspace_id, stored_path, mime_type, byte_size, sha256, ocr_status, created_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			id, ws, "", in.MimeType, len(raw), "", "pending", now)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"receipt_id": id, "ocr_status": "pending"}, nil
	})
}

func spendingRange(from, to string) (time.Time, time.Time) {
	if from == "" && to == "" {
		now := time.Now().UTC()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return from, from.AddDate(0, 1, -1)
	}
	f := parseDate(from)
	t := parseDate(to)
	if f.After(t) {
		f, t = t, f
	}
	return f, t
}

func parseDate(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now().UTC()
}
