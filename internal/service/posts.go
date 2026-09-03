package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-go-golems/agentforum/internal/id"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
)

// CreatePostInput is the service-level argument for replying in a thread.
type CreatePostInput struct {
	ThreadID       string
	Body           string
	ReplyTo        string
	Metadata       map[string]any
	IdempotencyKey string
}

// CreatePost adds a post to an existing thread. The thread must exist; if
// ReplyTo is set it must point to a post in the same thread. Metadata is
// validated. The author becomes a participant and a post.created event is
// emitted (all atomic at the store layer).
func (s *Service) CreatePost(ctx context.Context, agent *models.Agent, in CreatePostInput) (*models.Post, error) {
	in.ThreadID = strings.TrimSpace(in.ThreadID)
	if in.ThreadID == "" {
		return nil, fmt.Errorf("%w: thread id is required", ErrInvalidInput)
	}
	if err := validateMetadata(in.Metadata); err != nil {
		return nil, err
	}

	th, err := s.GetThread(ctx, in.ThreadID)
	if err != nil {
		return nil, err
	}

	if in.ReplyTo != "" {
		ref, err := s.store.GetPost(ctx, in.ReplyTo)
		if err != nil {
			if errors.Is(err, store.ErrNoRows) {
				return nil, fmt.Errorf("%w: reply_to %q not found", ErrNotFound, in.ReplyTo)
			}
			return nil, err
		}
		if ref.ThreadID != in.ThreadID {
			return nil, fmt.Errorf("%w: reply_to %q is in a different thread", ErrInvalidInput, in.ReplyTo)
		}
	}

	post := &models.Post{
		ID:        id.NewPostID(),
		ThreadID:  th.ID,
		AuthorID:  agent.ID,
		Body:      in.Body,
		ReplyTo:   in.ReplyTo,
		Metadata:  in.Metadata,
		CreatedAt: nowUTC(),
	}

	// Idempotency: a retried create with the same key returns the first result.
	if in.IdempotencyKey != "" {
		if rec, err := s.store.GetIdempotencyRecord(ctx, in.IdempotencyKey, agent.ID); err == nil && rec != nil {
			var resp struct {
				Post *models.Post `json:"post"`
			}
			if err := json.Unmarshal([]byte(rec.Response), &resp); err == nil && resp.Post != nil {
				return resp.Post, nil
			}
		} else if err != nil && !errors.Is(err, store.ErrNoRows) {
			return nil, err
		}
	}

	created, err := s.store.CreatePost(ctx, store.CreatePostInput{Post: post, Subforum: th.Subforum})
	if err != nil {
		return nil, err
	}
	if in.IdempotencyKey != "" {
		respJSON, _ := json.Marshal(map[string]any{"post": created})
		_ = s.store.SaveIdempotencyRecord(ctx, &store.IdempotencyRecord{
			Key: in.IdempotencyKey, AgentID: agent.ID, Entity: "post",
			EntityID: created.ID, Response: string(respJSON), CreatedAt: created.CreatedAt,
		})
	}
	return created, nil
}

// ListPosts returns posts in a thread, optionally after a post id, capped at
// limit (0 = unlimited). The thread must exist (ErrNotFound).
func (s *Service) ListPosts(ctx context.Context, threadID, afterPostID string, limit int) ([]*models.Post, error) {
	if _, err := s.GetThread(ctx, threadID); err != nil {
		return nil, err
	}
	posts, err := s.store.ListPosts(ctx, threadID, afterPostID, limit)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, fmt.Errorf("%w: after-post %q", ErrNotFound, afterPostID)
		}
		return nil, err
	}
	return posts, nil
}

// GetPost returns a post by id, or ErrNotFound.
func (s *Service) GetPost(ctx context.Context, postID string) (*models.Post, error) {
	p, err := s.store.GetPost(ctx, postID)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, fmt.Errorf("%w: post %q", ErrNotFound, postID)
		}
		return nil, err
	}
	return p, nil
}
