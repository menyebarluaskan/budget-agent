package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Entry describes one audit record.
type Entry struct {
	WorkspaceID string
	Actor       string
	Action      string
	EntityType  string
	EntityID    string
	Metadata    map[string]any
}

// Log inserts an audit record.
func Log(ctx context.Context, db *sql.DB, e Entry) error {
	meta := "{}"
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			meta = string(b)
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_log (id, workspace_id, actor, action, entity_type, entity_id, metadata_json, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), e.WorkspaceID, e.Actor, e.Action, e.EntityType, e.EntityID, meta,
		time.Now().UTC().Format(time.RFC3339))
	return err
}
