package service

import (
	"context"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
)

// TermFilter is re-exported from the store so the CLI builds filters without
// importing database types.
type TermFilter = store.TermFilter

// SearchInput is the service-level query for metadata/text search.
type SearchInput struct {
	Subforum     string
	Text         string
	Terms        []TermFilter
	CreatedAfter time.Time
	Limit        int
}

func toStoreFilter(in SearchInput) store.SearchFilter {
	return store.SearchFilter{
		Subforum:     in.Subforum,
		Text:         in.Text,
		Terms:        in.Terms,
		CreatedAfter: in.CreatedAfter,
		Limit:        in.Limit,
	}
}

// SearchThreads returns threads matching the filter.
func (s *Service) SearchThreads(ctx context.Context, in SearchInput) ([]*models.Thread, error) {
	return s.store.SearchThreads(ctx, toStoreFilter(in))
}

// SearchPosts returns posts matching the filter.
func (s *Service) SearchPosts(ctx context.Context, in SearchInput) ([]*models.Post, error) {
	return s.store.SearchPosts(ctx, toStoreFilter(in))
}

// SearchResults holds a combined search response.
type SearchResults struct {
	Threads []*models.Thread
	Posts   []*models.Post
}

// Search runs a cross-entity search. entityTypes selects which entities to
// return ("thread", "post", or both).
func (s *Service) Search(ctx context.Context, in SearchInput, entityTypes []string) (*SearchResults, error) {
	wantThread, wantPost := false, false
	for _, e := range entityTypes {
		switch e {
		case "thread":
			wantThread = true
		case "post":
			wantPost = true
		}
	}
	if !wantThread && !wantPost {
		wantThread, wantPost = true, true
	}
	res := &SearchResults{}
	if wantThread {
		threads, err := s.SearchThreads(ctx, in)
		if err != nil {
			return nil, err
		}
		res.Threads = threads
	}
	if wantPost {
		posts, err := s.SearchPosts(ctx, in)
		if err != nil {
			return nil, err
		}
		res.Posts = posts
	}
	return res, nil
}
