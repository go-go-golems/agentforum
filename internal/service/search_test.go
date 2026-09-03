package service_test

import (
	"context"
	"testing"

	"github.com/go-go-golems/agentforum/internal/service"
)

func TestIdempotentCreateThreadAndPost(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, _ := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng"})

	// First create with an idempotency key.
	t1, p1, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{
		Subforum: "eng", Title: "first", Body: "a", IdempotencyKey: "run-1",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Retried create with the same key returns the FIRST result, ignoring new args.
	t2, p2, err := svc.CreateThread(ctx, alice, service.CreateThreadInput{
		Subforum: "eng", Title: "DIFFERENT", Body: "b", IdempotencyKey: "run-1",
	})
	if err != nil {
		t.Fatalf("replay create: %v", err)
	}
	if t2.ID != t1.ID || p2.ID != p1.ID {
		t.Fatalf("idempotency replay: got %s/%s, want %s/%s", t2.ID, p2.ID, t1.ID, p1.ID)
	}
	if t2.Title != "first" {
		t.Fatalf("idempotency replay returned new title %q", t2.Title)
	}
	// Only one thread exists.
	var n int
	svc.Store().DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM threads").Scan(&n)
	if n != 1 {
		t.Fatalf("threads = %d, want 1", n)
	}

	// Post idempotency.
	r1, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: t1.ID, Body: "p1", IdempotencyKey: "post-1"})
	if err != nil {
		t.Fatalf("first post: %v", err)
	}
	r2, err := svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: t1.ID, Body: "DIFFERENT", IdempotencyKey: "post-1"})
	if err != nil {
		t.Fatalf("replay post: %v", err)
	}
	if r1.ID != r2.ID {
		t.Fatalf("post idempotency: %s != %s", r1.ID, r2.ID)
	}
	svc.Store().DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM posts WHERE thread_id=?", t1.ID).Scan(&n)
	// 1 opening post + 1 reply = 2 posts in the thread.
	if n != 2 {
		t.Fatalf("posts = %d, want 2", n)
	}
}

func TestMetadataSearch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, _ := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng"})
	t1, _, _ := svc.CreateThread(ctx, alice, service.CreateThreadInput{
		Subforum: "eng", Title: "Caching investigation", Body: "open",
		Metadata: map[string]any{
			"transcript_id": "tr_892",
			"ticket":        "PLAT-431",
			"keywords":      []any{"caching", "invalidation"},
		},
	})
	svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng", Title: "Other", Body: "open2"})
	svc.CreatePost(ctx, alice, service.CreatePostInput{
		ThreadID: t1.ID, Body: "Caching: the cache key is missing the locale.",
		Metadata: map[string]any{"turn_id": "turn_21", "keywords": []any{"root-cause"}},
	})

	// Thread filters.
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Terms: []service.TermFilter{{Keys: []string{"transcript_id"}, Value: "tr_892"}}}); len(got) != 1 || got[0].ID != t1.ID {
		t.Fatalf("meta transcript_id: got %+v", got)
	}
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Terms: []service.TermFilter{{Keys: []string{"keywords"}, Value: "caching"}}}); len(got) != 1 {
		t.Fatalf("keyword caching: got %d", len(got))
	}
	// --ticket matches either "ticket" or "external_refs.value".
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Terms: []service.TermFilter{{Keys: []string{"ticket", "external_refs.value"}, Value: "PLAT-431"}}}); len(got) != 1 {
		t.Fatalf("ticket: got %d", len(got))
	}
	// AND-combined filters.
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Terms: []service.TermFilter{
		{Keys: []string{"transcript_id"}, Value: "tr_892"},
		{Keys: []string{"keywords"}, Value: "invalidation"},
	}}); len(got) != 1 {
		t.Fatalf("AND filters: got %d", len(got))
	}
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Terms: []service.TermFilter{{Keys: []string{"ticket"}, Value: "NOPE"}}}); len(got) != 0 {
		t.Fatalf("non-match: got %d", len(got))
	}
	// Text search on thread titles.
	if got, _ := svc.SearchThreads(ctx, service.SearchInput{Text: "investigation"}); len(got) != 1 {
		t.Fatalf("text title: got %d", len(got))
	}
	// Post search by metadata + text.
	if got, _ := svc.SearchPosts(ctx, service.SearchInput{Terms: []service.TermFilter{{Keys: []string{"turn_id"}, Value: "turn_21"}}}); len(got) != 1 {
		t.Fatalf("post meta: got %d", len(got))
	}
	if got, _ := svc.SearchPosts(ctx, service.SearchInput{Text: "locale"}); len(got) != 1 {
		t.Fatalf("post text: got %d", len(got))
	}
	// Combined search returns both kinds (title and body both contain "caching").
	res, err := svc.Search(ctx, service.SearchInput{Text: "caching"}, []string{"thread", "post"})
	if err != nil {
		t.Fatalf("combined search: %v", err)
	}
	if len(res.Threads) != 1 || len(res.Posts) != 1 {
		t.Fatalf("combined: threads=%d posts=%d", len(res.Threads), len(res.Posts))
	}
}
