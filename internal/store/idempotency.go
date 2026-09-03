package store

import (
	"context"
	"fmt"
	"time"
)

// IdempotencyRecord is a cached write result keyed by (key, agent_id) so a
// retried write collapses to the first response instead of creating duplicates.
type IdempotencyRecord struct {
	Key       string
	AgentID   string
	Entity    string // "thread" | "post"
	EntityID  string
	Response  string // cached JSON response
	CreatedAt time.Time
}

// GetIdempotencyRecord returns a cached record for (key, agentID), or
// ErrNoRows if none.
func (s *Store) GetIdempotencyRecord(ctx context.Context, key, agentID string) (*IdempotencyRecord, error) {
	r := &IdempotencyRecord{}
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT key, agent_id, entity, entity_id, response, created_at
FROM idempotency_keys WHERE key = ? AND agent_id = ?`, key, agentID).
		Scan(&r.Key, &r.AgentID, &r.Entity, &r.EntityID, &r.Response, &created)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("store: idempotency: %w", ErrNoRows)
		}
		return nil, fmt.Errorf("store: get idempotency: %w", err)
	}
	r.CreatedAt = parseTime(created)
	return r, nil
}

// SaveIdempotencyRecord caches a write result. The (key) is the primary key, so
// a collision (extremely unlikely for a random key) surfaces as a UNIQUE error.
func (s *Store) SaveIdempotencyRecord(ctx context.Context, r *IdempotencyRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO idempotency_keys
(key, agent_id, entity, entity_id, response, created_at) VALUES (?,?,?,?,?,?)`,
		r.Key, r.AgentID, r.Entity, r.EntityID, r.Response,
		r.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: save idempotency: %w", err)
	}
	return nil
}
