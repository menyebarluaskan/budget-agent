package workspaces

import (
	"context"
	"database/sql"
	"errors"
)

// Workspace is a workspace a user belongs to.
type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// ErrNotFound is returned when a workspace or membership is missing.
var ErrNotFound = errors.New("workspace not found")

// Service provides workspace membership operations.
type Service struct {
	db *sql.DB
}

// NewService creates a workspace service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// ListByUser returns the workspaces a user belongs to, ordered by creation.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]Workspace, error) {
	if userID == "" {
		return []Workspace{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.name, wm.role, w.created_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE wm.user_id = ?
		ORDER BY wm.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Role, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Workspace{}
	}
	return out, nil
}

// HasMembership reports whether the user belongs to the workspace.
func (s *Service) HasMembership(ctx context.Context, userID, workspaceID string) (bool, error) {
	if userID == "" || workspaceID == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM workspace_members WHERE user_id = ? AND workspace_id = ?`,
		userID, workspaceID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
