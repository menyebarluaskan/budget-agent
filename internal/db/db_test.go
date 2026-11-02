package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "budget.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	// idempotent
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate second: %v", err)
	}

	// foreign keys enabled
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	// table exists
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='workspaces'").Scan(&name)
	if err != nil {
		t.Fatalf("workspaces table: %v", err)
	}

	// workspace scoping: insert two workspaces and an account in each; query scoped.
	if _, err := db.Exec(`INSERT INTO users (id, display_name, created_at) VALUES ('u1','a','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspaces (id, name, owner_user_id, created_at, updated_at) VALUES ('w1','W1','u1','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspaces (id, name, owner_user_id, created_at, updated_at) VALUES ('w2','W2','u1','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (id, workspace_id, name, type, currency, created_at, updated_at) VALUES ('a1','w1','cash','wallet','IDR','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts (id, workspace_id, name, type, currency, created_at, updated_at) VALUES ('a2','w2','cash','wallet','IDR','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE workspace_id = ?`, "w1").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("w1 account count = %d, want 1", count)
	}

	// FK enforcement: account referencing missing workspace fails.
	_, err = db.Exec(`INSERT INTO accounts (id, workspace_id, name, type, currency, created_at, updated_at) VALUES ('a3','nope','x','wallet','IDR','2024-01-01T00:00:00Z','2024-01-01T00:00:00Z')`)
	if err == nil {
		t.Fatal("expected FK violation, got nil")
	}
	_ = sql.ErrNoRows
	_ = os.Args
}
