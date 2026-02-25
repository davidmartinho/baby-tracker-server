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

	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE events RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate events: %v", err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE babies RESTART IDENTITY"); err != nil {
		t.Fatalf("failed to truncate babies: %v", err)
	}

	var babyID int64
	if err := db.QueryRowContext(ctx, "INSERT INTO babies (name) VALUES ($1) RETURNING id", "Charlie").Scan(&babyID); err != nil {
		t.Fatalf("failed to seed baby: %v", err)
	}

	t.Run("diaper", func(t *testing.T) {
		occurredAt := time.Date(2026, 2, 25, 9, 15, 0, 0, time.UTC)
		created, err := store.CreateEvent(ctx, server.Event{
			BabyID:     babyID,
			Type:       "diaper",
			OccurredAt: &occurredAt,
		})
		if err != nil {
			t.Fatalf("failed to create diaper event: %v", err)
		}
		if created.ID == 0 {
			t.Fatalf("expected event id to be set")
		}
		if created.Type != "diaper" {
			t.Fatalf("expected type diaper, got %q", created.Type)
		}
		if created.OccurredAt == nil || !created.OccurredAt.Equal(occurredAt) {
			t.Fatalf("expected occurred_at %v, got %v", occurredAt, created.OccurredAt)
		}
	})

	t.Run("nursing", func(t *testing.T) {
		startedAt := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
		endedAt := time.Date(2026, 2, 25, 10, 20, 0, 0, time.UTC)
		created, err := store.CreateEvent(ctx, server.Event{
			BabyID:    babyID,
			Type:      "nursing",
			Side:      "right",
			StartedAt: &startedAt,
			EndedAt:   &endedAt,
		})
		if err != nil {
			t.Fatalf("failed to create nursing event: %v", err)
		}
		if created.Type != "nursing" {
			t.Fatalf("expected type nursing, got %q", created.Type)
		}
		if created.Side != "right" {
			t.Fatalf("expected side right, got %q", created.Side)
		}
		if created.StartedAt == nil || !created.StartedAt.Equal(startedAt) {
			t.Fatalf("expected started_at %v, got %v", startedAt, created.StartedAt)
		}
		if created.EndedAt == nil || !created.EndedAt.Equal(endedAt) {
			t.Fatalf("expected ended_at %v, got %v", endedAt, created.EndedAt)
		}
	})

	t.Run("sleep", func(t *testing.T) {
		startedAt := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
		endedAt := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
		created, err := store.CreateEvent(ctx, server.Event{
			BabyID:    babyID,
			Type:      "sleep",
			StartedAt: &startedAt,
			EndedAt:   &endedAt,
		})
		if err != nil {
			t.Fatalf("failed to create sleep event: %v", err)
		}
		if created.Type != "sleep" {
			t.Fatalf("expected type sleep, got %q", created.Type)
		}
		if created.StartedAt == nil || !created.StartedAt.Equal(startedAt) {
			t.Fatalf("expected started_at %v, got %v", startedAt, created.StartedAt)
		}
		if created.EndedAt == nil || !created.EndedAt.Equal(endedAt) {
			t.Fatalf("expected ended_at %v, got %v", endedAt, created.EndedAt)
		}
	})
}
