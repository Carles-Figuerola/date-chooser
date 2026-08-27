// Package store provides SQLite-backed persistence for Date Chooser polls.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; registers itself as "sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store wraps a *sql.DB configured for the Date Chooser schema.
type Store struct {
	db *sql.DB
}

// Open opens (and, on a fresh volume, creates) the SQLite database at
// dbPath, enables foreign key enforcement, and applies the embedded,
// idempotent schema so the database self-initializes on first run.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: opening database: %w", err)
	}

	// modernc.org/sqlite does not support concurrent writers on a single
	// connection pool without contention; a single connection keeps this
	// v1 deployment (small groups, low write volume) simple and correct.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: enabling foreign keys: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: applying schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping verifies the database is reachable (used by GET /healthz).
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// DB exposes the underlying *sql.DB for callers (e.g. tests) that need
// direct query access beyond the Store's higher-level methods.
func (s *Store) DB() *sql.DB {
	return s.db
}
