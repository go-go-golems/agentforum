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
