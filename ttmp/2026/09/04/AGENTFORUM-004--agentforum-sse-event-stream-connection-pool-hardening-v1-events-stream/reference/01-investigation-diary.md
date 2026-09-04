---
Title: Investigation diary
Ticket: AGENTFORUM-004
Status: active
Topics:
    - forum
    - agents
    - sse
    - http-api
    - go
    - web-ui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/service/events_test.go
      Note: R4 verification test (Step 2)
ExternalSources: []
Summary: Chronological record for the SSE event stream phase.
LastUpdated: 2026-09-04T00:10:00-04:00
WhatFor: Record investigation, decisions, and implementation steps for the SSE stream.
WhenToUse: Read before implementing the SSE endpoint or the client switch.
---


# Investigation diary

## Goal

Record the investigation and design work for AGENTFORUM-004: the SSE
event stream and the connection-pool hardening it depends on.

## Step 1: Ticket creation and design guide

Created the ticket and wrote the intern-facing design guide. No code
changed; the deliverable is the implementation contract (S1–S4).

### Prompt Context

**User prompt (verbatim):** (see AGENTFORUM-003 diary Step 1 — same prompt)

**Assistant interpretation:** Create AGENTFORUM-004 with an intern-facing design guide, stored and uploaded.

**Inferred user intent:** See AGENTFORUM-003 Step 1.

