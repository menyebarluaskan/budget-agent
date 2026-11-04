package audit

import (
	"context"
	"path/filepath"
	"testing"

	"budget-agent/internal/db"
)

func TestLogInsertsRow(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "budget.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.Migrate(database); err != nil {
		t.Fatal(err)
	}
	// ensure default workspace exists
	if _, err := database.Exec(`INSERT INTO users (id, display_name, created_at) VALUES ('u','u','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO workspaces (id, name, owner_user_id, created_at, updated_at) VALUES ('w','w','u','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := Log(ctx, database, Entry{WorkspaceID: "w", Actor: "ai", Action: "draft.confirmed", EntityType: "transaction", EntityID: "tx1", Metadata: map[string]any{"source": "ai"}}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE workspace_id='w'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
