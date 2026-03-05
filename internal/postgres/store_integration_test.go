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

	if _, err := db.ExecContext(ctx,
		"INSERT INTO babies (name, birth_date, gender) VALUES ($1, $2, $3), ($4, $5, $6)",
		"Alice", "2024-01-12", "female",
		"Bob", "2023-06-10", "male",
	); err != nil {
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
	if got[0].BirthDate != "2024-01-12" {
		t.Fatalf("expected first baby birth date 2024-01-12, got %q", got[0].BirthDate)
	}
	if got[0].Gender != "female" {
		t.Fatalf("expected first baby gender female, got %q", got[0].Gender)
	}
	if got[1].Name != "Bob" {
		t.Fatalf("expected second baby Bob, got %q", got[1].Name)
	}
	if got[1].BirthDate != "2023-06-10" {
		t.Fatalf("expected second baby birth date 2023-06-10, got %q", got[1].BirthDate)
	}
	if got[1].Gender != "male" {
		t.Fatalf("expected second baby gender male, got %q", got[1].Gender)
	}
}
