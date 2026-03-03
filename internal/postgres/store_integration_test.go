package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"baby-tracker-server/internal/postgres"
	"baby-tracker-server/internal/server"
)

func TestStoreListBabies(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db for setup: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate babies: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO babies (name) VALUES ($1), ($2)", "Alice", "Bob"); err != nil {
		t.Fatalf("failed to seed babies: %v", err)
	}

	got, err := store.ListBabies(ctx)
	if err != nil {
		t.Fatalf("failed to list babies: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 babies, got %d", len(got))
	}
	if got[0].Name != "Alice" {
		t.Fatalf("expected first baby Alice, got %q", got[0].Name)
	}
	if got[1].Name != "Bob" {
		t.Fatalf("expected second baby Bob, got %q", got[1].Name)
	}
}

func TestStoreChildCRUD(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db for setup: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate babies: %v", err)
	}

	created, err := store.CreateChild(ctx, server.CreateChildInput{
		Name:      "Alice",
		BirthDate: "2024-01-01",
		Gender:    "female",
	})
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created child id to be set")
	}

	children, err := store.ListChildren(ctx)
	if err != nil {
		t.Fatalf("failed to list children: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].BirthDate != "2024-01-01" {
		t.Fatalf("expected birth date 2024-01-01, got %q", children[0].BirthDate)
	}
	if children[0].Gender != "female" {
		t.Fatalf("expected gender female, got %q", children[0].Gender)
	}

	updated, err := store.UpdateChild(ctx, created.ID, server.UpdateChildInput{
		Name:      "Alice Updated",
		BirthDate: "2024-01-02",
		Gender:    "other",
	})
	if err != nil {
		t.Fatalf("failed to update child: %v", err)
	}
	if updated.Name != "Alice Updated" {
		t.Fatalf("expected updated name Alice Updated, got %q", updated.Name)
	}

	if err := store.DeleteChild(ctx, created.ID); err != nil {
		t.Fatalf("failed to delete child: %v", err)
	}

	if err := store.DeleteChild(ctx, created.ID); !errors.Is(err, server.ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting missing child, got %v", err)
	}
}
