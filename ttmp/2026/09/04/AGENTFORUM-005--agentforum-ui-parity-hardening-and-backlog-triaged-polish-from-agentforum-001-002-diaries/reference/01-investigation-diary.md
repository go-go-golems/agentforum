---
Title: Investigation diary
Ticket: AGENTFORUM-005
Status: active
Topics:
    - forum
    - agents
    - backlog
    - web-ui
    - go
    - cli
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological record for the backlog triage ticket."
LastUpdated: 2026-09-04T00:10:00-04:00
WhatFor: "Record triage decisions and any implementation work on backlog items."
WhenToUse: "Read before picking up a backlog item."
---

# Investigation diary

## Goal

Durable record and triage of every polish/limitation item logged in the
AGENTFORUM-001/002 diaries, so nothing evaporates and priorities are
explicit.

## Step 1: Ticket creation and triage

### Prompt Context

**User prompt (verbatim):** (see AGENTFORUM-004 diary Step 2 — same prompt, C-list)

**Assistant interpretation:** After doing R4 (S1), decide which diary polish items to address vs. record for a future ticket, and write the decision down.

**Inferred user intent:** Keep the backlog from evaporating; get a concrete recommendation rather than a vague "later".

### What I did
- Created AGENTFORUM-005 and wrote the triage doc: 5 items to address (subforum watch UI, profile editing, W7 token anomaly probe, CI workflows, ListPosts pagination cursor), 4 deferred (composer preview, UI component tests, AGENTFORUM-001 CLI leftovers, README drift), 2 dropped/accepted (math-in-fences, hover-intent bridge), with file references and an ordering recommendation.

### Why
The C-list items were only in diary prose; a ticket with triage decisions makes them schedulable and re-triageable.

### What worked
- The triage forced the CI gap to the top: it is the only item that protects all future work, and it subsumes the guide-drift check leftover.

### What didn't work
- Nothing failed (triage phase).

### What I learned
- Most of the C-list splits cleanly into "contract exists, surface missing" (cheap parity work) vs. "new behavior" (defer-able polish).

### What was tricky to build
- N/A.

### What warrants a second pair of eyes
- The X-bucket calls (math-in-fences, hover-intent) are judgment calls to accept behavior; revisit if either becomes a user complaint.

### What should be done in the future
- A3, A4 first; then A1+A2 as one UI-parity phase; then A5.

### Code review instructions
- Start: the triage doc §2–§5.
- Validate: `docmgr doctor --ticket AGENTFORUM-005`.

### Technical details
- Ticket path: ttmp/2026/09/04/AGENTFORUM-005--agentforum-ui-parity-hardening-and-backlog-triaged-polish-from-agentforum-001-002-diaries.

## Step 2: P1 (A3) — register-token anomaly probe; guard tightened

Wrote the probe the AGENTFORUM-002 diary suggested. The case analysis
proves the guarded property; one real looseness surfaced (the guard
accepted the bare prefix) and was fixed.

### Prompt Context

**User prompt (verbatim):** "User task: AGENTFORUM-005, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done."

**Assistant interpretation:** Implement the AGENTFORUM-005 backlog ticket end to end with per-phase slips, diary steps, commits, and docmgr bookkeeping.

**Inferred user intent:** Work through the triaged backlog (A3→A4→A1+A2→A5) with the same phase discipline as the earlier milestones.

### What I did
- Wrote `web/src/store/forumApi.test.ts`: five tests over `RegisterAgentResponseSchema` decoding and the `setToken` guard, with a `localStorage` stub (the vitest environment is `node`).
- Tightened the guard in `web/src/store/forumApi.ts`: `startsWith("af_")` → regex `^af_[A-Za-z0-9_-]+$` (real tokens are `af_` + 43 base64url chars per `internal/id/id.go` `NewToken`).
- Updated the triage doc's A3 section with the conclusion.

### Why
A3 was the owed verification: prove the af_ guard suffices or find the root cause.

### What worked
- The full case analysis is now pinned by tests: no-token response → proto3 default `""` → rejected; non-string token → `fromJson` throws → mutation rejects before `setToken`; valid token round-trips; symptom strings rejected.

### What didn't work
- First run: 4 tests failed. Verbatim errors: `Error: cannot decode message agentforum.v1.Agent from JSON: key "displayName" is unknown` (the proto `Agent` has no `displayName` — display data lives in metadata; fixed the fixture) and `AssertionError: expected 'af_' to be ''` — **the probe found a real bug**: the guard stored the bare prefix `"af_"`, which would then 401 on every call.

### What I learned
- protojson decoding cannot fabricate a token value: absent fields decode to proto3 defaults, wrong-typed fields throw. The only path to a poisoned credential slot was the guard's own looseness (or pre-guard code storing `"undefined"`).

### What was tricky to build
- Testing `localStorage` in the `node` vitest environment: the API slice touches it only inside functions (never at import time), so a `beforeAll` `Object.defineProperty(globalThis, "localStorage", ...)` stub suffices — no jsdom needed.

### What warrants a second pair of eyes
- The tightened regex rejects any token not matching `af_[A-Za-z0-9_-]+`. If the server ever changes token encoding (e.g. adds `=` padding from non-url-safe base64), the guard would reject logins; the coupling to `internal/id.NewToken` is documented in the guard comment.

### What should be done in the future
- A4 (CI workflow) next.

### Code review instructions
- Start: `web/src/store/forumApi.ts` (`TOKEN_RE` + guard), `web/src/store/forumApi.test.ts`.
- Validate: `pnpm --dir web test` (9 tests) and `pnpm --dir web check`.

### Technical details
- Commit: see changelog. Verbatim failing output excerpt: `Error: cannot decode message agentforum.v1.Agent from JSON: key "displayName" is unknown` / `AssertionError: expected 'af_' to be ''`.
