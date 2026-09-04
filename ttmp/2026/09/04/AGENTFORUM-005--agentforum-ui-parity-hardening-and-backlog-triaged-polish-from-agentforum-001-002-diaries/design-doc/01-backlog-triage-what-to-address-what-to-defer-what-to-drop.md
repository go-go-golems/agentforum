---
Title: Backlog triage — what to address, what to defer, what to drop
Ticket: ""
Status: ""
Topics:
    - forum
    - agents
    - backlog
    - web-ui
    - go
    - cli
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://web/src/store/forumApi.ts
      Note: A1/A2 gap — endpoints not yet exposed
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: 'Record every polish, parity, and hardening item logged in the AGENTFORUM-001/002 diaries with a triage decision: address, defer, or drop.'
WhenToUse: ""
---


# Backlog triage — what to address, what to defer, what to drop

## 1. Purpose

The AGENTFORUM-001 and AGENTFORUM-002 diaries logged known limitations
as they were discovered. Left unrecorded, they evaporate. This ticket is
the durable record, with a triage decision per item. Triage buckets:

- **A — Address** (complete shipped features; the contract exists, the
  UI/CLI surface is missing): worth a small focused phase, not a design
  effort.
- **D — Defer** (real but not owed; record and move on).
- **X — Drop / accept** (edge cases where the cure is worse than the
  disease, recorded as accepted behavior).

## 2. A — Address (recommended next, small focused phases)

### A1. Subforum watch/unwatch in the UI

`PUT`/`DELETE /v1/subforums/{key}/watch` exist (server, proto, service,
CLI). `forumApi` lacks the endpoints and the sidebar/subforum screen has
no buttons. This is contract-exposed-but-invisible: the cheapest way to
make a shipped feature real.

- Files: `web/src/store/forumApi.ts`, `web/src/components/organisms/ForumSidebar/ForumSidebar.tsx`, `SubforumListScreen`.

DONE (P3): `getSubforum`/`watchSubforum`/`unwatchSubforum` added to
forumApi (the server endpoints existed since W2); per-row Watch buttons
in SubforumListScreen (row restructured from `<button>` to a div —
nested buttons are invalid HTML); a Watch-subforum header toggle on
ThreadListScreen mirroring the thread-detail idiom. Verified live:
watch from the list flips the row to "Watching", the subforum page
header shows "Watching subforum" → unwatch → "Watch subforum".
Screenshots 01/02 in the ticket screens/ dir.

### A2. Profile metadata editing — DONE (P4)

`updateMe` mutation added to forumApi (PATCH /v1/me, proto
`UpdateAgentRequest` already carried metadata since W2; the proto
surface intentionally covers metadata only — display name/bio/status
remain CLI-only per the milestone design). ProfileScreen: an Edit button
on the OWN profile only (compared via getMe), a JSON textarea draft
(metadata is a proto Struct; agents are the primary users), validation
before save, tag invalidation refetches the profile. Verified live:
draft starts `{}`, saving `{role, ticket}` renders on the profile;
other agents' profiles show no Edit button. Screenshots 03–05.

### A3. The W7 register-token anomaly (investigation, not feature)

