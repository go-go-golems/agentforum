package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
)

func newService(t *testing.T) (*service.Service, func()) {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := service.NewService(s)
	return svc, func() { _ = svc.Close() }
}

func TestRegisterAndAuth(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	agent, token, err := svc.Register(ctx, service.RegisterInput{
		Name: "researcher", DisplayName: "Research Agent", Bio: "investigates",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if agent.ID == "" || token == "" {
		t.Fatalf("expected id and token, got %+v / %q", agent, token)
	}

	// Token resolves to the same agent.
	me, err := svc.GetMe(ctx, token)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if me.ID != agent.ID {
		t.Fatalf("token resolved to %q, want %q", me.ID, agent.ID)
	}

	// Duplicate name conflicts.
	if _, _, err := svc.Register(ctx, service.RegisterInput{Name: "researcher"}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("duplicate name: want ErrConflict, got %v", err)
	}

	// Bad/missing token is unauthenticated.
	if _, err := svc.GetMe(ctx, ""); !errors.Is(err, service.ErrUnauthenticated) {
		t.Fatalf("empty token: want ErrUnauthenticated, got %v", err)
	}
	if _, err := svc.GetMe(ctx, "af_bogus"); !errors.Is(err, service.ErrUnauthenticated) {
		t.Fatalf("bogus token: want ErrUnauthenticated, got %v", err)
	}

	// Empty name is invalid.
	if _, _, err := svc.Register(ctx, service.RegisterInput{Name: "   "}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("blank name: want ErrInvalidInput, got %v", err)
	}
}

func TestUpdateMe(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	_, token, err := svc.Register(ctx, service.RegisterInput{Name: "bot"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	updated, err := svc.UpdateMe(ctx, token, service.UpdateMeInput{Status: "busy"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != "busy" {
		t.Fatalf("status = %q, want busy", updated.Status)
	}

	// Update without metadata should not wipe existing fields.
	if updated.Name != "bot" {
		t.Fatalf("name changed unexpectedly: %q", updated.Name)
	}
}

func TestRegisterReservedMetadataKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	_, _, err := svc.Register(ctx, service.RegisterInput{
		Name:     "reserved",
		Metadata: map[string]any{"_internal": "nope"},
	})
	if !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("reserved key: want ErrInvalidInput, got %v", err)
	}
}
