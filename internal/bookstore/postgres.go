package bookstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

type PostgresStore struct{ db *sql.DB }

func OpenPostgres(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error                   { return s.db.Close() }
func (s *PostgresStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Initialize is safe to rerun: schema creation and seed insertion are idempotent.
func (s *PostgresStore) Initialize(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS books (
		id BIGINT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
		updated_at TIMESTAMPTZ NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create books table: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO books (id, title, author, price_cents, updated_at) VALUES
		(1, 'The Go Programming Language', 'Alan A. A. Donovan and Brian W. Kernighan', 3999, '2024-01-01T00:00:00Z'),
		(2, 'Designing Data-Intensive Applications', 'Martin Kleppmann', 4599, '2024-01-01T00:00:00Z'),
		(3, 'Database Internals', 'Alex Petrov', 4299, '2024-01-01T00:00:00Z')
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("seed books: %w", err)
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (Book, error) {
	var b Book
	err := s.db.QueryRowContext(ctx, `SELECT id, title, author, price_cents, updated_at FROM books WHERE id = $1`, id).
		Scan(&b.ID, &b.Title, &b.Author, &b.PriceCents, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Book{}, ErrNotFound
	}
	if err != nil {
		return Book{}, err
	}
	return b, nil
}

func (s *PostgresStore) Update(ctx context.Context, id int64, input BookUpdate) (Book, error) {
	var b Book
	err := s.db.QueryRowContext(ctx, `UPDATE books SET title = $2, author = $3, price_cents = $4, updated_at = NOW()
		WHERE id = $1 RETURNING id, title, author, price_cents, updated_at`, id, input.Title, input.Author, input.PriceCents).
		Scan(&b.ID, &b.Title, &b.Author, &b.PriceCents, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Book{}, ErrNotFound
	}
	if err != nil {
		return Book{}, err
	}
	return b, nil
}
