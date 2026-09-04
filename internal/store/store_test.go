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

// TestOpenPoolSettings pins the R4 fix (AGENTFORUM-004 S1): the pool must
// stay raised so concurrent long-poll reads and arriving writes do not
// serialize on one connection. WAL readers do not block the writer; the
// DSN applies busy_timeout/foreign_keys to every pooled connection.
func TestOpenPoolSettings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "agentforum.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	stats := s.DB().Stats()
	if stats.MaxOpenConnections != 8 {
		t.Errorf("MaxOpenConnections = %d, want 8", stats.MaxOpenConnections)
	}
}

// TestOpenPragmasApply pins the DSN pragma bug (AGENTFORUM-004 S2): three
// url.Values.Set calls on the same "_pragma" key left only the last one,
// so the database ran in rollback-journal mode with no busy timeout from
// AGENTFORUM-001 onward — concurrent readers made writers fail with
// SQLITE_BUSY, surfacing as unmapped 500s while long-polls/streams were
// open (also the AGENTFORUM-005 CI flake's root cause).
func TestOpenPragmasApply(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "pragmas.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	for _, probe := range []struct {
		pragma string
		want   string
	}{
		{"journal_mode", "wal"},
		{"busy_timeout", "5000"},
		{"foreign_keys", "1"},
	} {
		var got string
		if err := s.DB().QueryRowContext(ctx, "PRAGMA "+probe.pragma).Scan(&got); err != nil {
			t.Fatalf("PRAGMA %s: %v", probe.pragma, err)
		}
		if got != probe.want {
			t.Errorf("PRAGMA %s = %s, want %s", probe.pragma, got, probe.want)
		}
	}
}
