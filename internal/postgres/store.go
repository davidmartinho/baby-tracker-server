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
		SELECT id, name, birth_date::text, gender
		FROM children
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
		if err := rows.Scan(&c.ID, &c.Name, &c.BirthDate, &c.Gender); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}
		data = append(data, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate children: %w", err)
	}

	return data, nil
}

func (s *Store) CreateChild(ctx context.Context, input server.CreateChildInput) (server.Child, error) {
	const query = `
		INSERT INTO children (name, birth_date, gender)
		VALUES ($1, $2, $3)
		RETURNING id, name, birth_date::text, gender
	`

	var c server.Child
	if err := s.db.QueryRowContext(ctx, query, input.Name, input.BirthDate, input.Gender).Scan(&c.ID, &c.Name, &c.BirthDate, &c.Gender); err != nil {
		return server.Child{}, fmt.Errorf("insert child: %w", err)
	}

	return c, nil
}

func (s *Store) UpdateChild(ctx context.Context, id int64, input server.UpdateChildInput) (server.Child, error) {
	const query = `
		UPDATE children
		SET name = $2,
			birth_date = $3,
			gender = $4
		WHERE id = $1
		RETURNING id, name, birth_date::text, gender
	`

	var c server.Child
	err := s.db.QueryRowContext(ctx, query, id, input.Name, input.BirthDate, input.Gender).Scan(&c.ID, &c.Name, &c.BirthDate, &c.Gender)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return server.Child{}, server.ErrChildNotFound
		}
		return server.Child{}, fmt.Errorf("update child: %w", err)
	}

	return c, nil
}

func (s *Store) DeleteChild(ctx context.Context, id int64) error {
	const query = `
		DELETE FROM children
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete child: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete child rows affected: %w", err)
	}

	if rows == 0 {
		return server.ErrChildNotFound
	}

	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	const ddl = `
		CREATE TABLE IF NOT EXISTS babies (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS children (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			birth_date DATE NOT NULL,
			gender TEXT NOT NULL CHECK (gender IN ('male', 'female', 'other')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	return nil
}
