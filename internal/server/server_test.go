package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/agentforum/internal/server"
	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
)

// newTestServer builds a server over a fresh SQLite database and returns
// the httptest.Server plus helpers. The suite drives the API exactly like
// an external client: HTTP requests, protojson bodies, bearer tokens.
func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := service.NewService(st)
	srv := server.New(svc)
	ts := httptest.NewServer(srv)
	return ts, func() {
		ts.Close()
		_ = svc.Close()
	}
}

type client struct {
	t     *testing.T
	base  string
	token string
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal body: %v", err)
		}
		rd = bytes.NewReader(data)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.t.Fatalf("%s %s: decode response: %v", method, path, err)
	}
	return resp.StatusCode, out
}

func get[T any](m map[string]any, key string) T {
	v, _ := m[key].(T)
	return v
}

func str(m map[string]any, key string) string { return get[string](m, key) }

// register creates an agent via the API and returns a token-authenticated
// client plus the agent id.
func register(t *testing.T, ts *httptest.Server, name string) (*client, string) {
	t.Helper()
	c := &client{t: t, base: ts.URL}
	code, out := c.do("POST", "/v1/agents/register", map[string]any{
		"schemaVersion": 1,
		"name":          name,
	})
	if code != http.StatusCreated {
		t.Fatalf("register %s: status %d, body %v", name, code, out)
	}
	token := str(out, "token")
	if !strings.HasPrefix(token, "af_") {
		t.Fatalf("register %s: token %q missing af_ prefix", name, token)
	}
	agent := get[map[string]any](out, "agent")
	if str(agent, "id") == "" || str(agent, "name") != name {
		t.Fatalf("register %s: bad agent %v", name, agent)
	}
	// token never appears in the agent object
	if _, ok := agent["token"]; ok {
		t.Fatalf("register %s: token leaked into agent object", name)
	}
	return &client{t: t, base: ts.URL, token: token}, str(agent, "id")
}

func TestRegisterAuthAndMe(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	alice, _ := register(t, ts, "alice")

	// duplicate name -> 409 conflict envelope
	c := &client{t: t, base: ts.URL}
	code, out := c.do("POST", "/v1/agents/register", map[string]any{"name": "alice"})
	if code != http.StatusConflict || str(out, "code") != "conflict" {
		t.Fatalf("duplicate register: status %d, body %v", code, out)
	}

	// missing token -> 401
	code, out = c.do("GET", "/v1/me", nil)
	if code != http.StatusUnauthorized || str(out, "code") != "unauthenticated" {
		t.Fatalf("no token: status %d, body %v", code, out)
	}

	// bad token -> 401
	bad := &client{t: t, base: ts.URL, token: "af_notarealtoken"}
	code, _ = bad.do("GET", "/v1/me", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("bad token: status %d", code)
	}

	// GET /v1/me with a good token
	code, out = alice.do("GET", "/v1/me", nil)
	if code != http.StatusOK || str(get[map[string]any](out, "agent"), "name") != "alice" {
		t.Fatalf("me: status %d, body %v", code, out)
	}

	// PATCH /v1/me with metadata
	code, out = alice.do("PATCH", "/v1/me", map[string]any{
		"schemaVersion": 1,
		"metadata":      map[string]any{"model": "test-1"},
	})
	if code != http.StatusOK {
		t.Fatalf("patch me: status %d, body %v", code, out)
	}
	agent := get[map[string]any](out, "agent")
	meta := get[map[string]any](agent, "metadata")
	if str(meta, "model") != "test-1" {
		t.Fatalf("patch me: metadata not updated: %v", meta)
	}

	// GET /v1/agents/{name}
	code, out = alice.do("GET", "/v1/agents/bob", nil)
	if code != http.StatusNotFound {
		t.Fatalf("unknown agent: status %d", code)
	}
	code, out = alice.do("GET", "/v1/agents/alice", nil)
	if code != http.StatusOK || str(get[map[string]any](out, "agent"), "name") != "alice" {
		t.Fatalf("get agent: status %d, body %v", code, out)
	}
}

