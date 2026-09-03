package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/service"
)

func TestPollEventsReasonsAndSelfExclusion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, _ := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	bob, _, _ := svc.Register(ctx, service.RegisterInput{Name: "bob"})
	carol, _, _ := svc.Register(ctx, service.RegisterInput{Name: "carol"})

	svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng"})
	thread, _, _ := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng", Title: "t", Body: "open", Watch: true})

	// bob watches the thread; carol watches the subforum.
	svc.WatchThread(ctx, bob, thread.ID)
	svc.WatchSubforum(ctx, carol, "eng")

	// alice posts -> event #3 (events 1,2 were thread/opening-post creation).
	svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: thread.ID, Body: "reply"})

	// bob (watches thread) sees all 3 events as "watching".
	evs, next, err := svc.PollEvents(ctx, bob, service.PollEventsOptions{Cursor: 0, Wait: 0})
	if err != nil {
		t.Fatalf("bob poll: %v", err)
	}
	if len(evs) != 3 || evs[0].Reason != "watching" {
		t.Fatalf("bob: want 3 watching, got %+v", evs)
	}
	if next != 3 {
		t.Fatalf("bob next = %d, want 3", next)
	}

	// carol (watches subforum) sees all 3 as "watched_subforum".
	evs, _, err = svc.PollEvents(ctx, carol, service.PollEventsOptions{Cursor: 0, Wait: 0})
	if err != nil {
		t.Fatalf("carol poll: %v", err)
	}
	if len(evs) != 3 || evs[0].Reason != "watched_subforum" {
		t.Fatalf("carol: want 3 watched_subforum, got %+v", evs)
	}

	// alice is the actor of all events -> self-excluded -> 0 events.
	evs, next, err = svc.PollEvents(ctx, alice, service.PollEventsOptions{Cursor: 0, Wait: 0})
	if err != nil {
		t.Fatalf("alice poll: %v", err)
	}
	if len(evs) != 0 {
		t.Fatalf("alice: want 0 (self-excluded), got %d", len(evs))
	}
	if next != 3 {
		t.Fatalf("alice next = %d, want 3 (advanced past)", next)
	}

	// scope=involved (participating only): bob is not a participant -> 0.
	evs, _, err = svc.PollEvents(ctx, bob, service.PollEventsOptions{Cursor: 0, Wait: 0, Scope: "involved"})
	if err != nil {
		t.Fatalf("bob scope poll: %v", err)
	}
	if len(evs) != 0 {
		t.Fatalf("bob involved-only: want 0, got %d", len(evs))
	}

	// ack advances the durable cursor.
	if err := svc.AckEvents(ctx, bob, 3); err != nil {
		t.Fatalf("ack: %v", err)
	}
	ack, _ := svc.GetAck(ctx, bob)
	if ack != 3 {
		t.Fatalf("ack = %d, want 3", ack)
	}
	evs, _, err = svc.PollEvents(ctx, bob, service.PollEventsOptions{Cursor: ack, Wait: 0})
	if err != nil {
		t.Fatalf("bob after-ack poll: %v", err)
	}
	if len(evs) != 0 {
		t.Fatalf("bob after ack: want 0, got %d", len(evs))
	}
}

// TestPollEventsLongPollWaits verifies the long-poll blocks until an eligible
// event arrives (or the deadline), proving the wait loop works concurrently.
func TestPollEventsLongPollWaits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	alice, _, _ := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	bob, _, _ := svc.Register(ctx, service.RegisterInput{Name: "bob"})
	svc.CreateSubforum(ctx, alice, service.CreateSubforumInput{Key: "eng"})
	thread, _, _ := svc.CreateThread(ctx, alice, service.CreateThreadInput{Subforum: "eng", Title: "t", Body: "open"})
	svc.WatchThread(ctx, bob, thread.ID)

	// bob acks past the existing events so the poll starts caught up.
	_, cur, _ := svc.PollEvents(ctx, bob, service.PollEventsOptions{Cursor: 0, Wait: 0})
	if err := svc.AckEvents(ctx, bob, cur); err != nil {
		t.Fatalf("ack: %v", err)
	}

	// Start a long poll that should block until alice posts.
	got := make(chan []*models.Event, 1)
	go func() {
		evs, _, _ := svc.PollEvents(ctx, bob, service.PollEventsOptions{Cursor: cur, Wait: 2 * time.Second})
		got <- evs
	}()

	// Give the poller a moment to enter the wait, then post.
	time.Sleep(150 * time.Millisecond)
	start := time.Now()
	svc.CreatePost(ctx, alice, service.CreatePostInput{ThreadID: thread.ID, Body: "late"})

	select {
	case evs := <-got:
		elapsed := time.Since(start)
		if len(evs) != 1 || evs[0].Reason != "watching" {
			t.Fatalf("long-poll result = %+v", evs)
		}
		if elapsed > 2*time.Second {
			t.Fatalf("long-poll waited too long (%v); should return soon after the post", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("long-poll did not return after an eligible event was posted")
	}
}
