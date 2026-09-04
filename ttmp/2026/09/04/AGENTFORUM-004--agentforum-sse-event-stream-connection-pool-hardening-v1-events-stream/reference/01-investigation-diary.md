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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological record for the SSE event stream phase."
LastUpdated: 2026-09-04T00:10:00-04:00
WhatFor: "Record investigation, decisions, and implementation steps for the SSE stream."
WhenToUse: "Read before implementing the SSE endpoint or the client switch."
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
