package postgres

import (
	"context"
	"database/sql"
	"errors"
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
		SELECT id, name, birth_date, gender
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
		var birthDate time.Time
		if err := rows.Scan(&b.ID, &b.Name, &birthDate, &b.Gender); err != nil {
			return nil, fmt.Errorf("scan baby: %w", err)
		}
		b.BirthDate = birthDate.Format("2006-01-02")
		data = append(data, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate babies: %w", err)
	}

	return data, nil
}

func (s *Store) CreateBaby(ctx context.Context, input server.CreateBabyInput) (server.Baby, error) {
	birthDate, err := time.Parse("2006-01-02", input.BirthDate)
	if err != nil {
		return server.Baby{}, fmt.Errorf("parse birthDate: %w", err)
	}

	const query = `
		INSERT INTO babies (name, birth_date, gender)
		VALUES ($1, $2, $3)
		RETURNING id, name, birth_date, gender
	`

	var baby server.Baby
	var dbBirthDate time.Time
	if err := s.db.QueryRowContext(ctx, query, input.Name, birthDate, input.Gender).Scan(
		&baby.ID,
		&baby.Name,
		&dbBirthDate,
		&baby.Gender,
	); err != nil {
		return server.Baby{}, fmt.Errorf("insert baby: %w", err)
	}
	baby.BirthDate = dbBirthDate.Format("2006-01-02")

	return baby, nil
}

func (s *Store) UpdateBaby(ctx context.Context, id int64, input server.UpdateBabyInput) (server.Baby, error) {
	birthDate, err := time.Parse("2006-01-02", input.BirthDate)
	if err != nil {
		return server.Baby{}, fmt.Errorf("parse birthDate: %w", err)
	}

	const query = `
		UPDATE babies
		SET name = $2, birth_date = $3, gender = $4
		WHERE id = $1
		RETURNING id, name, birth_date, gender
	`

	var baby server.Baby
	var dbBirthDate time.Time
	if err := s.db.QueryRowContext(ctx, query, id, input.Name, birthDate, input.Gender).Scan(
		&baby.ID,
		&baby.Name,
		&dbBirthDate,
		&baby.Gender,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return server.Baby{}, server.ErrNotFound
		}
		return server.Baby{}, fmt.Errorf("update baby: %w", err)
	}
	baby.BirthDate = dbBirthDate.Format("2006-01-02")

	return baby, nil
}

func (s *Store) DeleteBaby(ctx context.Context, id int64) error {
	const query = `DELETE FROM babies WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete baby: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete baby rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return server.ErrNotFound
	}

	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS babies (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			birth_date DATE,
			gender TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		ALTER TABLE babies ADD COLUMN IF NOT EXISTS birth_date DATE;
		ALTER TABLE babies ADD COLUMN IF NOT EXISTS gender TEXT;
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}
