package store

import (
	"context"
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
