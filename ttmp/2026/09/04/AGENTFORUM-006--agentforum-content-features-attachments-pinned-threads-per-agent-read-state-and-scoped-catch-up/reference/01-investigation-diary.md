---
Title: Investigation diary
Ticket: AGENTFORUM-006
Status: active
Topics:
    - forum
    - agents
    - attachments
    - read-state
    - pinning
    - http-api
    - go
    - cli
    - web-ui
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological record for the content-features ticket."
LastUpdated: 2026-09-04T00:10:00-04:00
WhatFor: "Record investigation, design decisions, and implementation steps for attachments, pinning, read state, and catch-up."
WhenToUse: "Read before implementing or reviewing AGENTFORUM-006."
---

# Investigation diary

## Goal

Record the investigation and design work for AGENTFORUM-006: post
attachments, pinned threads/posts, per-agent read state, and the
one-command scoped catch-up.

## Step 1: Ticket creation and preliminary design (hand-off draft)

Created the ticket and wrote a deliberately preliminary design doc
for the in-house expert to expand, correct, and improve. No code
changed. All decision records carry status PROPOSED; every feature
section ends with explicit open questions for the reviewer.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket with the features I asked for. Create a preliinary design doc, we'll hand it over to our in house expert to expand / correct / improve."

**Assistant interpretation:** Turn the three feature questions answered in conversation (attachments; pinned threads/comments; per-agent read tracking plus the one-command scoped catch-up) into a ticket whose design doc is an explicitly preliminary draft for expert review — not an implementation contract.

**Inferred user intent:** Capture the requirements durably with a first-cut design that takes positions where useful (so the reviewer edits something rather than a blank page), while leaving the real decisions explicitly open.

### What I did
- Added vocabulary topics (attachments, read-state, pinning); created ticket AGENTFORUM-006 with design doc + diary.
- Wrote the preliminary design doc: §2 records the verified current state (no attachments/pinning/read-position; ack + browser cursor semantics; the events-poll scoped-catch-up that already exists); §3–§6 give first-cut designs for pinning (pinned_at columns, ordering, events, any-agent-may-pin position), read state (read_state table, monotonic mark-read, orthogonal to the ack), catch-up (server-side aggregate endpoint, union scope, --mark-read), attachments (SQLite BLOBs, base64-in-protojson with a per-endpoint raised cap, 2 MiB, immutable lifecycle); §7 phases (pins → read state → catch-up → attachments); §8 test sketch; §9 four PROPOSED decision records; §10 file map. Every feature section carries Open questions for the reviewer.
- Anchored the current-state claims in the code as of ae2d3aa (including the 1 MiB body cap in internal/server/json.go that any attachment transport must reckon with).

### Why
The user wants the expert to own the design; the draft's job is to be concrete enough to correct.

### What worked
- Writing the conversation's prior answers (current state, the ~80% catch-up assessment, the global-vs-per-channel cursor distinction) directly into §2 meant the doc started from verified facts, not re-derivation.

### What didn't work
- Nothing (documentation phase).

### What I learned
- The catch-up feature's central question (subscription scope vs engagement scope) was already visible in the conversation answer; the doc formalizes it as D3 and leans union, flagged for the reviewer.

### What was tricky to build
- Keeping the doc honestly preliminary: taking a position (e.g., BLOBs over a disk store, base64 over multipart) while marking it contestable, instead of hedging everything into uselessness.

### What warrants a second pair of eyes
- D4 (attachment transport) is flagged in the doc itself as the most contestable position; D2 leaves three coexisting cursor mechanisms (ack, browser localStorage, read_state) — the reviewer should decide whether to consolidate.

### What should be done in the future
- Hand the doc to the in-house expert; their review should convert the PROPOSED decision records to accepted/rejected and resolve the open questions; implementation phases then follow §7 (or the reviewer's reordering).

### Code review instructions
- Start: the design doc §2 (current state), §9 (proposed decisions).
- Validate: `docmgr doctor --ticket AGENTFORUM-006`.

### Technical details
- Ticket path: ttmp/2026/09/04/AGENTFORUM-006--agentforum-content-features-attachments-pinned-threads-per-agent-read-state-and-scoped-catch-up.
