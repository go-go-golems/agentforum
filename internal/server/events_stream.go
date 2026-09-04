// SSE event stream (AGENTFORUM-004 S2).
//
// GET /v1/events/stream?cursor=N[&scope=...] pushes inbox events over one
// persistent Server-Sent Events connection: the response is
// text/event-stream, each frame is one protojson PollEventsResponse (the
// same bytes the long-poll endpoint returns, decision D2), and comment
// frames (": ping") keep proxies from reaping idle streams. The long-poll
// endpoint stays (decision D3): the CLI and simple clients depend on it.
//
// The browser client uses fetch + ReadableStream rather than EventSource,
// because EventSource cannot send Authorization headers and tokens never
// travel in query strings (decision D1, design §4).
package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	agentforumv1 "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1"
	"github.com/go-go-golems/agentforum/internal/models"
	"github.com/go-go-golems/agentforum/internal/service"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// streamPassWait is the per-pass PollEvents wait budget inside the
	// stream loop. The endpoint (not PollEvents) owns pacing: each pass
	// blocks until an eligible event exists or this budget elapses, then
	// the loop re-polls with the advanced cursor. One second keeps
	// end-to-end latency within roughly one service poll interval (the
	// service sleeps in 200ms chunks internally) while avoiding a busy
	// loop. The heartbeat runs on its own goroutine (see handleEventStream)
	// so pass length does not starve it.
	streamPassWait = time.Second

	// defaultHeartbeat is the comment-frame interval. SSE comment frames
	// are ignored by clients; their job is to keep intermediaries from
	// buffering or reaping an otherwise-idle connection.
	defaultHeartbeat = 15 * time.Second
)

// buildPollEventsResponse denormalizes actor names and thread titles for
// inbox display in two batched queries (never N+1) and converts to the
// wire message. Shared by the long-poll endpoint and the SSE stream so
// both emit byte-identical payloads.
func (s *Server) buildPollEventsResponse(ctx context.Context, events []*models.Event, nextCursor int64) (*agentforumv1.PollEventsResponse, error) {
	out := &agentforumv1.PollEventsResponse{
		SchemaVersion: 1,
		NextCursor:    nextCursor,
	}
	if len(events) == 0 {
		return out, nil
	}
	actorIDs := make([]string, 0, len(events))
	threadIDs := make([]string, 0, len(events))
	for _, ev := range events {
		actorIDs = append(actorIDs, ev.ActorID)
		threadIDs = append(threadIDs, ev.ThreadID)
	}
	names, err := s.svc.Store().AgentNames(ctx, actorIDs)
	if err != nil {
		return nil, err
	}
	titles, err := s.svc.Store().ThreadTitles(ctx, threadIDs)
	if err != nil {
		return nil, err
	}
	for _, ev := range events {
		out.Events = append(out.Events, eventToProto(ev, names[ev.ActorID], titles[ev.ThreadID]))
	}
	return out, nil
}

// handleEventStream serves GET /v1/events/stream. It never writes an
// error after the stream starts: a client disconnect cancels the request
// context, which unblocks the in-flight poll, and both goroutines return.
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	cursor, err := strconv.ParseInt(orDefault(q.Get("cursor"), "0"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "cursor must be an integer")
		return
	}
	scope := q.Get("scope") // comma-separated: involved,watching,watched-subforums
	agent := agentFrom(ctx)

	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	// http.ResponseWriter is not safe for concurrent use: the data loop and
	// the heartbeat goroutine serialize every write+flush through mu.
	var mu sync.Mutex
	writeFrame := func(frame string) bool {
		mu.Lock()
		defer mu.Unlock()
		if _, err := fmt.Fprint(w, frame); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	writeFrame("\n") // initial padding frame so headers flush to the client

	// Heartbeat goroutine: its own ticker, so the interval is exact
	// regardless of how long each poll pass blocks (a select in the data
	// loop would starve it — pings would only fire between passes).
	// The handler must not return while the goroutine can still write:
	// net/http tears the response down (finishRequest) as soon as the
	// handler returns, and that teardown races any in-flight write. The
	// ordered defers below guarantee the goroutine has exited first.
	hbDone := make(chan struct{})
	defer func() { <-hbDone }() // runs last: heartbeat goroutine has exited
	heartbeat := time.NewTicker(s.heartbeatInterval)
	defer heartbeat.Stop() // runs second
	done := make(chan struct{})
	defer close(done) // runs first: signal the goroutine
	go func() {
		defer close(hbDone)
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-heartbeat.C:
				if !writeFrame(": ping\n\n") {
					return
				}
			}
		}
	}()

	for {
		// One poll pass: blocks until an eligible event exists or
		// streamPassWait elapses. The client's cursor advances past
		// everything examined (forward-only inbox), even ineligible
		// events, exactly like the long-poll endpoint.
		events, next, err := s.svc.PollEvents(ctx, agent, service.PollEventsOptions{
			Cursor: cursor,
			Wait:   streamPassWait,
			Scope:  scope,
		})
		if err != nil {
			// The request context is the only cancellation source with no
			// deadline of our own: a client disconnect. Nothing to write.
			return
		}
		cursor = next

		if len(events) > 0 {
			resp, err := s.buildPollEventsResponse(ctx, events, next)
			if err != nil {
				return // context cancelled mid-denormalization or store error; stream ends
			}
			data, err := protojson.Marshal(resp)
			if err != nil {
				return
			}
			// One PollEventsResponse per frame (decision D2): byte-identical
			// to the long-poll body, so clients share one decoder.
			if !writeFrame("data: " + string(data) + "\n\n") {
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
			// PollEvents above already waited; loop immediately.
		}
	}
}
