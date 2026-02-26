package postgres_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"baby-tracker-server/internal/postgres"
	"baby-tracker-server/internal/server"
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

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE events, babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
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

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE events, babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
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

func TestStoreCreateEvent(t *testing.T) {
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

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE events, babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO babies (name) VALUES ($1)", "Mila"); err != nil {
		t.Fatalf("failed to seed baby: %v", err)
	}

	details, err := json.Marshal(map[string]any{"notes": "first change"})
	if err != nil {
		t.Fatalf("failed to build details: %v", err)
	}

	occurredAt := time.Now().UTC().Truncate(time.Second)
	got, err := store.CreateEvent(ctx, server.CreateEventInput{
		BabyID:     1,
		Type:       "diaper",
		OccurredAt: occurredAt,
		Details:    details,
	})
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	if got.ID == 0 {
		t.Fatal("expected event id to be set")
	}
	if got.BabyID != 1 {
		t.Fatalf("expected baby id 1, got %d", got.BabyID)
	}
	if got.Type != "diaper" {
		t.Fatalf("expected type diaper, got %q", got.Type)
	}
}
