package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/agentforum/internal/store"
)

// TestOpenAndMigrate creates the database, applies migrations, and confirms
// all domain tables exist. This is the P1 smoke test for the persistence layer.
func TestOpenAndMigrate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "agentforum.db")

	s, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	wantTables := []string{
		"agents", "subforums", "threads", "posts", "participants",
		"watches", "subforum_watches", "events", "event_acks",
		"metadata_terms", "idempotency_keys", "schema_migrations",
	}
	for _, tbl := range wantTables {
		var name string
		err := s.DB().QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q: %v", tbl, err)
		}
	}
}

// TestMigrateIdempotent reopens an existing database and asserts migrations
// are not reapplied (schema_migrations records are stable).
func TestMigrateIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "agentforum.db")

	for i := 0; i < 2; i++ {
		s, err := store.Open(ctx, dbPath)
		if err != nil {
			t.Fatalf("open #%d: %v", i+1, err)
		}
		var n int
		if err := s.DB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
			_ = s.Close()
			t.Fatalf("count migrations: %v", err)
		}
		_ = s.Close()
		if n == 0 {
			t.Fatalf("expected at least one applied migration")
		}
	}

	// Reopen once more and count again: should match the second open exactly.
	s, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("third open: %v", err)
	}
	defer func() { _ = s.Close() }()
	var n int
	if err := s.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 applied migration, got %d", n)
	}
}

// TestOpenCreatesParentDir verifies Open makes missing parent directories so
// a fresh AGENTFORUM_DB=/somewhere/new/x.db just works.
func TestOpenCreatesParentDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nested", "deep", "agentforum.db")
	s, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open with missing parent dir: %v", err)
	}
	defer func() { _ = s.Close() }()
	if err := s.DB().PingContext(ctx); err != nil {
		t.Fatalf("ping after create: %v", err)
	}
}
