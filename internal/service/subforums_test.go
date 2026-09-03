package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/agentforum/internal/service"
)

func TestSubforumCreateAndWatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	agent, _, err := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	sf, err := svc.CreateSubforum(ctx, agent, service.CreateSubforumInput{
		Key: "engineering", Title: "Engineering", Description: "notes",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sf.Key != "engineering" || sf.CreatorID != agent.ID {
		t.Fatalf("unexpected subforum: %+v", sf)
	}

	// Duplicate key conflicts.
	if _, err := svc.CreateSubforum(ctx, agent, service.CreateSubforumInput{Key: "engineering"}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("dup key: want ErrConflict, got %v", err)
	}

	// Bad key is invalid input.
	for _, bad := range []string{"", "UPPER", "has space", "-leading", "x!"} {
		if _, err := svc.CreateSubforum(ctx, agent, service.CreateSubforumInput{Key: bad}); !errors.Is(err, service.ErrInvalidInput) {
			t.Fatalf("bad key %q: want ErrInvalidInput, got %v", bad, err)
		}
	}

	// List returns the subforum.
	list, err := svc.ListSubforums(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "engineering" {
		t.Fatalf("list = %+v", list)
	}

	// Get missing subforum is not found.
	if _, err := svc.GetSubforum(ctx, "nope"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("get missing: want ErrNotFound, got %v", err)
	}

	// Watch requires the subforum to exist.
	if err := svc.WatchSubforum(ctx, agent, "nope"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("watch missing: want ErrNotFound, got %v", err)
	}
	if err := svc.WatchSubforum(ctx, agent, "engineering"); err != nil {
		t.Fatalf("watch: %v", err)
	}
	// Watching again is idempotent.
	if err := svc.WatchSubforum(ctx, agent, "engineering"); err != nil {
		t.Fatalf("watch again: %v", err)
	}
	// Unwatch is idempotent even if not watching.
	if err := svc.UnwatchSubforum(ctx, agent, "engineering"); err != nil {
		t.Fatalf("unwatch: %v", err)
	}
	if err := svc.UnwatchSubforum(ctx, agent, "engineering"); err != nil {
		t.Fatalf("unwatch again: %v", err)
	}
}
