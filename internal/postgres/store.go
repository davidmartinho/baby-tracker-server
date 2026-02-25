package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

func (s *Store) CreateEvent(ctx context.Context, babyID int64, input server.CreateEventInput) (server.Event, error) {
	const query = `
		INSERT INTO events (baby_id, type, start_time, end_time, side)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, baby_id, type, start_time, end_time, side
	`

	var (
		endTime sql.NullTime
		side    sql.NullString
	)
	if input.EndTime != nil {
		endTime = sql.NullTime{Time: *input.EndTime, Valid: true}
	}
	if input.Side != nil {
		side = sql.NullString{String: *input.Side, Valid: true}
	}

	var event server.Event
	var startTime time.Time
	if input.StartTime != nil {
		startTime = *input.StartTime
	}

	row := s.db.QueryRowContext(ctx, query, babyID, input.Type, startTime, endTime, side)
	var (
		gotEndTime sql.NullTime
		gotSide    sql.NullString
	)
	if err := row.Scan(&event.ID, &event.BabyID, &event.Type, &event.StartTime, &gotEndTime, &gotSide); err != nil {
		return server.Event{}, fmt.Errorf("insert event: %w", err)
	}

	if gotEndTime.Valid {
		event.EndTime = &gotEndTime.Time
	}
	if gotSide.Valid {
		event.Side = &gotSide.String
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
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NULL,
			side TEXT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}
