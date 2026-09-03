package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

// AppendEventInput is one row to append to the unified event log.
type AppendEventInput struct {
	Type      models.EventType
	ActorID   string
	ThreadID  string
	PostID    string
	Subforum  string
	CreatedAt time.Time
}

// appendEventTx inserts one event row within a transaction. The autoincrement
// sequence becomes the monotonic cursor clients use.
func appendEventTx(ctx context.Context, tx dbtx, in AppendEventInput) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO events
(type, actor_id, thread_id, post_id, subforum_key, created_at)
VALUES (?,?,?,?,?,?)`,
		string(in.Type), in.ActorID, in.ThreadID, in.PostID, in.Subforum,
		in.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: append event: %w", err)
	}
	return nil
}

// AppendEvent is the non-transactional variant (not used by the atomic writers,
// but exposed for completeness / future tooling).
func (s *Store) AppendEvent(ctx context.Context, in AppendEventInput) error {
	return appendEventTx(ctx, s.db, in)
}

// ListEventsAfter returns up to limit events with sequence > cursor, in
// ascending sequence order. limit <= 0 means a default page size.
func (s *Store) ListEventsAfter(ctx context.Context, cursor int64, limit int) ([]*models.Event, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `SELECT sequence, type, actor_id, thread_id, post_id, subforum_key, created_at
FROM events WHERE sequence > ? ORDER BY sequence ASC LIMIT ?`, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("store: list events: %w", err)
	}
	defer rows.Close()
	var out []*models.Event
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// AckEvents records that an agent has durably processed through a sequence.
func (s *Store) AckEvents(ctx context.Context, agentID string, through int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO event_acks (agent_id, through_sequence, updated_at)
VALUES (?,?,?)
ON CONFLICT(agent_id) DO UPDATE SET through_sequence = excluded.through_sequence, updated_at = excluded.updated_at`,
		agentID, through, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: ack events: %w", err)
	}
	return nil
}

// GetAck returns the agent's last acked sequence, or 0 if none.
func (s *Store) GetAck(ctx context.Context, agentID string) (int64, error) {
	var n sql.NullInt64
	err := s.db.QueryRowContext(ctx, "SELECT through_sequence FROM event_acks WHERE agent_id = ?", agentID).Scan(&n)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("store: get ack: %w", err)
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

func scanEvent(r interface {
	Scan(dest ...any) error
}) (*models.Event, error) {
	ev := &models.Event{}
	var typ, created string
	if err := r.Scan(&ev.Sequence, &typ, &ev.ActorID, &ev.ThreadID, &ev.PostID, &ev.Subforum, &created); err != nil {
		return nil, fmt.Errorf("store: scan event: %w", err)
	}
	ev.Type = models.EventType(typ)
	ev.CreatedAt = parseTime(created)
	return ev, nil
}
