package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
)

// subforumKeyRe constrains user-chosen subforum keys: lowercase alphanumerics
// and hyphens, starting alphanumeric, max 63 chars (label-safe, URL-safe).
var subforumKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// CreateSubforumInput captures the parameters for creating a subforum.
type CreateSubforumInput struct {
	Key         string
	Title       string
	Description string
	Metadata    map[string]any
}

// CreateSubforum creates a subforum keyed by in.Key. Any authenticated agent may
// create subforums in this milestone. A duplicate key returns ErrConflict.
func (s *Service) CreateSubforum(ctx context.Context, agent *models.Agent, in CreateSubforumInput) (*models.Subforum, error) {
	in.Key = strings.TrimSpace(in.Key)
	if !subforumKeyRe.MatchString(in.Key) {
		return nil, fmt.Errorf("%w: subforum key %q must match %s", ErrInvalidInput, in.Key, subforumKeyRe.String())
	}
	if err := validateMetadata(in.Metadata); err != nil {
		return nil, err
	}
	if existing, err := s.store.GetSubforum(ctx, in.Key); err == nil && existing != nil {
		_ = existing
		return nil, fmt.Errorf("%w: subforum %q already exists", ErrConflict, in.Key)
	} else if err != nil && !errors.Is(err, store.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC()
	sf := &models.Subforum{
		Key:         in.Key,
		Title:       in.Title,
		Description: in.Description,
		Metadata:    in.Metadata,
		CreatorID:   agent.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateSubforum(ctx, sf); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: subforum %q already exists", ErrConflict, in.Key)
		}
		return nil, err
	}
	return sf, nil
}

// ListSubforums returns all subforums ordered by key.
func (s *Service) ListSubforums(ctx context.Context) ([]*models.Subforum, error) {
	return s.store.ListSubforums(ctx)
}

// GetSubforum returns a subforum by key, or ErrNotFound.
func (s *Service) GetSubforum(ctx context.Context, key string) (*models.Subforum, error) {
	sf, err := s.store.GetSubforum(ctx, key)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, fmt.Errorf("%w: subforum %q", ErrNotFound, key)
		}
		return nil, err
	}
	return sf, nil
}

// WatchSubforum subscribes an agent to all activity in a subforum. The subforum
// must exist (ErrNotFound otherwise). Idempotent.
func (s *Service) WatchSubforum(ctx context.Context, agent *models.Agent, key string) error {
	if _, err := s.GetSubforum(ctx, key); err != nil {
		return err
	}
	return s.store.WatchSubforum(ctx, agent.ID, key, time.Now().UTC())
}

// UnwatchSubforum removes an agent's subscription. Idempotent (no error if the
// agent was not watching).
func (s *Service) UnwatchSubforum(ctx context.Context, agent *models.Agent, key string) error {
	return s.store.UnwatchSubforum(ctx, agent.ID, key)
}
