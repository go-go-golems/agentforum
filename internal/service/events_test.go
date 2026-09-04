package service_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
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

// TestPollEventsConcurrentLongPollers is the R4 verification the
// AGENTFORUM-002 design promised before the server shipped (and the
// AGENTFORUM-004 design made phase S1): N watchers hold concurrent
// long-polls while the writer posts. Every poller must receive the
// events inside its wait budget and no store error (e.g. "database is
// locked") may surface. Latencies are logged so pool settings can be
// compared before/after (diary Step 2).
func TestPollEventsConcurrentLongPollers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc, cleanup := newService(t)
	defer cleanup()

	const (
		n     = 16 // concurrent long-pollers (browser tabs)
		posts = 8  // writes arriving while polls are open
	)

	writer, _, err := svc.Register(ctx, service.RegisterInput{Name: "writer"})
	if err != nil {
		t.Fatalf("register writer: %v", err)
	}
	if _, err := svc.CreateSubforum(ctx, writer, service.CreateSubforumInput{Key: "eng"}); err != nil {
		t.Fatalf("create subforum: %v", err)
	}
	thread, _, err := svc.CreateThread(ctx, writer, service.CreateThreadInput{Subforum: "eng", Title: "t", Body: "open"})
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	// Register watchers, watch the thread, and start each from a caught-up
	// cursor so their polls block in the wait loop rather than draining.
	watchers := make([]*models.Agent, n)
	cursors := make([]int64, n)
	for i := 0; i < n; i++ {
		w, _, err := svc.Register(ctx, service.RegisterInput{Name: fmt.Sprintf("watcher-%02d", i)})
		if err != nil {
			t.Fatalf("register watcher %d: %v", i, err)
		}
		if err := svc.WatchThread(ctx, w, thread.ID); err != nil {
			t.Fatalf("watch %d: %v", i, err)
		}
		_, cur, err := svc.PollEvents(ctx, w, service.PollEventsOptions{Cursor: 0, Wait: 0})
		if err != nil {
			t.Fatalf("prime poll %d: %v", i, err)
		}
		watchers[i], cursors[i] = w, cur
	}

	var posted atomic.Value // time.Time of the first write
	latencies := make([]time.Duration, n)
	errCh := make(chan error, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			evs, _, err := svc.PollEvents(ctx, watchers[i], service.PollEventsOptions{
				Cursor: cursors[i], Wait: 5 * time.Second,
			})
			if err != nil {
				errCh <- fmt.Errorf("watcher %d poll: %w", i, err)
				return
			}
			if len(evs) == 0 {
				errCh <- fmt.Errorf("watcher %d: poll returned no events", i)
				return
			}
			for _, ev := range evs {
				if ev.Reason != models.ReasonWatching {
					errCh <- fmt.Errorf("watcher %d: reason %q, want watching", i, ev.Reason)
					return
				}
			}
			latencies[i] = time.Since(posted.Load().(time.Time))
		}(i)
	}

	// Let the pollers enter their wait cycles, then write while they are open.
	time.Sleep(300 * time.Millisecond)
	for j := 0; j < posts; j++ {
		if _, err := svc.CreatePost(ctx, writer, service.CreatePostInput{ThreadID: thread.ID, Body: fmt.Sprintf("post %d", j)}); err != nil {
			t.Fatalf("write %d: %v", j, err)
		}
		if j == 0 {
			posted.Store(time.Now())
		}
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	// Report the distribution so pool changes are comparable across runs
	// (recorded in the ticket diary).
	sort.Slice(latencies, func(a, b int) bool { return latencies[a] < latencies[b] })
	t.Logf("delivery latency for %d watchers (p50=%v max=%v): %v", n,
		latencies[n/2], latencies[n-1], latencies)

	// Every poller must see the events well inside the 5s budget. The
	// generous ceiling catches serialization disasters (writes starving
	// behind poll reads on a single connection) without being flaky.
	if max := latencies[n-1]; max > 3*time.Second {
		t.Fatalf("slowest watcher received events after %v (budget 5s, ceiling 3s)", max)
	}
}
