---
Title: Investigation diary
Ticket: AGENTFORUM-002
Status: active
Topics:
    - forum
    - agents
    - protobuf
    - web-ui
    - http-api
    - go
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/publish-vault/web/package.json
      Note: Analyzed for the copy map dependency decisions (§6.6)
    - Path: repo://ttmp/2026/09/03/AGENTFORUM-002--agentforum-web-ui-and-protobuf-payloads-http-server-publish-vault-look-reuse-embedded-frontend/design-doc/01-agentforum-web-ui-and-protobuf-payloads-analysis-design-and-implementation-guide.md
      Note: Primary deliverable of Step 1
ExternalSources: []
Summary: Chronological investigation and design record for the agentforum web UI + protobuf payloads phase.
LastUpdated: 2026-09-03T18:30:00-04:00
WhatFor: Record what was investigated, decided, and written for AGENTFORUM-002, so implementation can resume without re-deriving context.
WhenToUse: Read before starting any implementation phase of the web UI / HTTP server / protobuf work.
---



# Investigation diary

## Goal

Record the investigation and design work for AGENTFORUM-002: the protobuf
payload contract, the HTTP server, and the web UI copied from
publish-vault source.

## Step 1: Ticket creation, publish-vault source analysis, and the design doc

This step created the AGENTFORUM-002 ticket, analyzed the publish-vault web
source tree to build a concrete copy map, and wrote the intern-facing
analysis/design/implementation guide. No agentforum code was changed in this
step; the deliverable is the contract the implementation phases (W1–W8) will
follow.

### Prompt Context

**User prompt (verbatim):** "Use protobuf for the payloads, and also add and flesh out a ticket where you can literally copy source out of ~/code/wesen/go-go-golems/publish-vault/web to reuse the look and widgets. We'll unify later on, but this gives us a starting point. Create the web UI ticket, flesh out the current design for having protobuf payloads to use for the UI as well, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Open a new docmgr ticket for the web UI phase; design protobuf payloads shared by the Go server and the TS UI; specify exactly which publish-vault/web source files to copy to reuse the retro look and widget IR system; write a detailed intern-level design/implementation guide; store it in the ticket and upload the bundle to reMarkable.

**Inferred user intent:** The user wants the next milestone (HTTP server + web UI) fully specified before implementation starts, with the protobuf contract as the single source of truth for payloads, and a pragmatic UI starting point obtained by copying proven source rather than designing a new system. "We'll unify later" explicitly de-prioritizes shared-package extraction.

### What I did
- Read the `protobuf-go-ts-schema-exchange` skill (SKILL.md + references/templates.md) for schema/buf/protojson/RTK conventions.
- Explored `~/code/wesen/go-go-golems/publish-vault/web`: package.json (React 19, Vite, Tailwind v4, RTK, radix, resizable panels), `src/` tree (162 files; components 5,383 LOC, widgets 1,410 LOC, styles 1,054 LOC, store 281 LOC), styles layering (tokens/base/chrome/prose/bridge), widget IR system (ir/actions, cells, core, props + registry + WidgetRenderer), RTK Query pattern (vaultApi.ts), SSR-safe store factory, and the Go embed pattern (`pkg/web/embed.go`, serve command).
- Added vocabulary topics: `protobuf`, `web-ui`, `http-api`.
- Created ticket AGENTFORUM-002 with design doc + this diary; added 8 phase tasks (W1–W8).
- Wrote the full design doc (12 sections: exec summary, scope, current-system recap, proto contract, HTTP server, web UI copy map, screens/data flow, phased plan, testing, 8 decision records, risks, file map).

### Why
The AGENTFORUM-001 design doc §10 fixed the endpoint list but not JSON shapes, error envelopes, or routing. The web UI had no design at all. Both gaps block the next milestone, and the user supplied two direction decisions: protobuf payloads (for API and UI), and copy-don't-extract for the UI source.

### What worked
- The publish-vault analysis produced a file-by-file copy map with verbatim/adapt/do-not-copy classifications — the "literally copy source" requirement is directly executable from §6.1.
- The protobuf skill's RTK boundary rules (fromJson in transformResponse, no hand-written mirrors, bigint kept in cache) resolved exactly how "payloads for the UI as well" should work.
- AGENTFORUM-001's service surface (§3.3 table) mapped 1:1 onto request/response message pairs, confirming the server really is an adapter.

### What didn't work
- Nothing failed in this step (documentation only). One investigation dead end worth noting: publish-vault has no `.proto` files anywhere — its RTK types are hand-written (`src/types/index.ts`). So the proto schema is new work, not a copy; the design doc replaces that hand-written-types pattern with generated ones per the schema-exchange skill.

### What I learned
- publish-vault's widget IR tree is itself a port (from rag-evaluation-system v0.1.7, per its header comments) — that lineage explains `bridge.css`'s `--rag-*` → `--pv-*` mapping and makes the "copy verbatim now, unify later" strategy consistent with how this code already evolves.
- The `--pv-*` token namespace is deliberately canonical with a Tailwind `@theme inline` bridge; keeping it intact during the copy is what makes later extraction cheap.

### What was tricky to build
- N/A for code (no code changed). The tricky part of the *design*: reconciling protojson's type rules with the existing system. `int64` serializes as JSON strings (bigint in TS) and `google.protobuf.Struct` becomes a plain JsonObject — both clash with naive expectations. Resolved by writing §4.6 ("the two type traps") as mandatory reading plus round-trip tests in both languages sharing fixtures from `testdata/protojson/`.

### What warrants a second pair of eyes
- The copy map's adapt-vs-verbatim classifications (§6.1) — each "adapt" mark is a place where the fork diverges from publish-vault; wrong classifications create silent drift.
- Decision D2 (string RFC 3339 timestamps instead of google.protobuf.Timestamp) — defensible but a contract-level choice.
- Decision D8 (token in localStorage) — security posture call that deserves explicit sign-off.

### What should be done in the future
- W1–W8 implementation per §8 (tasks added to tasks.md).
- Re-verify `SetMaxOpenConns(1)` under N concurrent long-pollers before W7 ships (risk R4).
- Decide the R2 open question (protojson DiscardUnknown on decode) during W1.

### Code review instructions
- Start: `design-doc/01-…-implementation-guide.md` §2 (scope), §4 (proto), §6 (copy map).
- Validate: `docmgr doctor --ticket AGENTFORUM-002`; the guide's §12 file map should let a reader locate every referenced file in both repos.

### Technical details
- publish-vault/web stats at analysis time: 162 src files; components 5,383 LOC / widgets 1,410 / styles 1,054 / store 281 / lib 952 / search 829.
- Ticket path: `ttmp/2026/09/03/AGENTFORUM-002--agentforum-web-ui-and-protobuf-payloads-http-server-publish-vault-look-reuse-embedded-frontend`.