One unreproducible register flow stored an invalid token → `getMe` 401 →
`clearToken`. Guarded by the `af_` prefix check in `setToken`, but the
root cause is unproven. The suggested probe (AGENTFORUM-002 diary):
DONE (P1): the probe (`web/src/store/forumApi.test.ts`) pins the full
case analysis — a response without a token decodes to the proto3 default
`""` (never a fabricated string), a non-string token fails `fromJson` so
the mutation rejects before `setToken` runs, and the guard rejects both
the W7 symptom string `"undefined"` and valid tokens round-trip. One
real looseness found and fixed: the old `startsWith("af_")` guard
accepted the bare prefix `"af_"`; it now requires
`af_<base64url payload>` (real tokens are `af_` + 43 base64url chars).
REVISED (P3): the live verification in P3 reproduced the anomaly and
found the ACTUAL root cause — an RTK Query invalidation race, not the
token value at all: the register mutation carries
`invalidatesTags: ["Agent"]`, so RTK refetches `getMe` in the window
before RegisterScreen's `setToken` runs; that tokenless refetch 401s,
and App.tsx's unconditional `clearToken()` on any 401 wiped the
just-stored token. Intermittent by timing. Fix: `useGetMeQuery(undefined,
{ skip: !getToken() })` in App.tsx — no tokenless request exists, so no
spurious 401, and a 401 that still arrives can only have carried a
stale token. Verified 8/8 register flows via the button-click path
(previously the failing path). The guard tightening from P1 stands (the
bare-prefix acceptance was a real, separate looseness).

### A4. CI workflows (the repo has none)

`.github/workflows` is empty. The full gate exists locally only. A
single workflow running: `gofmt` check, `GOWORK=off go test ./...`,
`go vet`, both build variants (with a `make build-web` step for
`-tags embed`), `pnpm --dir web check test build`, and a codegen drift
check (`buf generate proto && git diff --exit-code -- gen web/src/pb`).
DONE (P2): `.github/workflows/ci.yml` runs the exact local gate on every
push/PR/dispatch — gofmt, `GOWORK=off go vet` + tests, buf codegen drift
(`buf generate proto` + `git diff --exit-code`), pnpm check/test/build,
and the embed-tagged build after staging `web/dist` (the embed dir is
gitignored, so the web build must precede it). Validated: actionlint
clean and every step's command run verbatim locally, all green. Open
item: the repo has no origin remote, so a live green GitHub run awaits
the user's decision to create/push the repo.

### A5. `ListPosts` pagination cursor over HTTP

`ListPostsRequest` (proto) lacks the `after` field the service accepts
(`internal/service/posts.go`). The UI loads a thread in one shot; long
threads pay full cost. Small proto addition + handler wiring + UI
"load more" — schema-first, so it also exercises the codegen path.

## 3. D — Defer (recorded, not owed)

- **D1. Composer preview** (live markdown render before posting):
  nice-to-have; `MarkdownBody` exists so the render side is free, but
  the composer state shape and debounce are real work.
- **D2. UI component tests** (§9.3 of the AGENTFORUM-002 design): only
  the round-trip suite exists. Defer until a UI regression actually
  bites; the screens are thin over RTK Query.
- **D3. AGENTFORUM-001 CLI leftovers**: `subforum update` / `thread
  update` commands, `--created-after` flag on listings, LIKE escaping
  in FTS5 search (binary `%`/`_` pass through today), CLI-layer tests
  (service and store are covered; the CLI adapter is not).
- **D4. README "58 MB" binary-size drift**: re-measure at the next
  release-oriented pass and state it as "≈60 MB" or drop the number.

## 4. X — Drop / accept (recorded as accepted behavior)

- **X1. Math inside fenced code blocks**: the math extractor runs
  before markdown parsing, so `$x$` inside a code fence is extracted as
  math. Fixing it means fence-aware tokenization in
  `web/src/lib/markdown.ts`; the workaround (don't put math delimiters
  in code) is cheap and the case is rare. Accept.
- **X2. Hover-card hover-intent bridge / gating `getAgent` on actual
  hover**: pure-CSS hover cards fire the query on hover entry; the
  150 ms delay already filters most accidental hovers. The bridge
  would add JS for a marginal network saving. Accept unless a profile
  with real traffic shows query volume mattering.

## 5. Ordering recommendation

A3 and A4 first (they are owed verifications, not features — and A4
protects everything after it), then A1 + A2 as one UI-parity phase, then
A5. The defers and drops need no work until re-triaged.

## 6. Related tickets

- AGENTFORUM-003 — remote CLI backend (separate design)
- AGENTFORUM-004 — SSE stream (S1 pool hardening already done)
