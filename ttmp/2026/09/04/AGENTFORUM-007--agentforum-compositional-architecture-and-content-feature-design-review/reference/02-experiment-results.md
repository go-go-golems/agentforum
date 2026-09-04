---
Title: Reproducible review experiments and baseline validation
Ticket: AGENTFORUM-007
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/01_review_probe_test.go
      Note: Six actual-code counterexamples
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/02_progress_model.py
      Note: Finite checks of proposed progress and ordering laws
ExternalSources: []
Summary: Six production-code counterexamples and finite model checks at commit 90a343c.
LastUpdated: 2026-09-04T15:30:00-04:00
WhatFor: Separate reproduced defects from design inferences.
WhenToUse: Before implementing regression fixes or evaluating the proposed progress contract.
---


# Experiments and validation

The baseline is commit `90a343c`. Production source was not edited. The Go probes create a temporary database, call the actual service/store/HTTP handlers, and remove the temporary database through Go testing cleanup. An HTTP recorder is used; no listening server or live forum is involved. The passing assertions describe existing defects, not desired behavior. Reverse them when writing production regression tests.

## 1. Production-code probes

Run from the repository root:

```bash
go test ./ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts -v -count=1
```

All six probes passed, reproducing:

```text
OBSERVED: ack(100); ack(50) => 50; future acknowledgments also accepted
OBSERVED: ListPosts after po_review_a omits po_review_b with identical timestamp
OBSERVED: participant + watcher receives no other-author posts for scope=watching
OBSERVED: failed idempotency save is ignored; identical retry creates a second post
OBSERVED: second agent using same key replaces first agent's replay record
OBSERVED: body over 1 MiB accepted when first MiB is valid JSON plus whitespace
```

The timestamp probe deliberately supplies identical times through the store API; it establishes a deterministic boundary defect, not an estimate of its frequency with the production clock. The idempotency failure probe installs a temporary SQLite trigger that raises `review injected idempotency failure` on replay-record insertion. Both calls nevertheless return success with distinct post IDs. This demonstrates the ignored-save failure path without claiming to have measured a concurrent race.

The scope probe uses an agent who both created/participates in and watches the thread. Other-agent posts exist. A `watching`-only poll returns no events because the participating label wins before filtering. This is distinct from the documented choice to show only one reason per event.

## 2. Finite mathematical model

```bash
python3 ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/02_progress_model.py
```

```text
PASS: 1536 finite checks of max join laws
PASS: immutable-order unread set survives all 720 display permutations
COUNTEREXAMPLE: loaded {1,2,6}; max=6 falsely acknowledges unseen {3,4,5}
PASS: snapshot traversal emits 1..6 once despite concurrent appends
COUNTEREXAMPLE: precedence-before-filter loses a valid watching match
```

This models the laws used in the design. It is not a proof of the proposed SQL migration, a performance benchmark, a distributed-system implementation, or a browser integration test. The prose review supplies the general max-join argument.

## 3. Existing baseline checks

`go test ./internal/... -count=1` passed: server, service, and store suites; remaining internal packages have no tests. `pnpm --dir web check` passed. `pnpm --dir web test` passed all 16 tests in three files (SSE parser: 6; protobuf roundtrip: 5; API/token handling: 5).

Green baseline tests and reproduced defects coexist because these boundary scenarios are not covered by those existing assertions. No claim is made that AGENTFORUM-006 is implemented or that its future race, migration, and browser gates pass.

## 4. Collection diagnostics

The initial sandboxed source collection failed with `Error loading content: getaddrinfo ENOTFOUND www.sqlite.org`. Retrying the same read-only downloads with network access allowed succeeded for the technical references. This was an environment restriction, not an application failure. The RFC and arXiv collectors subsequently printed CSS/MathJax parsing warnings while still writing usable extracts; source provenance and the diary record these qualifications.

An initial discovery command assumed `pkg/` existed (`rg: pkg: No such file or directory (os error 2)`); this repository uses `internal/`. A later shell brace/glob search produced `zsh:1: no matches found: ttmp/2026/09/03/AGENTFORUM-004*/reference/*.md`; subsequent discovery used `rg` across the ticket tree. Neither changed source files.
