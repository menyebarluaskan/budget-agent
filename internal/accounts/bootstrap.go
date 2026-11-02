package accounts

import (
	"context"
	"database/sql"
)

// EnsureDefaultWorkspace creates a default user and workspace if none exist.
// Used until T-108 wires real identity/workspace mapping.
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
