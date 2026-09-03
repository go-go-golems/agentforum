package store

import (
	"context"
	"fmt"
	"time"
)

// watchThreadTx idempotently records a thread watch inside a transaction.
func watchThreadTx(ctx context.Context, tx dbtx, agentID, threadID string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO watches
(agent_id, thread_id, created_at) VALUES (?,?,?)`,
		agentID, threadID, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: watch thread: %w", err)
	}
	return nil
}

// WatchThread subscribes an agent to a thread (idempotent).
func (s *Store) WatchThread(ctx context.Context, agentID, threadID string, at time.Time) error {
	return watchThreadTx(ctx, s.db, agentID, threadID, at)
}

// UnwatchThread removes a thread subscription (idempotent).
func (s *Store) UnwatchThread(ctx context.Context, agentID, threadID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM watches WHERE agent_id = ? AND thread_id = ?", agentID, threadID)
	if err != nil {
		return fmt.Errorf("store: unwatch thread: %w", err)
	}
	return nil
}

// IsWatchingThread reports whether an agent watches a thread.
func (s *Store) IsWatchingThread(ctx context.Context, agentID, threadID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM watches WHERE agent_id = ? AND thread_id = ?",
		agentID, threadID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("store: is watching thread: %w", err)
	}
	return n > 0, nil
}

// ListWatchingThreadIDs returns the threads an agent explicitly watches.
func (s *Store) ListWatchingThreadIDs(ctx context.Context, agentID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT thread_id FROM watches WHERE agent_id = ?", agentID)
	if err != nil {
		return nil, fmt.Errorf("store: list watching threads: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
