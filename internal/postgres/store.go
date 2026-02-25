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

func (s *Store) CreateEvent(ctx context.Context, event server.Event) (server.Event, error) {
	const query = `
		INSERT INTO events (baby_id, type, side, occurred_at, started_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, baby_id, type, side, occurred_at, started_at, ended_at
	`

	var created server.Event
	var side sql.NullString
	var occurredAt sql.NullTime
	var startedAt sql.NullTime
	var endedAt sql.NullTime

	row := s.db.QueryRowContext(
		ctx,
		query,
		event.BabyID,
		event.Type,
		toNullString(event.Side),
		toNullTime(event.OccurredAt),
		toNullTime(event.StartedAt),
		toNullTime(event.EndedAt),
	)
	if err := row.Scan(
		&created.ID,
		&created.BabyID,
		&created.Type,
		&side,
		&occurredAt,
		&startedAt,
		&endedAt,
	); err != nil {
		return server.Event{}, fmt.Errorf("insert event: %w", err)
	}

	if side.Valid {
		created.Side = side.String
	}
	if occurredAt.Valid {
		value := occurredAt.Time
		created.OccurredAt = &value
	}
	if startedAt.Valid {
		value := startedAt.Time
		created.StartedAt = &value
	}
	if endedAt.Valid {
		value := endedAt.Time
		created.EndedAt = &value
	}

	return created, nil
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
			side TEXT,
			occurred_at TIMESTAMPTZ,
			started_at TIMESTAMPTZ,
			ended_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT events_type_check CHECK (type IN ('diaper', 'nursing', 'sleep')),
			CONSTRAINT events_side_check CHECK (type <> 'nursing' OR side IN ('left', 'right')),
			CONSTRAINT events_time_check CHECK (
				(type = 'diaper' AND occurred_at IS NOT NULL AND started_at IS NULL AND ended_at IS NULL)
				OR (type IN ('nursing', 'sleep') AND started_at IS NOT NULL AND ended_at IS NOT NULL AND occurred_at IS NULL)
			),
			CONSTRAINT events_range_check CHECK (started_at IS NULL OR started_at <= ended_at)
		)
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}

func toNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func toNullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
