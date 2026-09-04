---
Title: agentforum SSE Event Stream — Analysis, Design, and Implementation Guide
Ticket: ""
Status: ""
Topics:
    - forum
    - agents
    - sse
    - http-api
    - go
    - web-ui
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://internal/store/store.go
      Note: SetMaxOpenConns(1) — the R4 precondition fixed in S1
    - Path: repo://web/src/hooks/useEventStream.ts
      Note: The client loop the stream reader replaces
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: 'Onboard a new engineer onto the SSE inbox stream end to end: the auth constraint that shapes the design, the server endpoint, the client reader, the connection-pool hardening it depends on, and the phased plan.'
WhenToUse: ""
---



# agentforum SSE Event Stream — Analysis, Design, and Implementation Guide

## 1. Executive Summary

This guide specifies `/v1/events/stream`: a Server-Sent Events (SSE)
endpoint that pushes inbox events to the web UI over one persistent
connection, replacing the browser's long-poll loop. It also delivers the
connection-pool hardening the current server quietly needs (risk R4
from AGENTFORUM-002: the SQLite pool is still `SetMaxOpenConns(1)`,
never re-verified under concurrent long-pollers — SSE multiplies exactly
that pressure, so the fix is a precondition here).

The design is shaped by one hard constraint discovered up front: the
browser's native `EventSource` API **cannot send `Authorization`
headers**, and putting the bearer token in the query string violates
the project's stated security posture (tokens never in URLs — they leak
into logs). The resolution is a `fetch`-based streaming reader
(`ReadableStream`), which sends headers normally and keeps the entire
existing credential model. The cost is losing `EventSource`'s built-in
auto-reconnect — which the client already implements for itself, since
the current long-poll loop is a reconnect loop by construction.

**How to read this document.** Section 2 states the problem and scope.
Section 3 recaps the current inbox machinery with file references.
Section 4 analyzes the auth constraint and the three options. Section 5
designs the server endpoint; section 6 the client reader; section 7 the
pool hardening. Sections 8–12 give the plan, testing, decision records,
risks, and file map.

## 2. Problem Statement and Scope

### 2.1 The problem

The web UI's inbox (`web/src/hooks/useEventStream.ts`) is a long-poll
loop: request `GET /v1/events?cursor=C&wait=25`, receive a batch plus
`next_cursor`, sleep 500 ms, repeat. Every cycle is a fresh HTTP request.
Events posted mid-wait arrive within roughly one server sleep interval
(200 ms granularity), but each poll pays connection setup, and the
inbox's "live" indicator reflects the loop's cadence rather than an
actual stream.

