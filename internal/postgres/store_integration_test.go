package postgres_test

import (
	"context"
	"database/sql"
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

func TestStoreCreateEvent(t *testing.T) {
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

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE events RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("failed to truncate events: %v", err)
	}

	var babyID int64
	if err := db.QueryRowContext(ctx, "INSERT INTO babies (name) VALUES ($1) RETURNING id", "Charlie").Scan(&babyID); err != nil {
		t.Fatalf("failed to seed baby: %v", err)
	}

	startedAt := time.Now().UTC().Truncate(time.Second)
	created, err := store.CreateEvent(ctx, server.Event{
		BabyID:    babyID,
		Type:      server.EventTypeDiaperChange,
		StartedAt: startedAt,
	})
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected created event id to be set")
	}
	if created.BabyID != babyID {
		t.Fatalf("expected baby id %d, got %d", babyID, created.BabyID)
	}
	if created.Type != server.EventTypeDiaperChange {
		t.Fatalf("expected type diaper_change, got %s", created.Type)
	}
	if !created.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %s, got %s", startedAt, created.StartedAt)
	}
}
