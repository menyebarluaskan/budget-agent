package tags

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"budget-agent/internal/accounts"
	"budget-agent/internal/db"
)

func setupTagsTest(t *testing.T) *Service {
	t.Helper()

	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "budget.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := accounts.EnsureDefaultWorkspace(ctx, database); err != nil {
		t.Fatalf("bootstrap workspace: %v", err)
	}
	return NewService(database)
}

func TestCreateTrimsName(t *testing.T) {
	ctx := context.Background()
	svc := setupTagsTest(t)

	tag, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "  Food  ", Color: "#ef4444"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if tag.Name != "Food" {
		t.Errorf("name = %q, want Food", tag.Name)
	}
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := setupTagsTest(t)

	if _, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "Food", Color: "#ef4444"}); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	for _, name := range []string{"Food", "food", "  FOOD  "} {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: name})
			if !errors.Is(err, ErrNameTaken) {
				t.Fatalf("error = %v, want ErrNameTaken", err)
			}
		})
	}
}

func TestUpdateRejectsDuplicateName(t *testing.T) {
	ctx := context.Background()
	svc := setupTagsTest(t)

	if _, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "Food"}); err != nil {
		t.Fatalf("create food: %v", err)
	}
	transport, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "Transport"})
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}

	name := " food "
	_, err = svc.Update(ctx, "default", transport.ID, UpdateInput{Name: &name})
	if !errors.Is(err, ErrNameTaken) {
		t.Fatalf("error = %v, want ErrNameTaken", err)
	}
}

func TestUpdateAllowsCaseOnlyRenameOfSameTag(t *testing.T) {
	ctx := context.Background()
	svc := setupTagsTest(t)

	tag, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "Food"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	name := "food"
	updated, err := svc.Update(ctx, "default", tag.ID, UpdateInput{Name: &name})
	if err != nil {
		t.Fatalf("update tag: %v", err)
	}
	if updated.Name != "food" {
		t.Errorf("name = %q, want food", updated.Name)
	}
}

func TestUpdateTrimsNameAndRecolors(t *testing.T) {
	ctx := context.Background()
	svc := setupTagsTest(t)

	tag, err := svc.Create(ctx, CreateInput{WorkspaceID: "default", Name: "Food", Color: "#ef4444"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	name := " Groceries "
	color := "#22c55e"
	updated, err := svc.Update(ctx, "default", tag.ID, UpdateInput{Name: &name, Color: &color})
	if err != nil {
		t.Fatalf("update tag: %v", err)
	}
	if updated.Name != "Groceries" {
		t.Errorf("name = %q, want Groceries", updated.Name)
	}
	if updated.Color != color {
		t.Errorf("color = %q, want %q", updated.Color, color)
	}
}
