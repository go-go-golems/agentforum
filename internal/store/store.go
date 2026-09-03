// Package store is the SQLite persistence layer for agentforum. It owns the
// *sql.DB, applies embedded migrations on Open, and exposes typed query
// methods grouped by entity (agents.go, subforums.go, …). The store knows
// nothing about tokens-as-auth or business rules; it returns models and
// errors, and the service layer decides what they mean (design doc §3.1).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps an open *sql.DB. Construct with Open; close with Close.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at dbPath, applies
// pending migrations, and returns a ready Store. If dbPath is empty a caller
// higher up should resolve a default (see DefaultDBPath).
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("store: db path is empty")
	}
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create db dir: %w", err)
		}
	}

	dsn := buildDSN(dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	// A single connection serializes writes cleanly under WAL and lets
	// migrations set PRAGMAs that persist on the connection. Long-poll reads
	// happen on their own queries and are fine under WAL.
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// buildDSN assembles the SQLite connection string with the pragmas the design
// doc calls for: WAL journaling, a 5s busy timeout, and foreign keys on.
func buildDSN(dbPath string) string {
	q := url.Values{}
	q.Set("_pragma", "journal_mode=WAL")
	q.Set("_pragma", "busy_timeout=5000")
	q.Set("_pragma", "foreign_keys=on")
	return "file:" + dbPath + "?" + q.Encode()
}

// DB exposes the underlying handle for advanced callers (e.g. the service
// layer running multi-statement transactions). Most code should prefer the
// typed methods on Store.
func (s *Store) DB() *sql.DB { return s.db }

// Close closes the database handle.
func (s *Store) Close() error { return s.db.Close() }

// parseTime parses an RFC3339Nano timestamp stored in the DB. A blank or bad
// value yields the zero time rather than an error, so reads never fail on a
// NULL-ish column.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
