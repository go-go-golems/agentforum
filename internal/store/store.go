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
	// WAL mode lets readers run concurrently with the single writer; the pool
	// exists so long-poll reads (and the batches they trigger) stop queueing
	// behind each other while writes arrive. PRAGMAs live in the DSN, so every
	// pooled connection gets busy_timeout and foreign_keys on open
	// (journal_mode=WAL persists in the database file). Verified under 16
	// concurrent long-pollers + writes: see TestPollEventsConcurrentLongPollers
	// (AGENTFORUM-004 S1, risk R4).
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

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
//
// Each pragma is a separate _pragma parameter. url.Values.Set REPLACES the
// previous value for a key, so three Set calls on "_pragma" would leave only
// the last one (foreign_keys) in the DSN — which is exactly what shipped
// from AGENTFORUM-001 until AGENTFORUM-004 S2: the database silently ran in
// rollback-journal mode with no busy timeout, and concurrent readers made
// writers fail with SQLITE_BUSY (observed as unmapped 500s on POST while a
// long-poll or stream was open, including the AGENTFORUM-005 CI flake).
// Pinned by TestOpenPragmasApply.
func buildDSN(dbPath string) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
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
