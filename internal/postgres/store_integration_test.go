package postgres_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"baby-tracker-server/internal/postgres"
)

func TestStoreListBabies(t *testing.T) {
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

func TestStoreSeedsBabiesOnEmptyDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	got, err := store.ListBabies(ctx)
	if err != nil {
		t.Fatalf("failed to list babies: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 seeded babies, got %d", len(got))
	}
	if got[0].Name != "Alice" || got[1].Name != "Bob" || got[2].Name != "Charlie" {
		t.Fatalf("unexpected seed data: %#v", got)
	}
}
