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

## Step 3: P2 (A4) — CI workflow mirroring the local gate

Wrote `.github/workflows/ci.yml` and validated it as far as a local-only
repo allows: actionlint clean and every step's command executed verbatim
on this machine, green. The live GitHub run is pending the user's
decision to create an origin remote (the repo has none).

### Prompt Context

**User prompt (verbatim):** (see Step 2 — same task prompt)

**Assistant interpretation:** Implement A4: a CI workflow running the full gate, verified green.

**Inferred user intent:** The gate exists only on this machine; CI makes every future change (including all remaining A items) protected.

### What I did
- Wrote `.github/workflows/ci.yml` (push to main, PRs, manual dispatch): checkout → setup-go (`go-version-file: go.mod`) → pnpm 10 + node 24 (cache keyed on `web/pnpm-lock.yaml`) → gofmt check → `GOWORK=off go vet ./...` → `GOWORK=off go test ./... -count=1` → `go build ./...` → buf setup + `buf generate proto` + `git diff --exit-code -- gen web/src/pb` → pnpm install/check/test/build → stage `web/dist` into `internal/server/embed/public` → `GOWORK=off go build -tags embed ./...`.
- Validated: `actionlint .github/workflows/ci.yml` clean; then ran each step's shell command locally, verbatim, all green (gofmt ok, vet ok, 3 package test runs ok, build ok, no codegen drift, pnpm install/check/9 tests/build ok, embed build ok).

### Why
A4 was the triage doc's top ordering recommendation: the only item that protects all future work, subsuming the guide-drift check (the agent guide is embedded via `go:generate`; the buf drift step covers generated code, and the embed build compiles the embedded help).

### What worked
- The step order falls out of one constraint: `internal/server/embed/public` is gitignored, so the embed-tagged build must come after the web build stages it.

### What didn't work
- `which actionlint` failed (not installed); fixed with `go install github.com/rhysd/actionlint/cmd/actionlint@latest`. First local `pnpm install` in the verbatim sequence printed `Done in 421ms using pnpm v10.13.1` — harmless version skew between the local pnpm (10.13.1 via install, 10.15.1 on PATH elsewhere); the workflow pins `version: 10`.

### What I learned
- The whole gate is plain shell — every CI step could be validated locally by running its command verbatim; only the Actions runner environment (setup actions, cache) is untestable without a remote.

### What was tricky to build
- Caching: `actions/setup-node` pnpm cache requires `cache-dependency-path: web/pnpm-lock.yaml` since the lockfile is not at the repo root.
- gofmt step in CI differs slightly from the local gate (no `grep -v embed/public` needed — the dir is gitignored and absent from a fresh checkout).

### What warrants a second pair of eyes
- The repo has **no origin remote**, so the workflow has never executed on GitHub runners. Exact evidence: `git config --get remote.origin.url` → empty; `git log --oneline -3` shows local-only history. Creating the repo under the go-go-golems org (or elsewhere) is a user decision; the run should be watched after that (`gh run watch`).

### What should be done in the future
- When the user decides on the remote: push, then `gh run watch` the first CI run and record the result here.

### Code review instructions
- Start: `.github/workflows/ci.yml`.
- Validate: `actionlint .github/workflows/ci.yml` and the step commands (or push and watch the run).

### Technical details
- Commit: see changelog. Local pnpm note: PATH `pnpm --version` = 10.15.1 but the frozen install reported v10.13.1 — verify no lockfile drift resulted (`git diff --exit-code -- web/pnpm-lock.yaml` was clean after install).
