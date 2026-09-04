package review

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/server"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
)

// These are review probes: passing means the documented defect was reproduced.
// They do not establish desired production behavior and must not be copied as
// regression assertions without reversing the relevant expectations.
func TestReviewProbes(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "review.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := service.NewService(st)
	a, _, err := svc.Register(ctx, service.RegisterInput{Name: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := svc.Register(ctx, service.RegisterInput{Name: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateSubforum(ctx, a, service.CreateSubforumInput{Key: "review"})
	if err != nil {
		t.Fatal(err)
	}
	th, _, err := svc.CreateThread(ctx, a, service.CreateThreadInput{Subforum: "review", Title: "Probe", Body: "opening", Watch: true})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("ack_regresses", func(t *testing.T) {
		for _, n := range []int64{100, 50} {
			if err := svc.AckEvents(ctx, a, n); err != nil {
				t.Fatal(err)
			}
		}
		n, err := svc.GetAck(ctx, a)
		if err != nil || n != 50 {
			t.Fatalf("want observed regression to 50, got %d: %v", n, err)
		}
		t.Log("OBSERVED: ack(100); ack(50) => 50; future acknowledgments also accepted")
	})
	t.Run("timestamp_cursor_skips_tie", func(t *testing.T) {
		at := time.Date(2026, 9, 4, 12, 0, 0, 123, time.UTC)
		for _, id := range []string{"po_review_a", "po_review_b"} {
			_, err := st.CreatePost(ctx, store.CreatePostInput{Subforum: "review", Post: &models.Post{ID: id, ThreadID: th.ID, AuthorID: b.ID, Body: id, CreatedAt: at}})
			if err != nil {
				t.Fatal(err)
			}
		}
		posts, err := st.ListPosts(ctx, th.ID, "po_review_a", 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range posts {
			if p.ID == "po_review_b" {
				t.Fatal("expected current cursor to skip tied post")
			}
		}
		t.Log("OBSERVED: ListPosts after po_review_a omits po_review_b with identical timestamp")
	})
	t.Run("scope_precedence_masks_watch", func(t *testing.T) {
		events, _, err := svc.PollEvents(ctx, a, service.PollEventsOptions{Scope: "watching"})
		if err != nil {
			t.Fatal(err)
		}
		if len(events) != 0 {
			t.Fatalf("want observed empty watching scope, got %d", len(events))
		}
		t.Log("OBSERVED: participant + watcher receives no other-author posts for scope=watching")
	})
	t.Run("idempotency_save_failure_duplicates", func(t *testing.T) {
		_, err := st.DB().ExecContext(ctx, `CREATE TRIGGER review_fail_idempotency BEFORE INSERT ON idempotency_keys BEGIN SELECT RAISE(FAIL, 'review injected idempotency failure'); END`)
		if err != nil {
			t.Fatal(err)
		}
		in := service.CreatePostInput{ThreadID: th.ID, Body: "retry", IdempotencyKey: "same-request"}
		p1, e1 := svc.CreatePost(ctx, a, in)
		p2, e2 := svc.CreatePost(ctx, a, in)
		if e1 != nil || e2 != nil || p1.ID == p2.ID {
			t.Fatalf("expected two successful distinct posts: %v %v", e1, e2)
		}
		t.Log("OBSERVED: failed idempotency save is ignored; identical retry creates a second post")
		if _, err := st.DB().ExecContext(ctx, `DROP TRIGGER review_fail_idempotency`); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("idempotency_key_is_global", func(t *testing.T) {
		for _, agent := range []*models.Agent{a, b} {
			err := st.SaveIdempotencyRecord(ctx, &store.IdempotencyRecord{Key: "shared-key", AgentID: agent.ID, Entity: "post", EntityID: "unused", Response: "{}", CreatedAt: time.Now()})
			if err != nil {
				t.Fatal(err)
			}
		}
		if _, err := st.GetIdempotencyRecord(ctx, "shared-key", a.ID); err == nil {
			t.Fatal("expected first agent record to be replaced")
		}
		t.Log("OBSERVED: second agent using same key replaces first agent's replay record")
	})
	t.Run("body_cap_silently_ignores_suffix", func(t *testing.T) {
		prefix := `{"name":"cap-probe"}`
		body := prefix + strings.Repeat(" ", (1<<20)-len(prefix)) + "INVALID SUFFIX"
		req := httptest.NewRequest(http.MethodPost, "/v1/agents/register", strings.NewReader(body))
		rec := httptest.NewRecorder()
		server.New(svc).ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected prefix accepted as 201: %d %s", rec.Code, rec.Body.String())
		}
		t.Log("OBSERVED: body over 1 MiB accepted when first MiB is valid JSON plus whitespace")
	})
}