func TestForumFlowOverHTTP(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	alice, aliceID := register(t, ts, "alice")
	bob, _ := register(t, ts, "bob")

	// create a subforum
	code, out := alice.do("POST", "/v1/subforums", map[string]any{
		"schemaVersion": 1,
		"key":           "engineering",
		"title":         "Engineering",
		"description":   "eng topics",
	})
	if code != http.StatusCreated {
		t.Fatalf("create subforum: status %d, body %v", code, out)
	}
	sf := get[map[string]any](out, "subforum")
	if str(sf, "key") != "engineering" {
		t.Fatalf("subforum: %v", sf)
	}

	// bad subforum key -> 422
	code, out = alice.do("POST", "/v1/subforums", map[string]any{"key": "Bad Key!"})
	if code != http.StatusUnprocessableEntity || str(out, "code") != "invalid_argument" {
		t.Fatalf("bad subforum key: status %d, body %v", code, out)
	}

	// create a thread with an opening post (Idempotency-Key header)
	code, out = alice.do("POST", "/v1/subforums/engineering/threads", map[string]any{
		"schemaVersion": 1,
		"title":         "Caching investigation",
		"metadata":      map[string]any{"ticket": "PLAT-431"},
		"initialPost":   map[string]any{"body": "Tracing invalidation."},
		"watch":         true,
	})
	if code != http.StatusCreated {
		t.Fatalf("create thread: status %d, body %v", code, out)
	}
	thread := get[map[string]any](out, "thread")
	threadID := str(thread, "id")
	post := get[map[string]any](out, "initialPost")
	if !strings.HasPrefix(threadID, "th_") || !strings.HasPrefix(str(post, "id"), "po_") {
		t.Fatalf("ids: thread %q post %q", threadID, str(post, "id"))
	}
	// protojson shape: postCount is a string, participating true
	if get[string](thread, "postCount") != "1" {
		t.Fatalf("postCount: %v", thread["postCount"])
	}
	if get[bool](thread, "participating") != true || get[bool](thread, "watching") != true {
		t.Fatalf("thread perspective: %v", thread)
	}

	// bob watches the thread, alice replies
	code, _ = bob.do("PUT", fmt.Sprintf("/v1/threads/%s/watch", threadID), nil)
	if code != http.StatusOK {
		t.Fatalf("bob watch: status %d", code)
	}
	code, out = alice.do("POST", fmt.Sprintf("/v1/threads/%s/posts", threadID), map[string]any{
		"schemaVersion": 1,
		"body":          "The cache key is missing the locale.",
	})
	if code != http.StatusCreated {
		t.Fatalf("create post: status %d, body %v", code, out)
	}
	reply := get[map[string]any](out, "post")
	if str(reply, "authorName") != "alice" || str(reply, "authorId") != aliceID {
		t.Fatalf("post author denormalization: %v", reply)
	}

	// bob polls the inbox: 3 events, all reason EVENT_REASON_WATCHING
	code, out = bob.do("GET", "/v1/events?cursor=0&wait=0", nil)
	if code != http.StatusOK {
		t.Fatalf("poll: status %d, body %v", code, out)
	}
	events := get[[]any](out, "events")
	if len(events) != 3 {
		t.Fatalf("poll: want 3 events, got %v", events)
	}
	first := events[0].(map[string]any)
	if str(first, "reason") != "EVENT_REASON_WATCHING" || str(first, "actorName") != "alice" {
		t.Fatalf("event shape: %v", first)
	}
	if str(first, "threadTitle") != "Caching investigation" {
		t.Fatalf("event threadTitle: %v", first)
	}
	if get[string](out, "nextCursor") != "3" {
		t.Fatalf("nextCursor: %v", out["nextCursor"])
	}

	// self-exclusion: alice's own poll returns the empty marker with the
	// cursor still advanced
	code, out = alice.do("GET", "/v1/events?cursor=0&wait=0", nil)
	if code != http.StatusOK || len(get[[]any](out, "events")) != 0 || get[string](out, "nextCursor") != "3" {
		t.Fatalf("alice self-exclusion: %v", out)
	}

	// list threads in the subforum, with stats and perspective
	code, out = bob.do("GET", "/v1/threads?subforum=engineering", nil)
	if code != http.StatusOK {
		t.Fatalf("list threads: status %d, body %v", code, out)
	}
	threads := get[[]any](out, "threads")
	if len(threads) != 1 {
		t.Fatalf("list threads: %v", threads)
	}
	listed := threads[0].(map[string]any)
	if get[string](listed, "postCount") != "2" || get[bool](listed, "watching") != true {
		t.Fatalf("listed thread: %v", listed)
	}

	// list posts with author names
	code, out = bob.do("GET", fmt.Sprintf("/v1/threads/%s/posts", threadID), nil)
	if code != http.StatusOK {
		t.Fatalf("list posts: status %d, body %v", code, out)
	}
	posts := get[[]any](out, "posts")
	if len(posts) != 2 {
		t.Fatalf("posts: %v", posts)
	}

	// search with metadata filter
	code, out = bob.do("POST", "/v1/search", map[string]any{
		"schemaVersion": 1,
		"metadata":      map[string]any{"ticket": "PLAT-431"},
	})
	if code != http.StatusOK {
		t.Fatalf("search: status %d, body %v", code, out)
	}
	hits := get[[]any](out, "hits")
	if len(hits) != 1 || str(hits[0].(map[string]any), "entityType") != "thread" {
		t.Fatalf("search hits: %v", hits)
	}

	// ack
	code, out = bob.do("POST", "/v1/events/ack", map[string]any{
		"schemaVersion": 1, "throughSequence": "3",
	})
	if code != http.StatusOK || get[string](out, "throughSequence") != "3" {
		t.Fatalf("ack: status %d, body %v", code, out)
	}

	// healthz (no auth)
	code, _ = (&client{t: t, base: ts.URL}).do("GET", "/healthz", nil)
	if code != http.StatusOK {
		t.Fatalf("healthz: status %d", code)
	}
}

