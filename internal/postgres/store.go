package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"baby-tracker-server/internal/server"
)

type Store struct {
	db *sql.DB
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.seedBabies(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListBabies(ctx context.Context) ([]server.Baby, error) {
	const query = `
		SELECT id, name
		FROM babies
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query babies: %w", err)
	}
	defer rows.Close()

	data := make([]server.Baby, 0)
	for rows.Next() {
		var b server.Baby
		if err := rows.Scan(&b.ID, &b.Name); err != nil {
			return nil, fmt.Errorf("scan baby: %w", err)
		}
		data = append(data, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate babies: %w", err)
	}

	return data, nil
}

func (s *Store) CreateEvent(ctx context.Context, input server.CreateEventInput) (server.Event, error) {
	const query = `
		INSERT INTO events (baby_id, type, occurred_at, details)
		VALUES ($1, $2, $3, $4)
		RETURNING id, baby_id, type, occurred_at, details
	`

	var event server.Event
	if err := s.db.QueryRowContext(
		ctx,
		query,
		input.BabyID,
		input.Type,
		input.OccurredAt,
		input.Details,
	).Scan(
		&event.ID,
		&event.BabyID,
		&event.Type,
		&event.OccurredAt,
		&event.Details,
	); err != nil {
		return server.Event{}, fmt.Errorf("insert event: %w", err)
	}

	return event, nil
}

func (s *Store) migrate(ctx context.Context) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS babies (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS events (
			id BIGSERIAL PRIMARY KEY,
			baby_id BIGINT NOT NULL REFERENCES babies(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL,
			details JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS events_baby_id_idx ON events (baby_id);
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}

func (s *Store) seedBabies(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM babies").Scan(&count); err != nil {
		return fmt.Errorf("count babies: %w", err)
	}
	if count > 0 {
		return nil
	}

	const seedQuery = `
		INSERT INTO babies (name)
		VALUES ($1), ($2), ($3)
	`
	if _, err := s.db.ExecContext(ctx, seedQuery, "Alice", "Bob", "Charlie"); err != nil {
		return fmt.Errorf("seed babies: %w", err)
	}

	return nil
}
