package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-go-golems/agentforum/internal/service"
)

func TestCreateThreadAtomicAndListing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, err := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	if err != nil {
		t.Fatalf("register alice: %v", err)
	}
	bob, _, err := svc.Register(ctx, service.RegisterInput{Name: "bob"})
	if err != nil {
		t.Fatalf("register bob: %v", err)
	}
	if _, err := svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng", Title: "Eng"}); err != nil {
		t.Fatalf("create subforum: %v", err)
	}

	// Empty title / missing subforum are invalid.
	if _, _, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng"}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("empty title: want ErrInvalidInput, got %v", err)
	}
	if _, _, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{Title: "t"}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("missing subforum: want ErrInvalidInput, got %v", err)
	}
	// Missing subforum is not found.
	if _, _, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "nope", Title: "t"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("missing subforum: want ErrNotFound, got %v", err)
	}

	thread, post, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{
		Subforum: "eng", Title: "Caching", Body: "tracing",
		Metadata: map[string]any{"ticket": "PLAT-1", "keywords": []any{"caching"}},
	})
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if thread.ID == "" || post.ID == "" || post.ThreadID != thread.ID {
		t.Fatalf("bad thread/post: %+v / %+v", thread, post)
	}

	// Atomic write emitted exactly thread.created + post.created (2 events).
	var n int
	if err := svc.Store().DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&n); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if n != 2 {
		t.Fatalf("events = %d, want 2", n)
	}
	// Metadata terms were flattened for the thread.
	var tTerms int
	if err := svc.Store().DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM metadata_terms WHERE entity_type='thread' AND entity_id=?", thread.ID).Scan(&tTerms); err != nil {
		t.Fatalf("count thread terms: %v", err)
	}
	if tTerms != 2 { // ticket=PLAT-1, keywords=caching (1 keyword)
		t.Fatalf("thread terms = %d, want 2", tTerms)
	}

	// Alice is involved (creator); not watching.
	if got, _ := svc.ListThreads(ctx, alice, service.ListThreadsOptions{Involved: true}); len(got) != 1 {
		t.Fatalf("alice involved = %d, want 1", len(got))
	}
	if got, _ := svc.ListThreads(ctx, alice, service.ListThreadsOptions{Watching: true}); len(got) != 0 {
		t.Fatalf("alice watching = %d, want 0", len(got))
	}
	// Bob is not involved yet.
	if got, _ := svc.ListThreads(ctx, bob, service.ListThreadsOptions{Involved: true}); len(got) != 0 {
		t.Fatalf("bob involved = %d, want 0", len(got))
	}

	// Bob replies → becomes participant.
	if _, err := svc.CreatePost(ctx, bob, service.CreatePostInput{ThreadID: thread.ID, Body: "found it"}); err != nil {
		t.Fatalf("bob post: %v", err)
	}
	if got, _ := svc.ListThreads(ctx, bob, service.ListThreadsOptions{Involved: true}); len(got) != 1 {
		t.Fatalf("bob involved after post = %d, want 1", len(got))
	}
}

func TestCreatePostReplyToValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, _ := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng"})
	t1, p1, _ := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng", Title: "t1", Body: "a"})
	t2, _, _ := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng", Title: "t2", Body: "b"})

	// reply_to in a different thread is invalid input.
	if _, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: t2.ID, ReplyTo: p1.ID, Body: "x"}); !errors.Is(err, service.ErrInvalidInput) {
		t.Fatalf("cross-thread reply: want ErrInvalidInput, got %v", err)
	}
	// reply_to that doesn't exist is not found.
	if _, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: t1.ID, ReplyTo: "po_nope", Body: "x"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("missing reply_to: want ErrNotFound, got %v", err)
	}
	// posting in a missing thread is not found.
	if _, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: "th_nope", Body: "x"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("missing thread post: want ErrNotFound, got %v", err)
	}
	// valid reply in same thread works.
	if _, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: t1.ID, ReplyTo: p1.ID, Body: "ok"}); err != nil {
		t.Fatalf("valid reply: %v", err)
	}
}
