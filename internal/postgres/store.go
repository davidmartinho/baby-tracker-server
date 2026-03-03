package postgres

import (
	"context"
	"database/sql"
	"errors"
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

func (s *Store) ListChildren(ctx context.Context) ([]server.Child, error) {
	const query = `
		SELECT id, name, birth_date, gender
		FROM babies
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query children: %w", err)
	}
	defer rows.Close()

	data := make([]server.Child, 0)
	for rows.Next() {
		var c server.Child
		var birthDate string
		if err := rows.Scan(&c.ID, &c.Name, &birthDate, &c.Gender); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}
		c.BirthDate = birthDate
		data = append(data, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate children: %w", err)
	}

	return data, nil
}

func (s *Store) CreateChild(ctx context.Context, input server.CreateChildInput) (server.Child, error) {
	const query = `
		INSERT INTO babies (name, birth_date, gender)
		VALUES ($1, $2::date, $3)
		RETURNING id, name, birth_date::text, gender
	`

	var created server.Child
	if err := s.db.QueryRowContext(ctx, query, input.Name, input.BirthDate, input.Gender).
		Scan(&created.ID, &created.Name, &created.BirthDate, &created.Gender); err != nil {
		return server.Child{}, fmt.Errorf("insert child: %w", err)
	}

	return created, nil
}

func (s *Store) UpdateChild(ctx context.Context, id int64, input server.UpdateChildInput) (server.Child, error) {
	const query = `
		UPDATE babies
		SET name = $2,
			birth_date = $3::date,
			gender = $4
		WHERE id = $1
		RETURNING id, name, birth_date::text, gender
	`

	var updated server.Child
	if err := s.db.QueryRowContext(ctx, query, id, input.Name, input.BirthDate, input.Gender).
		Scan(&updated.ID, &updated.Name, &updated.BirthDate, &updated.Gender); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return server.Child{}, server.ErrNotFound
		}
		return server.Child{}, fmt.Errorf("update child: %w", err)
	}

	return updated, nil
}

func (s *Store) DeleteChild(ctx context.Context, id int64) error {
	const query = `DELETE FROM babies WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete child: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete child rows affected: %w", err)
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
			birth_date DATE NOT NULL DEFAULT CURRENT_DATE,
			gender TEXT NOT NULL DEFAULT 'unknown',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	const alterBirthDateDDL = `
		ALTER TABLE babies
		ADD COLUMN IF NOT EXISTS birth_date DATE NOT NULL DEFAULT CURRENT_DATE
	`
	if _, err := s.db.ExecContext(ctx, alterBirthDateDDL); err != nil {
		return fmt.Errorf("migrate birth_date column: %w", err)
	}

	const alterGenderDDL = `
		ALTER TABLE babies
		ADD COLUMN IF NOT EXISTS gender TEXT NOT NULL DEFAULT 'unknown'
	`
	if _, err := s.db.ExecContext(ctx, alterGenderDDL); err != nil {
		return fmt.Errorf("migrate gender column: %w", err)
	}

	return nil
}
