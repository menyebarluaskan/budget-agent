package tags

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
}

type CreateInput struct {
	WorkspaceID string
	Name        string
	Color       string
}

type UpdateInput struct {
	Name  *string
	Color *string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

var ErrNotFound = errors.New("not found")

func (s *Service) List(ctx context.Context, workspaceID string) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, name, color, created_at FROM tags WHERE workspace_id = ? ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Tag{}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Tag, error) {
	var t Tag
	err := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, color, created_at FROM tags WHERE id = ? AND workspace_id = ?`, id, workspaceID).Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Color, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tag{}, ErrNotFound
		}
		return Tag{}, err
	}
	return t, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Tag, error) {
	if in.Name == "" {
		return Tag{}, fmt.Errorf("name required")
	}
	id := uuid.NewString()
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO tags (id, workspace_id, name, color, created_at) VALUES (?,?,?,?,?)`, id, in.WorkspaceID, in.Name, in.Color, now)
	if err != nil {
		return Tag{}, err
	}
	return s.Get(ctx, in.WorkspaceID, id)
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (Tag, error) {
	t, err := s.Get(ctx, workspaceID, id)
	if err != nil {
		return Tag{}, err
	}
	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.Color != nil {
		t.Color = *in.Color
	}
	_, err = s.db.ExecContext(ctx, `UPDATE tags SET name=?, color=? WHERE id=? AND workspace_id=?`, t.Name, t.Color, id, workspaceID)
	if err != nil {
		return Tag{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}