package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

// CreateSubforum inserts a new subforum. Key uniqueness is enforced by the
// schema; a violation surfaces as a SQLite UNIQUE error the service maps.
func (s *Store) CreateSubforum(ctx context.Context, sf *models.Subforum) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO subforums
(key, title, description, metadata, creator_id, created_at, updated_at)
VALUES (?,?,?,?,?,?,?)`,
		sf.Key, sf.Title, sf.Description, marshalMetadata(sf.Metadata),
		sf.CreatorID, sf.CreatedAt.UTC().Format(time.RFC3339Nano),
		sf.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: create subforum: %w", err)
	}
	return nil
}

// ListSubforums returns all subforums ordered by key.
func (s *Store) ListSubforums(ctx context.Context) ([]*models.Subforum, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+subforumColumns+" FROM subforums ORDER BY key ASC")
	if err != nil {
		return nil, fmt.Errorf("store: list subforums: %w", err)
	}
	defer rows.Close()
	var out []*models.Subforum
	for rows.Next() {
		sf, err := scanSubforum(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sf)
	}
	return out, rows.Err()
}

// GetSubforum returns one subforum by key.
func (s *Store) GetSubforum(ctx context.Context, key string) (*models.Subforum, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT "+subforumColumns+" FROM subforums WHERE key = ?", key)
	sf, err := scanSubforum(row)
	if err != nil {
		return nil, err
	}
	return sf, nil
}

// UpdateSubforum patches mutable subforum fields and bumps updated_at.
func (s *Store) UpdateSubforum(ctx context.Context, sf *models.Subforum) error {
	_, err := s.db.ExecContext(ctx, `UPDATE subforums SET
title=?, description=?, metadata=?, updated_at=? WHERE key=?`,
		sf.Title, sf.Description, marshalMetadata(sf.Metadata),
		sf.UpdatedAt.UTC().Format(time.RFC3339Nano), sf.Key)
	if err != nil {
		return fmt.Errorf("store: update subforum: %w", err)
	}
	return nil
}

const subforumColumns = "key, title, description, metadata, creator_id, created_at, updated_at"

func scanSubforum(r interface {
	Scan(dest ...any) error
}) (*models.Subforum, error) {
	sf := &models.Subforum{Metadata: map[string]any{}}
	var meta, created, updated string
	if err := r.Scan(
		&sf.Key, &sf.Title, &sf.Description, &meta, &sf.CreatorID,
		&created, &updated,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("store: subforum: %w", ErrNoRows)
		}
		return nil, fmt.Errorf("store: scan subforum: %w", err)
	}
	sf.Metadata = unmarshalMetadata(meta)
	sf.CreatedAt = parseTime(created)
	sf.UpdatedAt = parseTime(updated)
	return sf, nil
}

// --- subforum watches ---------------------------------------------------

// WatchSubforum idempotently records an agent's subscription to a subforum.
func (s *Store) WatchSubforum(ctx context.Context, agentID, key string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO subforum_watches
(agent_id, subforum_key, created_at) VALUES (?,?,?)`,
		agentID, key, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: watch subforum: %w", err)
	}
	return nil
}

// UnwatchSubforum removes an agent's subscription; not an error if absent.
func (s *Store) UnwatchSubforum(ctx context.Context, agentID, key string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM subforum_watches WHERE agent_id = ? AND subforum_key = ?",
		agentID, key)
	if err != nil {
		return fmt.Errorf("store: unwatch subforum: %w", err)
	}
	return nil
}

// IsWatchingSubforum reports whether an agent watches a subforum.
func (s *Store) IsWatchingSubforum(ctx context.Context, agentID, key string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM subforum_watches WHERE agent_id = ? AND subforum_key = ?",
		agentID, key).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("store: is watching subforum: %w", err)
	}
	return n > 0, nil
}

// WatchedSubforumKeys returns the keys of subforums an agent watches.
func (s *Store) WatchedSubforumKeys(ctx context.Context, agentID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT subforum_key FROM subforum_watches WHERE agent_id = ? ORDER BY subforum_key",
		agentID)
	if err != nil {
		return nil, fmt.Errorf("store: list watched subforums: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
