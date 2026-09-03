// Package service is the business-logic layer of agentforum. It wraps the
// store with rules: uniqueness, token auth, idempotency, event emission, and
// the reason computation behind the unified inbox. It never imports cobra or
// glazed, so a future HTTP server can reuse it unchanged (design doc §3.1,
// §10.4).
package service

import (
	"context"
	"time"

	"github.com/go-go-golems/agentforum/internal/store"
)

// Service holds the store and exposes domain operations. Methods are grouped
// by entity across sibling files (agents.go, subforums.go, …) and added in
// later phases; the P1 milestone only needs construction and close.
type Service struct {
	store *store.Store
}

// NewService wraps an open store.
func NewService(s *store.Store) *Service { return &Service{store: s} }

// Store exposes the underlying store for service-internal use.
func (s *Service) Store() *store.Store { return s.store }

// Close releases the underlying database handle.
func (s *Service) Close() error { return s.store.Close() }

// nowUTC is the shared current time for service writes, fixed to UTC.
func nowUTC() time.Time { return time.Now().UTC() }

// Ping verifies the database is reachable. Used by the P1 `db init` command.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.DB().PingContext(ctx)
}