### What I did
- Added vocabulary topics (`sse`); created ticket AGENTFORUM-004 with design doc + diary.
- Verified the R4 premise in code before designing around it: `internal/store/store.go:46` still sets `SetMaxOpenConns(1)` (the AGENTFORUM-002 design's "re-verify before the server ships" precondition never ran). Pool hardening is therefore phase S1, a precondition.
- Wrote the design guide: the auth analysis (EventSource cannot send headers → fetch/ReadableStream chosen over query-token and cookie options, D1), the server endpoint sketch (stdlib Flusher, heartbeat comments, one PollEventsResponse per frame), the client reader with chunk-boundary buffering, the pool change + the N-poller verification test the previous milestone owed, phased plan S1–S4, testing strategy, four decision records, risks, file map.

### Why
SSE was specified as a sketch (AGENTFORUM-002 §5.5); the auth constraint and the pool precondition are the two things that make it a ticket rather than a paragraph.

### What worked
- The constraint analysis resolved cleanly: the existing reconnect loop means losing EventSource auto-reconnect costs nothing.

### What didn't work
- Nothing failed (documentation phase).

### What I learned
- `EventSource`'s header limitation is the entire reason this design is fetch-based; the query-string option would violate the written security posture of both prior milestones.

### What was tricky to build
- N/A (documentation).

### What warrants a second pair of eyes
- Pool size 8 (D4) is a guess with a measurement plan; S1's before/after timings must be recorded before it becomes doctrine.
- Heartbeat interval (15 s) vs proxy behavior — deploy-time question, flagged in §11.

### What should be done in the future
- S1–S4 per the plan.

### Code review instructions
- Start: design doc §4 (auth options) and §7 (pool hardening).
- Validate: `docmgr doctor --ticket AGENTFORUM-004`.

### Technical details
- Ticket path: ttmp/2026/09/04/AGENTFORUM-004--agentforum-sse-event-stream-connection-pool-hardening-v1-events-stream.

## Step 2: S1 — pool raised to 8, N-poller verification run (R4 closed)

Wrote the concurrency test the AGENTFORUM-002 design promised, ran it
against the old single-connection pool and the new one, and raised the
pool. The fix is verified structurally and measured at test scale; the
measurement numbers and their honest interpretation are below.

### Prompt Context

**User prompt (verbatim):** "do  1. R4: connection pool under concurrent long-pollers — the most substantive one. The design said \"re-verify SetMaxOpenConns(1) under N concurrent pollers before the server phase ships; likely needs a raised pool with WAL.\"    The store still has SetMaxOpenConns(1) (store.go:46), and W7 shipped without that re-verification. With several browser tabs holding long-polls plus writes arriving, everything serializes on one connection. This was a stated shipping precondition that slipped through. and of these, which would you address / write down for a future ticket? C. Known limitations logged in the diaries (polish, not design scope) [list of 8 polish items]"

**Assistant interpretation:** Implement AGENTFORUM-004 phase S1 now (pool hardening + the N-poller test with before/after measurements), and triage the polish list into what to address vs. record for a future ticket.

**Inferred user intent:** Close the one genuinely owed engineering debt before starting new features, and keep the polish list from evaporating — get a concrete recommendation and a durable record.

### What I did
- Added `TestPollEventsConcurrentLongPollers` (`internal/service/events_test.go`): 16 watchers hold concurrent 5 s long-polls over a watched thread; the writer posts 8 replies while the polls are open; every watcher must receive `watching`-reason events inside the budget; per-watcher latencies are logged as a sorted distribution for cross-run comparison.
- Baseline run at `SetMaxOpenConns(1)` (3×, verbatim p50/max): 100.46ms/100.99ms, 103.89ms/104.56ms, 100.78ms/101.20ms. First-to-last watcher spread ≈ 2.7 ms. No store errors.
- Raised the pool: `db.SetMaxOpenConns(8); db.SetMaxIdleConns(8)` in `internal/store/store.go`, replacing the single-connection rationale comment with the WAL/pool rationale and a pointer to the test.
- Re-run at pool=8 (3×, verbatim p50/max): 99.48ms/99.68ms, 100.20ms/100.48ms, 101.95ms/102.39ms. First-to-last watcher spread ≈ 0.4 ms.
- Added `TestOpenPoolSettings` (`internal/store/store_test.go`): pins `MaxOpenConnections == 8` so the fix cannot silently regress.
- Full gate: gofmt clean, `GOWORK=off go test ./... -count=1` green, `go vet` ok, builds with and without `-tags embed` ok, and `-race` green across store/service/server (the concurrency test runs under the race detector).

### Why
R4 was a stated shipping precondition that slipped through W7. SSE (S2–S4) multiplies exactly this pressure, so the fix lands first.

### What worked
- The PRAGMAs were already in the DSN (`_pragma` values apply per pooled connection in the modernc driver), so raising the pool required no per-connection setup — the single-connection comment's migration concern was already solved by construction.
- The race detector over the 16-poller test gave a free deep validation of the store under concurrent access.

### What didn't work
- Nothing failed. The honest caveat: at test scale (empty DB, 500-event pages mostly empty), the absolute latency delta between pool settings is ~1 ms — the dominant 100 ms is the pollers' 200 ms wake cycle, which is by design.

### What I learned
- The measurable signal at test scale is not p50 (wake-cycle dominated) but the delivery spread across watchers: the 16 watchers' 64 queries per wake cycle stopped serializing on one connection (2.7 ms → 0.4 ms spread). At production scale (full 500-event pages, real membership sets), the same serialization would queue writes behind long reads — that structural risk is what the fix removes.
- `db.Stats().MaxOpenConnections` is assertable directly, making pool configuration a regression-testable property.

### What was tricky to build
- Measuring the right thing: latency must be measured from the first write's completion (after `CreatePost` returns), not from goroutine launch, or the numbers are wake-cycle noise. The `posted atomic.Value` timestamp set right after the first post does it.
- The old single-connection comment claimed PRAGMAs "persist on the connection" as a reason for pool=1; verifying that DSN pragmas apply per pooled connection removed the last argument for keeping it.

### What warrants a second pair of eyes
- Pool size 8 remains a reasoned default, not a measured optimum — D4 said "measure in S1"; at test scale no queueing appeared at either 1 or 8, so the number is not yet evidence-backed for production loads. Revisit if a deployment shows contention.

### What should be done in the future
- S2–S4 (SSE endpoint, client switch, docs) per the plan.

### Code review instructions
- Start: `internal/store/store.go` (pool change + rationale comment), `internal/service/events_test.go` (`TestPollEventsConcurrentLongPollers`), `internal/store/store_test.go` (`TestOpenPoolSettings`).
- Validate: `GOWORK=off go test ./internal/service/ -run TestPollEventsConcurrentLongPollers -count=1 -v` (observe the latency log line), then the full gate.

### Technical details
- Baseline command (pool=1, before the store change): `GOWORK=off go test ./internal/service/ -run TestPollEventsConcurrentLongPollers -count=1 -v`.
- Post-change command: same, plus `GOWORK=off go test -race ./internal/store/ ./internal/service/ ./internal/server/ -count=1`.

## Step 3: S2 — SSE endpoint; two real bugs found under it (one of them the AGENTFORUM-005 flake)

Implemented `GET /v1/events/stream` per the design, extracted the
shared response builder, and wrote the streaming test suite. Verifying
it under `-race` exposed two real bugs: a handler-return race in the
stream itself, and — the significant one — the store's DSN pragmas were
never applied, meaning the database has run in rollback-journal mode
with no busy timeout since AGENTFORUM-001.

### Prompt Context

**User prompt (verbatim):** "AGENTFORUM-004, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill), Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done."

**Assistant interpretation:** Implement AGENTFORUM-004 S2–S4 (S1 already done): the SSE endpoint, the client switch, docs — with the phase discipline.

**Inferred user intent:** Finish the SSE milestone with the same slip/diary/commit cadence as the earlier tickets.

### What I did
- Extracted `buildPollEventsResponse` from `handlePollEvents` (denormalization + conversion, shared by both endpoints — byte-identical payloads by construction).
- Wrote `internal/server/events_stream.go`: `text/event-stream`, one protojson `PollEventsResponse` per data frame, `: ping` comment frames, `streamPassWait=1s` passes, cursor advances exactly like the long-poll endpoint. Route `GET /v1/events/stream` behind the auth middleware; `heartbeatInterval` field on Server (default 15s, test-overridable in-package).
- Wrote `internal/server/events_stream_test.go` (in-package, so the heartbeat interval is reachable; minimal HTTP helpers duplicated from the external test package — different Go test packages cannot share symbols): delivery, heartbeat, client-disconnect (via `httptest.Server.Close` completing, which blocks on outstanding requests — a leaked handler would hang it).
- Fixed two bugs found under `-race` (below). Commits: `e3d1ec9` (store DSN fix), `ae9e1e3` (SSE endpoint + tests).

### Why
The design's S2 contract; the endpoint is the transport half of the milestone.

### What worked
- The disconnect-via-`ts.Close()` test design: it turns "the handler leaks" into "the test hangs", which is unambiguous.
- The store fix made the whole server suite measurably faster (~2.0s vs earlier runs that included 15s failure timeouts).

### What didn't work
- First heartbeat design: a `select { case <-heartbeat.C: ... default: }` inside the data loop starves the ticker — `PollEvents` blocks up to 1s per pass, so pings fired at pass granularity (1 ping in 350ms at a 50ms interval). Fixed with a dedicated heartbeat goroutine serialized through a mutex with the data writes.
- First version of that goroutine raced `-race` (`bufio.(*Writer)` vs `finishRequest`): the handler returned while the heartbeat goroutine was mid-write. Fixed with ordered defers — `close(done)` (signal), `heartbeat.Stop()`, then `<-hbDone` (wait) — so the handler cannot return before the goroutine exits.
- The delivery test failed ~50% under `-race` with the POST returning 500. Verbatim error after temporarily logging the elided message: `store: commit create post: database is locked (5) (SQLITE_BUSY)`.
- Root cause (the significant one): `buildDSN` called `url.Values.Set("_pragma", …)` three times — **Set replaces the previous value**, so the DSN only ever contained `foreign_keys=on`. Empirical probe before the fix: `journal_mode=delete`, `busy_timeout=0`. The database has run in rollback-journal mode with no busy timeout since AGENTFORUM-001. Concurrent readers (long-polls, listings, the new stream) made writers fail with SQLITE_BUSY → unmapped 500s. This is the root cause of the AGENTFORUM-005 CI flake (`TestLongPollDelivery` "poll status 500") whose cause was recorded there as unexplained.
- Test-side mistake while instrumenting: `t.Fatalf` inside the poster goroutine is lost (Goexit on the wrong goroutine) — the POST's 500 was invisible until its result went through a channel.
- Minor: `q.Set` fix needed the function-call pragma syntax (`journal_mode(WAL)`) and `foreign_keys(1)`; `TestOpenPragmasApply` pins all three (probe values: wal / 5000 / 1).

### What I learned
- `url.Values.Set` replaces; `Add` is the only way to build repeated query keys. Three Set calls on `"_pragma"` is a bug that type-checks, runs, and passes every functional test — until concurrency probes it.
- The AGENTFORUM-004 S1 diary entry's premise was wrong: it measured read parallelization on a rollback-journal database. The pool raise stands (reads stopped queueing — that part was real), but the "WAL already enabled" claim was false until this step. The AGENTFORUM-002 design's R4 ("likely needs a raised pool with WAL") was half-right: the pool was raised, the WAL never existed.
- http.ResponseWriter teardown races any concurrent write the moment the handler returns; goroutines that write must be joined before return, not just signalled.

### What was tricky to build
- Ordering the three defers so signal → stop → wait runs without deadlock (registration order reversed because defers are LIFO).
- Diagnosing through an elided error message: the 500 body says "internal error" by design (correct — do not leak internals), which meant the temp log line in `writeServiceError` was the only window into the store error. Removed after diagnosis.

### What warrants a second pair of eyes
- The heartbeat goroutine + mutex + ordered-defer join is the concurrency-critical core of the endpoint; the `-race` suite covers it, but a second reading of the defer order is cheap insurance.
- WAL is now active in production paths for the first time — all prior "WAL" claims in earlier docs/diaries were aspirational. The full suite passing on WAL is good evidence, but any WAL-specific behavior (checkpointing, -wal file growth on long-lived servers) should be watched on a real deployment.

### What should be done in the future
- S3 (client switch) next; S4 (docs + gate + bundle) after.

### Code review instructions
- Start: `internal/server/events_stream.go` (endpoint + the two concurrency comments), `internal/store/store.go` (`buildDSN` comment documents the bug), `internal/store/store_test.go` (`TestOpenPragmasApply`), `internal/server/events_stream_test.go`.
- Validate: `GOWORK=off go test -race ./internal/server/ -run "TestEventStream|TestLongPoll" -count=3` and `GOWORK=off go test ./internal/store/ -run TestOpenPragmasApply -count=1 -v`.

### Technical details
- Commits: e3d1ec9 (store), ae9e1e3 (endpoint). Failing probe before the fix: `journal_mode=delete busy_timeout=0`; after: `wal / 5000 / 1`.
- Race trace (abridged): `bufio.(*Writer).WriteString ← net/http.(*response).finishRequest ← (*conn).serve` concurrent with the heartbeat goroutine's write.
