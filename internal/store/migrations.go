package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migrate applies every embedded migration that has not yet been recorded in
// schema_migrations, in filename order. Each migration runs in its own
// transaction so a failure leaves the database at the last good state.
func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    name       TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("store: ensure schema_migrations: %w", err)
	}

	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("store: glob migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := s.isMigrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}
		if err := s.applyMigration(ctx, name, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) isMigrationApplied(ctx context.Context, name string) (bool, error) {
	var n string
	err := s.db.QueryRowContext(ctx,
		"SELECT name FROM schema_migrations WHERE name = ?", name).Scan(&n)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("store: check migration %s: %w", name, err)
	default:
		return true, nil
	}
}

func (s *Store) applyMigration(ctx context.Context, name, body string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin tx for %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("store: apply migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)",
		name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("store: record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit migration %s: %w", name, err)
	}
	return nil
}

// MigrationBaseName is the bare filename of a migration path, for diagnostics.
func MigrationBaseName(path string) string { return filepath.Base(path) }
