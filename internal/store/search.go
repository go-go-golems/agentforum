package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

// TermFilter selects entities that have a metadata term matching Value under
// any of Keys (OR within a filter). Multiple TermFilters are AND-combined.
// Keys with more than one entry supports the --ticket convenience (which
// matches either a top-level "ticket" or "external_refs.value").
type TermFilter struct {
	Keys  []string
	Value string
}

// SearchFilter is the shared query shape for cross-entity metadata/text search.
type SearchFilter struct {
	Subforum     string
	Text         string
	Terms        []TermFilter
	CreatedAfter time.Time
	Limit        int
}

// appendTermExists adds an `EXISTS (… metadata_terms …)` clause for one
// TermFilter, scoped to a given entity type and the column holding the entity
// id (e.g. "threads.id" or "posts.id"). Returns the new WHERE fragment; args
// are appended in order.
func appendTermExists(entityType, entityCol string, f TermFilter, args *[]any) string {
	placeholders := make([]string, len(f.Keys))
	for i, k := range f.Keys {
		placeholders[i] = "?"
		*args = append(*args, k)
	}
	*args = append(*args, f.Value)
	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM metadata_terms t WHERE t.entity_type = ? AND t.entity_id = %s AND t.key IN (%s) AND t.value = ?)",
		entityCol, strings.Join(placeholders, ","))
}

// metadataWhere builds the AND-combined WHERE fragments for a slice of terms.
func metadataWhere(entityType, entityCol string, terms []TermFilter, args *[]any) []string {
	var out []string
	for _, f := range terms {
		if len(f.Keys) == 0 || f.Value == "" {
			continue
		}
		*args = append(*args, entityType) // entity_type placeholder
		out = append(out, appendTermExists(entityType, entityCol, f, args))
	}
	return out
}

// SearchThreads returns threads matching the filter (subforum, title text, and
// AND-combined metadata terms), newest-updated first.
func (s *Store) SearchThreads(ctx context.Context, f SearchFilter) ([]*models.Thread, error) {
	var (
		where []string
		args  []any
	)
	if f.Subforum != "" {
		where = append(where, "subforum_key = ?")
		args = append(args, f.Subforum)
	}
	if f.Text != "" {
		where = append(where, "title LIKE ?")
		args = append(args, "%"+f.Text+"%")
	}
	where = append(where, metadataWhere("thread", "threads.id", f.Terms, &args)...)
	if !f.CreatedAfter.IsZero() {
		where = append(where, "created_at > ?")
		args = append(args, f.CreatedAfter.UTC().Format(time.RFC3339Nano))
	}
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
		return nil, fmt.Errorf("store: search threads: %w", err)
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

// postColumnsQualified is postColumns with posts. prefixes, for queries that
// JOIN threads (so "id" is not ambiguous).
const postColumnsQualified = "posts.id, posts.thread_id, posts.author_id, posts.body, posts.reply_to, posts.metadata, posts.created_at"

// SearchPosts returns posts matching the filter. Subforum is applied via a
// join to threads; text matches the post body; metadata terms are AND-combined.
func (s *Store) SearchPosts(ctx context.Context, f SearchFilter) ([]*models.Post, error) {
	var (
		where []string
		args  []any
	)
	q := "SELECT " + postColumnsQualified + " FROM posts JOIN threads ON posts.thread_id = threads.id"
	if f.Subforum != "" {
		where = append(where, "threads.subforum_key = ?")
		args = append(args, f.Subforum)
	}
	if f.Text != "" {
		where = append(where, "posts.body LIKE ?")
		args = append(args, "%"+f.Text+"%")
	}
	where = append(where, metadataWhere("post", "posts.id", f.Terms, &args)...)
	if !f.CreatedAfter.IsZero() {
		where = append(where, "posts.created_at > ?")
		args = append(args, f.CreatedAfter.UTC().Format(time.RFC3339Nano))
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY posts.created_at DESC"
	if f.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, f.Limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: search posts: %w", err)
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
