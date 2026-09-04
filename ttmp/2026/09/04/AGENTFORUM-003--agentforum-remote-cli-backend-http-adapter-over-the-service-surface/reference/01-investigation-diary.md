---
Title: Investigation diary
Ticket: AGENTFORUM-003
Status: active
Topics:
    - forum
    - agents
    - remote-backend
    - http-api
    - go
    - glazed
    - cli
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological record for the remote CLI backend phase."
LastUpdated: 2026-09-04T00:10:00-04:00
WhatFor: "Record investigation, decisions, and implementation steps for the remote backend."
WhenToUse: "Read before implementing the remote CLI backend."
---

# Investigation diary

## Goal

Record the investigation and design work for AGENTFORUM-003: the remote
CLI backend over the AGENTFORUM-002 wire contract.

## Step 1: Ticket creation and design guide

Created the ticket and wrote the intern-facing design guide. No code
changed; the deliverable is the implementation contract (R1–R5).

### Prompt Context

**User prompt (verbatim):** "ok, continue. did we create the new tickets? if not, create them, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create AGENTFORUM-003 and AGENTFORUM-004 (they had only been discussed), each with an intern-facing design guide stored in the ticket and uploaded to reMarkable.

**Inferred user intent:** Convert the two follow-up recommendations into real tickets with the same deliverable quality as AGENTFORUM-001/002 before implementation starts.

### What I did
- Added vocabulary topics (`remote-backend`); created ticket AGENTFORUM-003 with design doc + diary.
- Audited the exact CLI→service call surface (grep over `internal/cli/*.go`): 24 distinct methods with call counts, recorded in design §3.3 — the interface extraction is derived from this evidence, not from memory.
- Wrote the design guide: interface extraction (D1: interface over parallel command trees), the `remote.Backend` adapter with `do()` helper pseudocode, endpoint table (§5.3), error mapping (§6.2), long-poll timeout coupling (§6.3), phased plan R1–R5, testing parity strategy, four decision records, risks, file map.

### Why
The remote backend was specified as a sketch (AGENTFORUM-002 §5.6); it needs an implementation-grade contract — in particular the interface-extraction decision and the server's `GetAck` gap.

### What worked
- The CLI call-surface grep turned one real gap immediately: `GetAck` has no HTTP endpoint (design §5.3, task R2).

### What didn't work
- Nothing failed (documentation phase).

### What I learned
- The CLI calls `ResolveAgent` 11 times vs. every other method once — auth resolution dominates the surface, which argues for the interface keeping it cheap (it is: one HTTP round-trip behind the scenes per command that needs an agent).

### What was tricky to build
- N/A (documentation).

### What warrants a second pair of eyes
- The `Backend` interface in §4.2 must match the real signatures exactly at R1 time — re-grep before extracting; the design records call counts as evidence but signatures should be copied from the source, not the doc.

### What should be done in the future
- R1–R5 per the plan.

### Code review instructions
- Start: design doc §4 (interface) and §5.3 (endpoint table).
- Validate: `docmgr doctor --ticket AGENTFORUM-003`.

### Technical details
- Ticket path: ttmp/2026/09/04/AGENTFORUM-003--agentforum-remote-cli-backend-http-adapter-over-the-service-surface.
