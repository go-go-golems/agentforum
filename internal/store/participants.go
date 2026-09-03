package store

import (
	"context"
	"fmt"
	"time"
)

// upsertParticipantTx records that an agent posted in a thread, bumping
// last_post_at on conflict. Used by both thread creation and post creation.
func upsertParticipantTx(ctx context.Context, tx dbtx, agentID, threadID string, at time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO participants (agent_id, thread_id, last_post_at)
VALUES (?,?,?)
ON CONFLICT(agent_id, thread_id) DO UPDATE SET last_post_at = excluded.last_post_at`,
		agentID, threadID, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: upsert participant: %w", err)
	}
	return nil
}

// IsParticipant reports whether an agent has participated in a thread.
func (s *Store) IsParticipant(ctx context.Context, agentID, threadID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM participants WHERE agent_id = ? AND thread_id = ?",
		agentID, threadID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("store: is participant: %w", err)
	}
	return n > 0, nil
}

// ListParticipantThreadIDs returns the threads an agent has participated in.
func (s *Store) ListParticipantThreadIDs(ctx context.Context, agentID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT thread_id FROM participants WHERE agent_id = ?", agentID)
	if err != nil {
		return nil, fmt.Errorf("store: list participant threads: %w", err)
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
