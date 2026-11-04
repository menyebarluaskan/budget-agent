package accounts

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// EnsureDefaultWorkspace creates a default user and workspace if none exist.
// Used for dev mode (AUTH_MODE=dev) and tests; production tailnet users get a
// personal workspace via EnsurePersonalWorkspace on first request.
func EnsureDefaultWorkspace(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id, display_name, created_at) VALUES ('default','Default','2024-01-01T00:00:00Z')`); err != nil {
			return err
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, owner_user_id, created_at, updated_at) VALUES ('default','Default','default','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, created_at) VALUES ('default','default','owner','2024-01-01T00:00:00Z')`); err != nil {
			return err
		}
	}
	return nil
}

// EnsurePersonalWorkspaceUser upserts a user for the given Tailscale login and
// ensures they have at least one personal workspace, returning both the user ID
// and the workspace ID. Each distinct Tailscale user gets their own isolated
// workspace by default.
func EnsurePersonalWorkspaceUser(ctx context.Context, db *sql.DB, tailscaleLogin string) (string, string, error) {
	if tailscaleLogin == "" {
		return "", "", nil
	}
	now := time.Now().UTC().Format(time.RFC3339)

	// Upsert user by tailscale_login.
	var userID string
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE tailscale_login = ?`, tailscaleLogin).Scan(&userID)
	if err == sql.ErrNoRows {
		userID = uuid.NewString()
		if _, err := db.ExecContext(ctx, `INSERT INTO users (id, tailscale_login, display_name, created_at) VALUES (?,?,?,?)`,
			userID, tailscaleLogin, tailscaleLogin, now); err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}

	// Return an existing membership's workspace if any.
	var wsID string
	err = db.QueryRowContext(ctx, `
		SELECT wm.workspace_id FROM workspace_members wm
		WHERE wm.user_id = ? ORDER BY wm.created_at LIMIT 1`, userID).Scan(&wsID)
	if err == nil && wsID != "" {
		return userID, wsID, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}

	// Create a personal workspace.
	wsID = uuid.NewString()
	if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, owner_user_id, created_at, updated_at) VALUES (?,?,?,?,?)`,
		wsID, "Personal", userID, now, now); err != nil {
		return "", "", err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role, created_at) VALUES (?,?,?,?)`,
		wsID, userID, "owner", now); err != nil {
		return "", "", err
	}
	return userID, wsID, nil
}

// EnsurePersonalWorkspace upserts a user for the given Tailscale login and
// ensures they have at least one personal workspace, returning its ID. It is a
// thin wrapper over EnsurePersonalWorkspaceUser for callers that only need the
// workspace ID.
func EnsurePersonalWorkspace(ctx context.Context, db *sql.DB, tailscaleLogin string) (string, error) {
	_, wsID, err := EnsurePersonalWorkspaceUser(ctx, db, tailscaleLogin)
	return wsID, err
}
