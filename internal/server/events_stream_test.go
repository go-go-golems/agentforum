package server

// SSE stream tests (AGENTFORUM-004 S2). The stream is read incrementally
// from the response body with a bufio reader; frames are separated by a
// blank line, data frames start with "data: ", heartbeat frames with ":".

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/agentforum/internal/service"
	"github.com/go-go-golems/agentforum/internal/store"
)

// Minimal HTTP test helpers, duplicated from the external test package
// (server_test.go is package server_test; these tests are package server
// so they can shorten the unexported heartbeat interval).
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
// client.
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
	c.token = str(out, "token")
	return c, str(get[map[string]any](out, "agent"), "id")
}

// streamServer builds a dedicated test server and returns both the
// httptest server and the inner *Server (so tests can shorten the
// heartbeat interval).
func streamServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	svc := service.NewService(st)
	srv := New(svc)
	ts := httptest.NewServer(srv)
	t.Cleanup(func() {
		ts.Close()
		_ = svc.Close()
	})
	return ts, srv
}

// streamFixture registers alice+bob, creates a subforum and thread, bob
// watches, and drains bob's cursor so a stream started at that cursor is
// caught up and blocks until alice posts.
func streamFixture(t *testing.T) (*httptest.Server, *Server, *client, *client, string, string) {
	t.Helper()
	ts, srv := streamServer(t)

	alice, _ := register(t, ts, "alice")
	bob, _ := register(t, ts, "bob")

	_, _ = alice.do("POST", "/v1/subforums", map[string]any{"key": "eng"})
	_, out := alice.do("POST", "/v1/subforums/eng/threads", map[string]any{
		"title":       "t",
		"initialPost": map[string]any{"body": "open"},
	})
	threadID := str(get[map[string]any](out, "thread"), "id")
	_, _ = bob.do("PUT", "/v1/threads/"+threadID+"/watch", nil)

	_, out = bob.do("GET", "/v1/events?cursor=0&wait=0", nil)
	cursor := get[string](out, "nextCursor")
	return ts, srv, alice, bob, threadID, cursor
}

// readFrame reads one SSE frame (up to the blank line) and returns its
// data payload, or "" for comment frames.
func readFrame(t *testing.T, br *bufio.Reader) (data string, ok bool) {
	t.Helper()
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return "", false
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			break // frame boundary
		}
		lines = append(lines, line)
	}
	var payload strings.Builder
	for _, l := range lines {
		if strings.HasPrefix(l, "data: ") {
			payload.WriteString(strings.TrimPrefix(l, "data: "))
		}
	}
	return payload.String(), true
}

func openStream(t *testing.T, ts *httptest.Server, c *client, cursor string) (*http.Response, *bufio.Reader, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		ts.URL+"/v1/events/stream?cursor="+cursor, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("open stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("stream status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		cancel()
		t.Fatalf("content type %q", ct)
	}
	return resp, bufio.NewReader(resp.Body), cancel
}

func TestEventStreamDeliversEvents(t *testing.T) {
	ts, _, alice, bob, threadID, cursor := streamFixture(t)

	resp, br, cancel := openStream(t, ts, bob, cursor)
	defer cancel()
	defer resp.Body.Close()

	// Post while the stream is open; the frame must arrive promptly. The
	// result is reported on a channel: t.Fatalf inside a non-test goroutine
	// is lost (Goexit on the wrong goroutine), which hid the real failure
	// during development.
	postRes := make(chan int, 1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		code, out := alice.do("POST", "/v1/threads/"+threadID+"/posts", map[string]any{
			"schemaVersion": 1, "body": "streamed reply",
		})
		if code != http.StatusCreated {
			t.Errorf("poster: status %d, body %v", code, out)
		}
		postRes <- code
	}()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			select {
			case code := <-postRes:
				t.Fatalf("no data frame within 5s (post returned %d — the write landed but was not delivered)", code)
			default:
				t.Fatal("no data frame within 5s (the post never completed)")
			}
		default:
		}
		data, ok := readFrame(t, br)
		if !ok {
			t.Fatal("stream ended before a data frame")
		}
		if data == "" {
			continue // heartbeat
		}
		var frame map[string]any
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			t.Fatalf("frame is not JSON: %v (%s)", err, data)
		}
		events := get[[]any](frame, "events")
		if len(events) != 1 {
			t.Fatalf("want 1 event, got %v", frame)
		}
		if get[string](frame, "nextCursor") == cursor {
			t.Fatalf("cursor did not advance: %v", frame)
		}
		if reason := str(events[0].(map[string]any), "reason"); reason == "" {
			// eventToProto fills reason; its absence would mean a bare event
			t.Fatalf("event missing reason: %v", events[0])
		}
		return // success
	}
}

func TestEventStreamHeartbeat(t *testing.T) {
	ts, srv, _, bob, _, cursor := streamFixture(t)
	// Short heartbeat so the test observes multiple comment frames fast.
	srv.heartbeatInterval = 50 * time.Millisecond

	resp, br, cancel := openStream(t, ts, bob, cursor)
	defer cancel()
	defer resp.Body.Close()

	// ~350ms of idleness must produce several ": ping" frames.
	pings := 0
	deadline := time.Now().Add(350 * time.Millisecond)
	for time.Now().Before(deadline) {
		data, ok := readFrame(t, br)
		if !ok {
			break
		}
		if data == "" {
			pings++
		} else {
			t.Fatalf("unexpected data frame on idle stream: %s", data)
		}
	}
	if pings < 3 {
		t.Fatalf("want >=3 heartbeat frames in 350ms at 50ms interval, got %d", pings)
	}
}

func TestEventStreamClientDisconnect(t *testing.T) {
	ts, srv, _, bob, _, cursor := streamFixture(t)
	// Keep passes short so the handler notices the disconnect quickly.
	srv.heartbeatInterval = 50 * time.Millisecond

	resp, _, cancel := openStream(t, ts, bob, cursor)
	// Let the stream get past the initial flush, then disconnect.
	time.Sleep(150 * time.Millisecond)
	cancel()
	_ = resp.Body.Close()

	// httptest.Server.Close blocks until outstanding requests finish; if
	// the handler ignored the disconnect and kept streaming, this hangs.
	// (The fixture's t.Cleanup close is skipped: close manually here.)
	done := make(chan struct{})
	go func() {
		ts.Close()
		close(done)
	}()
	select {
	case <-done:
		return // handler exited with its request context
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down: the stream handler leaked past client disconnect")
	}
}
