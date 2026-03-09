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

	store, db := mustSetupStoreAndDB(t, ctx, databaseURL)
	defer func() {
		_ = store.Close()
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

	store, db := mustSetupStoreAndDB(t, ctx, databaseURL)
	defer func() {
		_ = store.Close()
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE children RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate children: %v", err)
	}

	created, err := store.CreateChild(ctx, server.CreateChildInput{
		Name:      "Eve",
		BirthDate: "2024-01-02",
		Gender:    "female",
	})
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected created child id")
	}
	if created.Name != "Eve" || created.BirthDate != "2024-01-02" || created.Gender != "female" {
		t.Fatalf("unexpected created child: %+v", created)
	}

	listed, err := store.ListChildren(ctx)
	if err != nil {
		t.Fatalf("failed to list children: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 child, got %d", len(listed))
	}

	updated, err := store.UpdateChild(ctx, created.ID, server.UpdateChildInput{
		Name:      "Eva",
		BirthDate: "2024-01-03",
		Gender:    "female",
	})
	if err != nil {
		t.Fatalf("failed to update child: %v", err)
	}
	if updated.Name != "Eva" || updated.BirthDate != "2024-01-03" {
		t.Fatalf("unexpected updated child: %+v", updated)
	}

	if err := store.DeleteChild(ctx, created.ID); err != nil {
		t.Fatalf("failed to delete child: %v", err)
	}

	listed, err = store.ListChildren(ctx)
	if err != nil {
		t.Fatalf("failed to list children after delete: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected 0 children after delete, got %d", len(listed))
	}
}

func TestStoreUpdateChildNotFound(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, db := mustSetupStoreAndDB(t, ctx, databaseURL)
	defer func() {
		_ = store.Close()
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE children RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate children: %v", err)
	}

	_, err := store.UpdateChild(ctx, 999, server.UpdateChildInput{
		Name:      "Eve",
		BirthDate: "2024-01-02",
		Gender:    "female",
	})
	if !errors.Is(err, server.ErrChildNotFound) {
		t.Fatalf("expected ErrChildNotFound, got %v", err)
	}
}

func TestStoreDeleteChildNotFound(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, db := mustSetupStoreAndDB(t, ctx, databaseURL)
	defer func() {
		_ = store.Close()
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE children RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate children: %v", err)
	}

	err := store.DeleteChild(ctx, 999)
	if !errors.Is(err, server.ErrChildNotFound) {
		t.Fatalf("expected ErrChildNotFound, got %v", err)
	}
}

func mustSetupStoreAndDB(t *testing.T, ctx context.Context, databaseURL string) (*postgres.Store, *sql.DB) {
	t.Helper()

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("failed to open db for setup: %v", err)
	}

	return store, db
}