func TestIdempotencyOverHTTP(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()
	alice, _ := register(t, ts, "alice")

	_, _ = alice.do("POST", "/v1/subforums", map[string]any{"key": "eng"})

	body := map[string]any{
		"schemaVersion":  1,
		"title":          "first",
		"initialPost":    map[string]any{"body": "open"},
		"idempotencyKey": "run-1",
	}
	code, first := alice.do("POST", "/v1/subforums/eng/threads", body)
	if code != http.StatusCreated {
		t.Fatalf("create 1: status %d, body %v", code, first)
	}
	// retry with different title but same key -> same first result
	body["title"] = "DIFFERENT"
	code, second := alice.do("POST", "/v1/subforums/eng/threads", body)
	if code != http.StatusCreated {
		t.Fatalf("create 2: status %d, body %v", code, second)
	}
	firstID := str(get[map[string]any](first, "thread"), "id")
	secondID := str(get[map[string]any](second, "thread"), "id")
	if firstID != secondID {
		t.Fatalf("idempotency: first %q second %q", firstID, secondID)
	}

	// exactly one thread exists
	code, out := alice.do("GET", "/v1/threads?subforum=eng", nil)
	if code != http.StatusOK || len(get[[]any](out, "threads")) != 1 {
		t.Fatalf("thread count: %v", out)
	}
}

// TestLongPollDelivery mirrors the service-level concurrency test: a poll
// with a wait budget blocks until another agent posts.
func TestLongPollDelivery(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	alice, _ := register(t, ts, "alice")
	bob, _ := register(t, ts, "bob")

	_, _ = alice.do("POST", "/v1/subforums", map[string]any{"key": "eng"})
	_, out := alice.do("POST", "/v1/subforums/eng/threads", map[string]any{
		"title":       "t",
		"initialPost": map[string]any{"body": "open"},
	})
	threadID := str(get[map[string]any](out, "thread"), "id")
	_, _ = bob.do("PUT", fmt.Sprintf("/v1/threads/%s/watch", threadID), nil)

	// Non-blocking poll first: bob drains the 2 events created so far and
	// learns the cursor. The long-poll below then starts caught-up, which
	// is the only state in which it blocks.
	_, out = bob.do("GET", "/v1/events?cursor=0&wait=0", nil)
	cursor := get[string](out, "nextCursor")
	if cursor != "2" {
		t.Fatalf("drain poll cursor: %v", out)
	}

	start := time.Now()
	var pollStatus int
	var pollBody map[string]any
	done := make(chan struct{})
	go func() {
		defer close(done)
		pollStatus, pollBody = bob.do("GET", "/v1/events?cursor="+cursor+"&wait=2", nil)
	}()

	time.Sleep(150 * time.Millisecond)
	_, _ = alice.do("POST", fmt.Sprintf("/v1/threads/%s/posts", threadID), map[string]any{
		"schemaVersion": 1, "body": "reply",
	})

	<-done
	elapsed := time.Since(start)
	if pollStatus != http.StatusOK {
		t.Fatalf("poll status %d", pollStatus)
	}
	if events := get[[]any](pollBody, "events"); len(events) != 1 {
		t.Fatalf("poll events: %v", pollBody)
	}
	if get[string](pollBody, "nextCursor") != "3" {
		t.Fatalf("poll nextCursor: %v", pollBody)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("long-poll returned after the full deadline: %v", elapsed)
	}
}

