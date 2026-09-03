package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

const postColumns = "id, thread_id, author_id, body, reply_to, metadata, created_at"

// CreatePostInput carries a pre-built post plus the thread's subforum (so the
// event row can be populated without an extra lookup inside the transaction).
type CreatePostInput struct {
	Post     *models.Post
	Subforum string
}

// CreatePost atomically inserts a post, upserts the author as a participant,
// reindexes the post's metadata terms, appends a post.created event, and bumps
// the thread's updated_at. Any failure rolls back.
func (s *Store) CreatePost(ctx context.Context, in CreatePostInput) (*models.Post, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: begin create post: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p := in.Post
	if _, err := tx.ExecContext(ctx, `INSERT INTO posts
(id, thread_id, author_id, body, reply_to, metadata, created_at)
VALUES (?,?,?,?,?,?,?)`,
		p.ID, p.ThreadID, p.AuthorID, p.Body, p.ReplyTo, marshalMetadata(p.Metadata),
		p.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, fmt.Errorf("store: insert post: %w", err)
	}
	if err := upsertParticipantTx(ctx, tx, p.AuthorID, p.ThreadID, p.CreatedAt); err != nil {
		return nil, err
	}
	if err := indexMetadataTermsTx(ctx, tx, "post", p.ID, p.Metadata); err != nil {
		return nil, err
	}
	if err := appendEventTx(ctx, tx, AppendEventInput{
		Type: models.EventPostCreated, ActorID: p.AuthorID,
		ThreadID: p.ThreadID, PostID: p.ID, Subforum: in.Subforum, CreatedAt: p.CreatedAt,
	}); err != nil {
		return nil, err
	}
	if err := tx.QueryRowContext(ctx,
		"UPDATE threads SET updated_at = ? WHERE id = ? RETURNING 1",
		p.CreatedAt.UTC().Format(time.RFC3339Nano), p.ThreadID).Scan(new(int)); err != nil {
		// RETURNING keeps the bump inside the tx; a missing row is a real error.
		return nil, fmt.Errorf("store: bump thread updated_at: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit create post: %w", err)
	}
	return p, nil
}

// GetPost returns one post by id, or ErrNoRows.
func (s *Store) GetPost(ctx context.Context, id string) (*models.Post, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+postColumns+" FROM posts WHERE id = ?", id)
	p, err := scanPost(row)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListPosts returns posts in a thread, optionally after a given post id, newest
// last, capped at limit (0 = unlimited). The "after" post must exist in thread.
func (s *Store) ListPosts(ctx context.Context, threadID, afterPostID string, limit int) ([]*models.Post, error) {
	q := "SELECT " + postColumns + " FROM posts WHERE thread_id = ?"
	args := []any{threadID}
	if afterPostID != "" {
		var afterAt string
		err := s.db.QueryRowContext(ctx,
			"SELECT created_at FROM posts WHERE id = ? AND thread_id = ?",
			afterPostID, threadID).Scan(&afterAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("store: after-post %q: %w", afterPostID, ErrNoRows)
			}
			return nil, fmt.Errorf("store: lookup after-post: %w", err)
		}
		q += " AND created_at > ?"
		args = append(args, afterAt)
	}
	q += " ORDER BY created_at ASC, id ASC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list posts: %w", err)
	}
	defer rows.Close()
	var out []*models.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanPost(r interface {
	Scan(dest ...any) error
}) (*models.Post, error) {
	p := &models.Post{Metadata: map[string]any{}}
	var meta, created string
	if err := r.Scan(
		&p.ID, &p.ThreadID, &p.AuthorID, &p.Body, &p.ReplyTo, &meta, &created,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("store: post: %w", ErrNoRows)
		}
		return nil, fmt.Errorf("store: scan post: %w", err)
	}
	p.Metadata = unmarshalMetadata(meta)
	p.CreatedAt = parseTime(created)
	return p, nil
}
