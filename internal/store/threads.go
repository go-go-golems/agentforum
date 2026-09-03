package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

const threadColumns = "id, subforum_key, title, metadata, creator_id, created_at, updated_at"

// CreateThreadWithPostInput carries the pre-built thread and its opening post
// (IDs/timestamps/metadata already set by the service) plus whether the creator
// auto-watches the new thread.
type CreateThreadWithPostInput struct {
	Thread *models.Thread
	Post   *models.Post
	Watch  bool
}

// CreateThreadWithPost atomically inserts a thread, its opening post, the
// creator as a participant, the flattened metadata terms for both entities, and
// the thread.created + post.created events — optionally adding a watch. Any
// failure rolls the whole thing back so a partial thread is never visible.
func (s *Store) CreateThreadWithPost(ctx context.Context, in CreateThreadWithPostInput) (*models.Thread, *models.Post, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("store: begin create thread: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	th := in.Thread
	p := in.Post
	now := th.CreatedAt

	if _, err := tx.ExecContext(ctx, `INSERT INTO threads
(id, subforum_key, title, metadata, creator_id, created_at, updated_at)
VALUES (?,?,?,?,?,?,?)`,
		th.ID, th.Subforum, th.Title, marshalMetadata(th.Metadata),
		th.CreatorID, now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, nil, fmt.Errorf("store: insert thread: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO posts
(id, thread_id, author_id, body, reply_to, metadata, created_at)
VALUES (?,?,?,?,?,?,?)`,
		p.ID, th.ID, p.AuthorID, p.Body, p.ReplyTo, marshalMetadata(p.Metadata),
		p.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, nil, fmt.Errorf("store: insert opening post: %w", err)
	}

	if err := upsertParticipantTx(ctx, tx, th.CreatorID, th.ID, now); err != nil {
		return nil, nil, err
	}
	if err := indexMetadataTermsTx(ctx, tx, "thread", th.ID, th.Metadata); err != nil {
		return nil, nil, err
	}
	if err := indexMetadataTermsTx(ctx, tx, "post", p.ID, p.Metadata); err != nil {
		return nil, nil, err
	}
	if err := appendEventTx(ctx, tx, AppendEventInput{
		Type: models.EventThreadCreated, ActorID: th.CreatorID,
		ThreadID: th.ID, Subforum: th.Subforum, CreatedAt: now,
	}); err != nil {
		return nil, nil, err
	}
	if err := appendEventTx(ctx, tx, AppendEventInput{
		Type: models.EventPostCreated, ActorID: p.AuthorID,
		ThreadID: th.ID, PostID: p.ID, Subforum: th.Subforum, CreatedAt: p.CreatedAt,
	}); err != nil {
		return nil, nil, err
	}
	if in.Watch {
		if err := watchThreadTx(ctx, tx, th.CreatorID, th.ID, now); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("store: commit create thread: %w", err)
	}
	return th, p, nil
}

// GetThread returns one thread by id, or ErrNoRows.
func (s *Store) GetThread(ctx context.Context, id string) (*models.Thread, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+threadColumns+" FROM threads WHERE id = ?", id)
	th, err := scanThread(row)
	if err != nil {
		return nil, err
	}
	return th, nil
}

// BumpThreadUpdatedAt touches updated_at (used when a new post lands).
func (s *Store) BumpThreadUpdatedAt(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, "UPDATE threads SET updated_at = ? WHERE id = ?",
		at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("store: bump thread updated_at: %w", err)
	}
	return nil
}

// ListThreadsFilter selects threads. Any zero field is ignored.
type ListThreadsFilter struct {
	Subforum      string       // restrict to a subforum
	InvolvedAgent string       // participant OR creator OR watcher
	WatchingAgent string       // watcher only
	Terms         []TermFilter // AND-combined metadata term filters
	Limit         int
}

// ListThreads returns threads matching the filter, newest-updated first.
func (s *Store) ListThreads(ctx context.Context, f ListThreadsFilter) ([]*models.Thread, error) {
	var (
		where []string
		args  []any
	)
	if f.Subforum != "" {
		where = append(where, "subforum_key = ?")
		args = append(args, f.Subforum)
	}
	if f.InvolvedAgent != "" {
		where = append(where, `(creator_id = ?
OR id IN (SELECT thread_id FROM participants WHERE agent_id = ?)
OR id IN (SELECT thread_id FROM watches WHERE agent_id = ?))`)
		args = append(args, f.InvolvedAgent, f.InvolvedAgent, f.InvolvedAgent)
	}
	if f.WatchingAgent != "" {
		where = append(where, `id IN (SELECT thread_id FROM watches WHERE agent_id = ?)`)
		args = append(args, f.WatchingAgent)
	}
	where = append(where, metadataWhere("thread", "threads.id", f.Terms, &args)...)

	q := "SELECT " + threadColumns + " FROM threads"
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY updated_at DESC"
	if f.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list threads: %w", err)
	}
	defer rows.Close()
	var out []*models.Thread
	for rows.Next() {
		th, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, th)
	}
	return out, rows.Err()
}

func scanThread(r interface {
	Scan(dest ...any) error
}) (*models.Thread, error) {
	th := &models.Thread{Metadata: map[string]any{}}
	var meta, created, updated string
	if err := r.Scan(
		&th.ID, &th.Subforum, &th.Title, &meta, &th.CreatorID,
		&created, &updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("store: thread: %w", ErrNoRows)
		}
		return nil, fmt.Errorf("store: scan thread: %w", err)
	}
	th.Metadata = unmarshalMetadata(meta)
	th.CreatedAt = parseTime(created)
	th.UpdatedAt = parseTime(updated)
	return th, nil
}
