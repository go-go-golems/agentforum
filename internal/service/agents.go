package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/id"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/store"
)

// RegisterInput captures the parameters for registering a new agent.
type RegisterInput struct {
	Name        string
	DisplayName string
	Bio         string
	Metadata    map[string]any
}

// Register creates a new agent with a unique name and returns the agent plus
// the plaintext token (returned exactly once). A duplicate name returns
// ErrConflict. Metadata is validated (§7.3).
func (s *Service) Register(ctx context.Context, in RegisterInput) (*models.Agent, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, "", fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if err := validateMetadata(in.Metadata); err != nil {
		return nil, "", err
	}

	// Pre-check name uniqueness for a clean 409; the UNIQUE index is the real
	// guarantee and also maps to ErrConflict on a race.
	if existing, err := s.store.GetAgentByName(ctx, in.Name); err == nil && existing != nil {
		_ = existing
		return nil, "", fmt.Errorf("%w: agent name %q already exists", ErrConflict, in.Name)
	} else if err != nil && !errors.Is(err, store.ErrNoRows) {
		return nil, "", err
	}

	now := time.Now().UTC()
	token := id.NewToken()
	agent := &models.Agent{
		ID:          id.NewAgentID(),
		Name:        in.Name,
		DisplayName: strings.TrimSpace(in.DisplayName),
		Bio:         in.Bio,
		Metadata:    in.Metadata,
		TokenHash:   id.HashToken(token),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateAgent(ctx, agent); err != nil {
		if isUniqueViolation(err) {
			return nil, "", fmt.Errorf("%w: agent name %q already exists", ErrConflict, in.Name)
		}
		return nil, "", err
	}
	return agent, token, nil
}

// UpdateMeInput holds the mutable profile fields. Fields left empty are not
// changed (clearing is intentionally out of scope for the milestone).
type UpdateMeInput struct {
	DisplayName string
	Bio         string
	Status      string
	Metadata    map[string]any
	HasMetadata bool
}

// GetMe resolves the authenticated agent from a bearer token.
func (s *Service) GetMe(ctx context.Context, token string) (*models.Agent, error) {
	return s.ResolveAgent(ctx, token)
}

// UpdateMe updates the authenticated agent's mutable fields and returns the
// new state.
func (s *Service) UpdateMe(ctx context.Context, token string, in UpdateMeInput) (*models.Agent, error) {
	agent, err := s.ResolveAgent(ctx, token)
	if err != nil {
		return nil, err
	}
	if in.HasMetadata {
		if err := validateMetadata(in.Metadata); err != nil {
			return nil, err
		}
		agent.Metadata = in.Metadata
	}
	if in.DisplayName != "" {
		agent.DisplayName = in.DisplayName
	}
	if in.Bio != "" {
		agent.Bio = in.Bio
	}
	if in.Status != "" {
		agent.Status = in.Status
	}
	agent.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateAgent(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

// GetAgentByName returns the public view of an agent by name (no token hash
// leaks because models.Agent.TokenHash has json:"-").
func (s *Service) GetAgentByName(ctx context.Context, name string) (*models.Agent, error) {
	a, err := s.store.GetAgentByName(ctx, name)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, fmt.Errorf("%w: agent %q", ErrNotFound, name)
		}
		return nil, err
	}
	return a, nil
}

// ResolveAgent authenticates a bearer token and returns the agent. An empty or
// unknown token yields ErrUnauthenticated. Every authenticated command starts
// here.
func (s *Service) ResolveAgent(ctx context.Context, token string) (*models.Agent, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrUnauthenticated
	}
	agent, err := s.store.GetAgentByTokenHash(ctx, id.HashToken(token))
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, ErrUnauthenticated
		}
		return nil, err
	}
	return agent, nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// modernc.org/sqlite returns errors whose message contains "UNIQUE constraint".
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint")
}
