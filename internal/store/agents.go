package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
)

// CreateAgent inserts a new agent. The caller must set ID, Name, TokenHash,
// CreatedAt, UpdatedAt. Uniqueness of name and token_hash is enforced by the
// schema; a violation surfaces as a SQLite UNIQUE error the service maps.
func (s *Store) CreateAgent(ctx context.Context, a *models.Agent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agents
(id, name, display_name, bio, status, metadata, token_hash, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?)`,
		a.ID, a.Name, a.DisplayName, a.Bio, a.Status,
		marshalMetadata(a.Metadata), a.TokenHash,
		a.CreatedAt.UTC().Format(time.RFC3339Nano), a.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store: create agent: %w", err)
	}
	return nil
}

// GetAgentByName returns the agent with the given unique name.
func (s *Store) GetAgentByName(ctx context.Context, name string) (*models.Agent, error) {
	return s.getAgent(ctx, "SELECT "+agentColumns+" FROM agents WHERE name = ?", name)
}

// GetAgentByID returns the agent with the given id.
func (s *Store) GetAgentByID(ctx context.Context, id string) (*models.Agent, error) {
	return s.getAgent(ctx, "SELECT "+agentColumns+" FROM agents WHERE id = ?", id)
}

// GetAgentByTokenHash returns the agent whose token hash matches.
func (s *Store) GetAgentByTokenHash(ctx context.Context, hash string) (*models.Agent, error) {
	return s.getAgent(ctx, "SELECT "+agentColumns+" FROM agents WHERE token_hash = ?", hash)
}

// UpdateAgent patches the mutable profile fields and bumps updated_at.
func (s *Store) UpdateAgent(ctx context.Context, a *models.Agent) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agents SET
display_name=?, bio=?, status=?, metadata=?, updated_at=? WHERE id=?`,
		a.DisplayName, a.Bio, a.Status, marshalMetadata(a.Metadata),
		a.UpdatedAt.UTC().Format(time.RFC3339Nano), a.ID)
	if err != nil {
		return fmt.Errorf("store: update agent: %w", err)
	}
	return nil
}

const agentColumns = "id, name, display_name, bio, status, metadata, token_hash, created_at, updated_at"

// AgentNames returns id -> name for the given agent ids in one query. Used
// to denormalize author/actor names in HTTP responses without N+1 lookups.
func (s *Store) AgentNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	ph := strings.Repeat("?,", len(ids))
	ph = ph[:len(ph)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name FROM agents WHERE id IN ("+ph+")",
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

func (s *Store) getAgent(ctx context.Context, q string, args ...any) (*models.Agent, error) {
	row := s.db.QueryRowContext(ctx, q, args...)
	a, err := scanAgent(row)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// scanAgent reads one agent row from a single-row scanner.
func scanAgent(r interface {
	Scan(dest ...any) error
}) (*models.Agent, error) {
	a := &models.Agent{Metadata: map[string]any{}}
	var meta, tokenHash, created, updated string
	if err := r.Scan(
		&a.ID, &a.Name, &a.DisplayName, &a.Bio, &a.Status,
		&meta, &tokenHash, &created, &updated,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("store: agent: %w", ErrNoRows)
		}
		return nil, fmt.Errorf("store: scan agent: %w", err)
	}
	a.TokenHash = tokenHash
	a.Metadata = unmarshalMetadata(meta)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	return a, nil
}

// ErrNoRows is the store-level sentinel for a missing row, re-exposed so the
// service does not import database/sql just to match sql.ErrNoRows.
var ErrNoRows = sql.ErrNoRows
