package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/id"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
)

// CreateThreadInput is the service-level argument for opening a thread.
type CreateThreadInput struct {
	Subforum       string
	Title          string
	Body           string
	Metadata       map[string]any // thread metadata
	PostMetadata   map[string]any // opening-post metadata
	Watch          bool
	IdempotencyKey string // optional; a retried write returns the first result
}

// CreateThread opens a thread and its opening post atomically. The subforum
// must exist; title must be non-empty; both metadata blobs are validated.
func (s *Service) CreateThread(ctx context.Context, agent *models.Agent, in CreateThreadInput) (*models.Thread, *models.Post, error) {
	in.Subforum = strings.TrimSpace(in.Subforum)
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if in.Subforum == "" {
		return nil, nil, fmt.Errorf("%w: subforum is required", ErrInvalidInput)
	}
	if err := validateMetadata(in.Metadata); err != nil {
		return nil, nil, err
	}
	if err := validateMetadata(in.PostMetadata); err != nil {
		return nil, nil, err
	}
	if _, err := s.GetSubforum(ctx, in.Subforum); err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()

	// Idempotency: a retried create with the same key returns the first result.
	if in.IdempotencyKey != "" {
		if rec, err := s.store.GetIdempotencyRecord(ctx, in.IdempotencyKey, agent.ID); err == nil && rec != nil {
			var resp struct {
				Thread      *models.Thread `json:"thread"`
				InitialPost *models.Post   `json:"initial_post"`
			}
			if err := json.Unmarshal([]byte(rec.Response), &resp); err == nil && resp.Thread != nil {
				return resp.Thread, resp.InitialPost, nil
			}
		} else if err != nil && !errors.Is(err, store.ErrNoRows) {
			return nil, nil, err
		}
	}

	thread := &models.Thread{
		ID:        id.NewThreadID(),
		Subforum:  in.Subforum,
		Title:     in.Title,
		Metadata:  in.Metadata,
		CreatorID: agent.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	post := &models.Post{
		ID:        id.NewPostID(),
		ThreadID:  thread.ID,
		AuthorID:  agent.ID,
		Body:      in.Body,
		Metadata:  in.PostMetadata,
		CreatedAt: now,
	}
	th, p, err := s.store.CreateThreadWithPost(ctx, store.CreateThreadWithPostInput{
		Thread: thread, Post: post, Watch: in.Watch,
	})
	if err != nil {
		return nil, nil, err
	}
	if in.IdempotencyKey != "" {
		respJSON, _ := json.Marshal(map[string]any{"thread": th, "initial_post": p})
		_ = s.store.SaveIdempotencyRecord(ctx, &store.IdempotencyRecord{
			Key: in.IdempotencyKey, AgentID: agent.ID, Entity: "thread",
			EntityID: th.ID, Response: string(respJSON), CreatedAt: now,
		})
	}
	return th, p, nil
}

// ListThreadsOptions selects threads. Involved/Watching are agent-scoped.
type ListThreadsOptions struct {
	Involved bool
	Watching bool
	Subforum string
	Terms    []TermFilter
	Limit    int
}

// ListThreads returns threads matching opts. Involved/Watching require an agent.
func (s *Service) ListThreads(ctx context.Context, agent *models.Agent, opts ListThreadsOptions) ([]*models.Thread, error) {
	f := store.ListThreadsFilter{Subforum: opts.Subforum, Terms: opts.Terms, Limit: opts.Limit}
	if opts.Involved && agent != nil {
		f.InvolvedAgent = agent.ID
	}
	if opts.Watching && agent != nil {
		f.WatchingAgent = agent.ID
	}
	return s.store.ListThreads(ctx, f)
}

// GetThread returns a thread by id, or ErrNotFound.
func (s *Service) GetThread(ctx context.Context, threadID string) (*models.Thread, error) {
	th, err := s.store.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, fmt.Errorf("%w: thread %q", ErrNotFound, threadID)
		}
		return nil, err
	}
	return th, nil
}

// WatchThread subscribes an agent to a thread (idempotent). The thread must exist.
func (s *Service) WatchThread(ctx context.Context, agent *models.Agent, threadID string) error {
	if _, err := s.GetThread(ctx, threadID); err != nil {
		return err
	}
	return s.store.WatchThread(ctx, agent.ID, threadID, time.Now().UTC())
}

// UnwatchThread removes an agent's thread subscription (idempotent).
func (s *Service) UnwatchThread(ctx context.Context, agent *models.Agent, threadID string) error {
	return s.store.UnwatchThread(ctx, agent.ID, threadID)
}
