package workspaces

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Workspace is a workspace a user belongs to.
type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// Member is a user's membership in a workspace.
type Member struct {
	UserID      string `json:"user_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

// Invite is a created invite (plaintext token returned only at creation).
type Invite struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// ErrForbidden is returned when the caller lacks permission.
var ErrForbidden = errors.New("forbidden")

// ErrNotFound is returned when a workspace or membership is missing.
var ErrNotFound = errors.New("workspace not found")

// ErrInvalidInvite is returned for an invalid/expired invite token.
var ErrInvalidInvite = errors.New("invalid or expired invite")

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

// Role returns the user's role in the workspace.
func (s *Service) Role(ctx context.Context, userID, workspaceID string) (string, error) {
	var role string
	err := s.db.QueryRowContext(ctx, `
		SELECT role FROM workspace_members WHERE user_id = ? AND workspace_id = ?`,
		userID, workspaceID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

// IsOwner reports whether the user is an owner of the workspace.
func (s *Service) IsOwner(ctx context.Context, userID, workspaceID string) (bool, error) {
	role, err := s.Role(ctx, userID, workspaceID)
	if err != nil {
		return false, err
	}
	return role == "owner", nil
}

// ListMembers returns members of a workspace.
func (s *Service) ListMembers(ctx context.Context, workspaceID string) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.tailscale_login, u.display_name, wm.role, wm.created_at
		FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = ?
		ORDER BY wm.created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		var login sql.NullString
		if err := rows.Scan(&m.UserID, &login, &m.DisplayName, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		if login.Valid {
			m.Login = login.String
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Member{}
	}
	return out, nil
}

// CreateInvite creates an invite token for a workspace. Caller must be owner.
func (s *Service) CreateInvite(ctx context.Context, workspaceID, createdBy string, ttl time.Duration) (Invite, error) {
	owner, err := s.IsOwner(ctx, createdBy, workspaceID)
	if err != nil {
		return Invite{}, err
	}
	if !owner {
		return Invite{}, ErrForbidden
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return Invite{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	now := time.Now().UTC()
	expires := now.Add(ttl).UTC().Format(time.RFC3339)
	id := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO workspace_invites (id, workspace_id, token_hash, created_by, expires_at, created_at)
		VALUES (?,?,?,?,?,?)`, id, workspaceID, tokenHash, createdBy, expires, now.Format(time.RFC3339))
	if err != nil {
		return Invite{}, err
	}
	return Invite{Token: token, ExpiresAt: expires}, nil
}

// RedeemInvite adds the calling user as a member of the invite's workspace.
func (s *Service) RedeemInvite(ctx context.Context, token, userID string) (string, error) {
	if token == "" || userID == "" {
		return "", ErrInvalidInvite
	}
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	var inviteID, workspaceID, expiresAt string
	var redeemedBy sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, expires_at, redeemed_by FROM workspace_invites WHERE token_hash = ?`,
		tokenHash).Scan(&inviteID, &workspaceID, &expiresAt, &redeemedBy)
	if err == sql.ErrNoRows {
		return "", ErrInvalidInvite
	}
	if err != nil {
		return "", err
	}
	exp, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(exp) {
		return "", ErrInvalidInvite
	}
	if redeemedBy.Valid && redeemedBy.String != userID {
		return "", ErrInvalidInvite
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO workspace_members (workspace_id, user_id, role, created_at)
		VALUES (?,?,?,?)`, workspaceID, userID, "member", now)
	if err != nil {
		return "", err
	}
	if !redeemedBy.Valid {
		_, err = s.db.ExecContext(ctx, `
			UPDATE workspace_invites SET redeemed_by = ?, redeemed_at = ? WHERE id = ?`,
			userID, now, inviteID)
		if err != nil {
			return "", err
		}
	}
	return workspaceID, nil
}

// RemoveMember removes a member from a workspace. Owner can remove others;
// a member can remove themselves (leave). Owners cannot be removed.
func (s *Service) RemoveMember(ctx context.Context, workspaceID, actorID, targetUserID string) error {
	actorRole, err := s.Role(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}
	targetRole, err := s.Role(ctx, targetUserID, workspaceID)
	if err != nil {
		return err
	}
	if targetRole == "owner" {
		return ErrForbidden
	}
	if actorID != targetUserID && actorRole != "owner" {
		return ErrForbidden
	}
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?`,
		workspaceID, targetUserID)
	return err
}

// RenameWorkspace renames a workspace (owner only).
func (s *Service) RenameWorkspace(ctx context.Context, workspaceID, actorID, name string) error {
	owner, err := s.IsOwner(ctx, actorID, workspaceID)
	if err != nil {
		return err
	}
	if !owner {
		return ErrForbidden
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx, `UPDATE workspaces SET name = ?, updated_at = ? WHERE id = ?`,
		name, now, workspaceID)
	return err
}

// WorkspaceName returns the workspace name.
func (s *Service) WorkspaceName(ctx context.Context, workspaceID string) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM workspaces WHERE id = ?`, workspaceID).Scan(&name)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return name, nil
}