// TestUnknownFieldsAccepted pins decision R2: a client sending newer
// schema fields gets a normal response, not a 400.
func TestUnknownFieldsAccepted(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()
	alice, _ := register(t, ts, "alice")

	code, out := alice.do("POST", "/v1/subforums", map[string]any{
		"schemaVersion":   1,
		"key":             "eng",
		"someFutureField": map[string]any{"nested": true},
	})
	if code != http.StatusCreated {
		t.Fatalf("unknown field: status %d, body %v", code, out)
	}
}

// TestListPostsPagination (A5, AGENTFORUM-005): the after cursor and limit
// travel as query parameters; the service already accepted them — this pins
// the HTTP surface and the page-advance behavior.
func TestListPostsPagination(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	alice, _ := register(t, ts, "alice")
	code, _ := alice.do("POST", "/v1/subforums", map[string]any{
		"schemaVersion": 1, "key": "eng", "title": "Eng",
	})
	if code != http.StatusCreated {
		t.Fatalf("create subforum: %d", code)
	}
	code, out := alice.do("POST", "/v1/subforums/eng/threads", map[string]any{
		"schemaVersion": 1, "title": "t", "initialPost": map[string]any{"body": "0"},
	})
	if code != http.StatusCreated {
		t.Fatalf("create thread: %d, %v", code, out)
	}
	threadID := str(get[map[string]any](out, "thread"), "id")

	// opening post + 4 replies = 5 posts total.
	for i := 1; i <= 4; i++ {
		code, _ = alice.do("POST", fmt.Sprintf("/v1/threads/%s/posts", threadID), map[string]any{
			"schemaVersion": 1, "body": fmt.Sprintf("reply %d", i),
		})
		if code != http.StatusCreated {
			t.Fatalf("post %d: %d", i, code)
		}
	}

	list := func(query string) []string {
		code, out := alice.do("GET", fmt.Sprintf("/v1/threads/%s/posts%s", threadID, query), nil)
		if code != http.StatusOK {
			t.Fatalf("list posts %q: status %d, body %v", query, code, out)
		}
		posts := get[[]any](out, "posts")
		ids := make([]string, 0, len(posts))
		for _, p := range posts {
			ids = append(ids, str(p.(map[string]any), "id"))
		}
		return ids
	}

	page1 := list("?limit=2")
	if len(page1) != 2 {
		t.Fatalf("page 1: want 2 posts, got %d", len(page1))
	}
	page2 := list(fmt.Sprintf("?limit=2&after=%s", page1[1]))
	if len(page2) != 2 {
		t.Fatalf("page 2: want 2 posts, got %d", len(page2))
	}
	if page2[0] == page1[0] || page2[0] == page1[1] {
		t.Fatalf("page 2 overlaps page 1: %v vs %v", page2, page1)
	}
	page3 := list(fmt.Sprintf("?limit=2&after=%s", page2[1]))
	if len(page3) != 1 {
		t.Fatalf("page 3: want 1 post, got %d", len(page3))
	}
	// an unknown after cursor is a 404 (service maps ErrNoRows)
	code, out = alice.do("GET", fmt.Sprintf("/v1/threads/%s/posts?after=po_nonexistent", threadID), nil)
	if code != http.StatusNotFound || str(out, "code") != "not_found" {
		t.Fatalf("unknown after: status %d, body %v", code, out)
	}
}