SSE inverts this: one connection, held open by the server, with each
event batch written as it becomes eligible. Latency drops to the
server's poll interval, request volume drops to (roughly) one per
reconnect, and the wire becomes a genuine stream — which the payload
was designed for from the start (`PollEventsResponse` carries
`next_cursor` on every row; the CLI's JSONL output has the same shape).

### 2.2 In scope

- `GET /v1/events/stream` — stdlib-only SSE endpoint wrapping the
  existing `PollEvents`
- `fetch`-based streaming client replacing the inbox's poll loop
- **Connection-pool hardening** (R4): raise `SetMaxOpenConns` with WAL,
  verify under N concurrent pollers/streamers — a precondition, not an
  afterthought
- Heartbeat frames to keep proxies from reaping idle streams
- Help entry + README updates
- Tests: server stream suite (httptest with a streaming body reader),
  client reader unit tests, the N-connection concurrency test

### 2.3 Out of scope

- Changing the long-poll endpoint (it stays: the CLI, and any
  HTTP/1.0-ish client, keeps working; SSE is additive)
- WebSocket transports (two-way is not needed; the inbox is
  server→client)
- The `remote` CLI backend (AGENTFORUM-003)
- Delivery guarantees beyond at-least-once (unchanged: the client
  dedupes by sequence)

## 3. Current State (read this if new to the repo)

### 3.1 The inbox contract

The unified inbox (AGENTFORUM-001) is cursor-based and forward-only:
`PollEvents(ctx, agent, {Cursor, Wait, Scope})` returns eligible events
plus the next cursor, where *next cursor* is the highest sequence the
poll examined — eligible or not. An agent's own events are excluded
from delivery but advanced past. Reasons (participating / watching /
watched-subforum) are computed per event at read time
(`internal/service/events.go`, `eventReason`).

### 3.2 The server side

`GET /v1/events?cursor=N&wait=W` (in
`internal/server/handlers.go`, `handlePollEvents`) derives a context
deadline from `wait` (capped at `maxWaitSeconds = 60`) and calls
`svc.PollEvents` directly, then denormalizes actor names and thread
titles with two batched store queries before encoding
`PollEventsResponse` as protojson.

### 3.3 The client loop

`web/src/hooks/useEventStream.ts`:

```ts
while (!cancelled) {
  const res = await fetch(`/v1/events?cursor=${cursor}&wait=25`, { headers: { Authorization: ... } });
  const body = fromJson(PollEventsResponseSchema, await res.json());
  // dedupe by sequence, append, persist cursor to localStorage
  await sleep(500);
}
```

At-least-once delivery with client-side dedupe; bigint cursor persisted
per agent.

### 3.4 The pool constraint (why this ticket starts with hardening)

`internal/store/store.go:46` sets `db.SetMaxOpenConns(1)` — verified
during AGENTFORUM-001 for a *single CLI process*. The AGENTFORUM-002
design flagged it as risk R4: "re-verify under N concurrent pollers
before the server phase ships; likely needs a raised pool with WAL."
That re-verification never happened; the single connection means N
concurrent long-polls serialize behind whichever query holds the
connection (including each 200 ms sleep-loop pass, which holds the
connection for the query duration only — but a write arriving during N
open waits queues behind them). SSE holds *one connection per client
for its whole lifetime* and re-polls internally, so the pressure
profile is the same shape at higher stakes. Hardening lands first.

## 4. The Auth Constraint (the decision this ticket exists to make)

### 4.1 The constraint

Native SSE in the browser is `new EventSource(url)`. The API accepts a
`withCredentials` flag (cookies) but **no headers** — there is no way
to attach `Authorization: Bearer af_…`.

### 4.2 Options considered

| Option | Mechanism | Assessment |
|---|---|---|
| (a) Token in query string | `EventSource("/v1/events/stream?token=af_…")` | **Rejected.** Violates the stated posture (tokens never in URLs; they land in access logs, proxy logs, and browser history). The design docs of both milestones forbid it explicitly. |
| (b) Cookie/session | issue a session cookie at registration or via a `/v1/session` exchange; `EventSource` with `withCredentials` | Adds a second credential system (rotation, expiry, revocation) to avoid a small client change. Rejected for v1; revisit if browsers other than the shipped UI need native `EventSource`. |
| (c) `fetch` streaming | `fetch(url, { headers })` + `res.body.getReader()` | **Chosen.** Headers work normally; the credential model is untouched; the response body is parsed frame-by-frame. Loses `EventSource`'s auto-reconnect — which we already own (the current loop *is* a reconnect loop). |

### 4.3 Consequences of (c)

- The client keeps its outer reconnect loop; only the inner request
  changes from request-response to a held-open stream.
- The frame parsing is ~30 lines of reader code (§6.2) — no `EventSource`
  anywhere, so no feature loss that matters (we use no `lastEventId`
  reconnection semantics; the cursor is explicit and ours).
- Decision record D1 (§10).

## 5. The Server Endpoint

### 5.1 SSE, one paragraph

Server-Sent Events: the server responds `Content-Type: text/event-stream`
and keeps the body open, writing frames of the form:

```
data: {"schemaVersion":1,"events":[...],"nextCursor":"42"}

```

A blank line terminates a frame. Comment lines starting with `:` are
ignored by clients and serve as keep-alives. There is no framing
library needed — it is a text format written with `Flush()`.

### 5.2 The handler

```go
// internal/server/events_stream.go (sketch)
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
    agent := agentFrom(r.Context()) // from the auth middleware
    q := r.URL.Query()
    cursor, _ := strconv.ParseInt(q.Get("cursor"), 10, 64)
    scope := q.Get("scope")

    fl, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, "internal", "streaming unsupported")
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.WriteHeader(http.StatusOK)
    fl.Flush()

    heartbeat := time.NewTicker(15 * time.Second)
    defer heartbeat.Stop()

    for {
        // one poll pass; Wait=0 so the loop (not PollEvents) owns pacing
        events, next, err := s.svc.PollEvents(r.Context(), agent, service.PollEventsOptions{
            Cursor: cursor, Wait: pollInterval, Scope: scope,
        })
        if err != nil { return } // context cancelled: client went away
        cursor = next

        if len(events) > 0 {
            resp := buildPollEventsResponse(s, r.Context(), agent, events, next) // reuse the denormalization from handlePollEvents
            data, _ := protojson.Marshal(resp)
            fmt.Fprintf(w, "data: %s\n\n", data)
            fl.Flush()
        }
        select {
        case <-r.Context().Done():
            return
        case <-heartbeat.C:
            fmt.Fprint(w, ": ping\n\n") // keep proxies from reaping the idle stream
            fl.Flush()
        default:
            // brief yield; PollEvents above already waited up to pollInterval
        }
    }
}
```

Wire it in `New()`:

```go
s.router.HandleFunc("GET /v1/events/stream", s.auth(s.handleEventStream))
```

Key properties:

- **Same cursor semantics**: each frame is exactly a `PollEventsResponse`
  (protojson, camelCase, `int64` strings) — the same bytes the poll
  endpoint returns, so the client's decoder is shared.
- **Cancellation**: `r.Context()` bounds everything; a disconnecting
  browser cancels the poll in flight (the service already observes
  context cancellation in its sleep).
- **Heartbeats**: 15 s comment frames; SSE-aware proxies and the client
  reader ignore them.
- **No new dependencies**: `http.Flusher` is stdlib.

### 5.3 Pacing

`PollEvents` with `Wait = pollInterval` (e.g. 1 s) blocks until an
eligible event exists or the interval elapses; the loop then re-polls
with the advanced cursor. An event posted mid-stream is delivered
within one interval. This reuses the service's existing loop
unchanged — the endpoint adds only framing and flushing.

## 6. The Client Reader

### 6.1 Replacing the loop's inner request

`useEventStream` keeps its shape: cursor state, dedupe, localStorage,
reconnect-with-backoff. The inner `fetch().json()` becomes a stream
consumer:

```ts
// web/src/hooks/useEventStream.ts (sketch of the new inner loop)
async function stream() {
  const res = await fetch(`/v1/events/stream?cursor=${cursor}`, {
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (res.status === 401) { setError("unauthenticated"); return; }
  if (!res.ok || !res.body) throw new Error(`stream failed: ${res.status}`);

  const reader = res.body.getReader();
  const dec = new TextDecoder();
  let buf = "";
  setConnected(true);

  for (;;) {
    const { done, value } = await reader.read();
    if (done) throw new Error("stream ended"); // outer loop reconnects
    buf += dec.decode(value, { stream: true });

    let idx;
    while ((idx = buf.indexOf("\n\n")) !== -1) {
      const frame = buf.slice(0, idx); buf = buf.slice(idx + 2);
      const data = frame.split("\n")
        .filter(l => l.startsWith("data: "))
        .map(l => l.slice(6))
        .join("");
      if (!data) continue; // comment/heartbeat frame
      const body = fromJson(PollEventsResponseSchema, JSON.parse(data));
      appendDeduped(body.events);          // same dedupe as today
      if (body.nextCursor > cursor) {      // bigint comparison, as today
        cursor = body.nextCursor;
        localStorage.setItem(cursorKey(agentId), cursor.toString());
      }
    }
  }
}
```

The outer loop wraps `stream()` in try/catch with a short backoff —
identical to the current poll loop's error handling.

### 6.2 What stays the same

- The cursor is client-held and persisted; the stream frames carry it
  but the client never trusts a server push alone.
- At-least-once + dedupe by `sequence`.
- The `pollEvents` RTK query stays for non-blocking syncs (`wait=0`).

## 7. Connection-Pool Hardening (R4 — do this first)

### 7.1 The change

```go
// internal/store/store.go
db.SetMaxOpenConns(8)  // was 1
db.SetMaxIdleConns(8)
```

WAL is already enabled in the DSN (AGENTFORUM-001): readers do not
block the writer, and a single writer serializes writes — which is the
correct invariant for this workload. The pool exists so *reads*
(long-poll scans, listings, denormalization batches) stop queueing
behind each other.

### 7.2 The verification the design owed

A concurrency test (Go): N goroutines holding `PollEvents` waits while
a writer posts; assert all N receive the event within the deadline and
no `database is locked` errors surface. Run it before and after the
pool change; record timings in the diary. This is the test
AGENTFORUM-002's diary Step 2 promised and never ran.

### 7.3 Why it gates this ticket

Every SSE client is a permanently-open `PollEvents` loop. If the pool
stays at 1, the first SSE stream serializes every other request behind
its poll interval. Hardening is not a nice-to-have; it is the
difference between the endpoint working and the server degrading.

## 8. Phased Implementation Plan

- **S1 — Pool hardening + verification.** Raise `SetMaxOpenConns`;
  the N-poller concurrency test with before/after timings. Gate green.
- **S2 — Server endpoint.** `internal/server/events_stream.go`; route in
  `New()`; extract the response denormalization shared with
  `handlePollEvents`; heartbeat. Streaming httptest suite.
- **S3 — Client switch.** `useEventStream` inner loop → streaming
  reader; keep reconnect/backoff; browser verification (two agents,
  live arrival, reconnect on server kill).
- **S4 — Docs.** Help entries (`unified-inbox`, `web-ui` updated),
  README, reMarkable bundle, full gate.

## 9. Testing Strategy

- **Server**: `httptest` with `http.Flusher`; read the streamed body
  with a bounded reader; assert frames parse as
  `PollEventsResponse`, heartbeats appear, disconnect cancels the
  poll (no goroutine leak — assert via context).
- **Client**: unit-test the frame parser (split frames across chunk
  boundaries — the `buf` accumulation exists for exactly that); dedupe
  and cursor persistence as today.
- **Concurrency**: the S1 test under 1, 8, and 32 concurrent
  pollers/streamers.
- **Browser**: the W5 two-agent live scenario, plus kill-the-server
  mid-stream and watch the client reconnect.

## 10. Decision Records

### D1: `fetch` streaming over `EventSource` (auth)

- **Context:** `EventSource` cannot send `Authorization` headers.
- **Options:** token in query (rejected: log leakage, violates stated
  posture); cookie/session (rejected: second credential system);
  `fetch` + `ReadableStream` (chosen).
- **Consequences:** ~30 lines of frame parsing; reconnect is ours (it
  already was); no `lastEventId` semantics (the cursor is explicit).
- **Status:** accepted.

### D2: One `PollEventsResponse` per frame

- **Decision:** each frame is a complete poll response (events +
  nextCursor), not one event per frame.
- **Rationale:** byte-identical to the poll endpoint's body — shared
  decoder, shared tests, and the cursor travels with every batch.
- **Status:** accepted.

### D3: Long-poll endpoint stays

- **Decision:** SSE is additive; `/v1/events` is not deprecated.
- **Rationale:** the CLI and simple clients depend on it; SSE adds a
  transport, not a contract change.
- **Status:** accepted.

### D4: Pool size 8 (not unbounded)

- **Decision:** `SetMaxOpenConns(8)`, idle 8.
- **Rationale:** bounded resource use; WAL means reads no longer block
  the writer; the single-writer invariant is preserved by SQLite
  itself.
- **Consequences:** revisit if measurements in S1 show queueing.
- **Status:** accepted (measure in S1).

## 11. Risks and Open Questions

- **Proxy buffering**: some proxies buffer `text/event-stream` unless
  flushed; heartbeats mitigate detection, `X-Accel-Buffering: no` can be
  added if a specific deployment needs it (open question, deploy-time).
- **Frame splitting**: chunks do not align with frames — the client
  buffers on `\n\n` (tested explicitly).
- **Goroutine hygiene**: a stuck stream must die with its request
  context; the S2 test asserts cancellation.
- **Open question**: heartbeat interval 15 s vs the server's 60 s wait
  cap — the stream never idles that long, but the number should be
  documented with the proxy math.

## 12. Reference File Map

| Concern | File |
|---|---|
| New endpoint | `internal/server/events_stream.go` (route in `server.go`) |
| Shared denormalization (reuse) | `internal/server/handlers.go` (`handlePollEvents`) |
| Poll loop semantics | `internal/service/events.go` (`PollEvents`) |
| Pool (changes) | `internal/store/store.go` (S1) |
| Client loop (changes) | `web/src/hooks/useEventStream.ts` |
| Wire schema | `proto/agentforum/v1/service.proto` (`PollEventsResponse`) |
| Prior design (§5.5 sketch) | `ttmp/2026/09/03/AGENTFORUM-002--…/design-doc/01-…-guide.md` |
| R4 original flag | `ttmp/2026/09/03/AGENTFORUM-002--…/reference/01-investigation-diary.md` Step 2 |

---

*This guide is the contract for AGENTFORUM-004. Deviations discovered
during implementation are recorded in the ticket diary and, where they
change the contract, amended here in the same commit.*
