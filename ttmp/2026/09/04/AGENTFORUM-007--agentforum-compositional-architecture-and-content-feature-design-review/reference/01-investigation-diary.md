---
Title: Investigation diary
Ticket: AGENTFORUM-007
Status: active
Topics:
    - backend
    - frontend
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/01_review_probe_test.go
      Note: Actual-code probes
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/02_progress_model.py
      Note: Finite mathematical model
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/03_collect_sources.sh
      Note: Technical reference collection
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-007--agentforum-compositional-architecture-and-content-feature-design-review/scripts/04_collect_fundamentals.sh
      Note: Foundational paper collection
ExternalSources: []
Summary: Investigation of the existing forum, six reproducible counterexamples, fundamentals research, design synthesis, and reMarkable delivery.
LastUpdated: 2026-09-04T15:25:23.879717031-04:00
WhatFor: Record evidence, decisions, validation, and handoff for this architecture review.
WhenToUse: Before resuming the review or implementing its recommendations.
---


# Investigation diary

## Goal

Review the existing system and AGENTFORUM-006 from first principles, preserve evidence in a new ticket, and deliver an intern-readable design review to reMarkable. No production implementation was requested in this turn.

## Step 1: Establish the actual system and reproduce boundary defects

Created AGENTFORUM-007 and traced tickets 001–006 against the current source. The important distinction was between historical intentions and implemented behavior: the remote CLI remains planned, while HTTP, SSE, and the UI parity work are present. The source also exposed gaps which become correctness prerequisites for durable read tracking.

Six small Go probes call the actual application against temporary databases. All reproduced their target behavior while the existing baseline suites passed. The review therefore separates test coverage gaps from general architectural preferences.

### Prompt Context

**User prompt (verbatim):**

> Analyze the docmgr tickets and past diaries, and the upcoming AGENTFORUM-006, and analyze it from fundamentals (maths, computer science), to create a more composable and streamlined / coherent design. Create a new docmgr ticket for your code review / architecture / design review. Create  a detailed analysis / design / implementation review that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
>
> You can run experiments and write scripts and all that in the scripts/ folder of the ticket, you can look for resources to flesh out the review and store it in sources/ (use defuddle to download as markdwon).

**Assistant interpretation:** Create a new review ticket, inspect history and code, run focused experiments, write a detailed architecture/design/implementation review, and upload the result.

**Inferred user intent:** Improve the conceptual foundations before adding interacting content features, with enough evidence and explanation for a new engineer to implement them coherently.

### What I did

- Read the ticket-research, docmgr, diary, reMarkable-upload, and Defuddle skills and the research skill's required writing/checklist references.
- Confirmed a clean starting worktree and baseline commit `90a343c`.
- Created the ticket, primary design document, and diary with docmgr.
- Read AGENTFORUM-006's full draft and examined historical diary sections, source, wire schemas, tests, and build configuration.
- Added `scripts/01_review_probe_test.go` and `02_progress_model.py`.
- Ran `go test ./internal/... -count=1`, `pnpm --dir web check`, and `pnpm --dir web test`.

### Why

The draft combines mutable display order with a durable read frontier. That cannot be made safe by a small sorting change. Concrete counterexamples make the distinction visible before implementation grows around it.

### What worked

- Six actual-code probes reproduced acknowledgment regression, tied-timestamp omission, reason-precedence filtering, ignored replay-save failure, global replay-key replacement, and request-body suffix truncation.
- Go internal tests passed; web type checking and all 16 web tests passed.
- The finite model checked max-merge laws, all 720 display permutations for six posts, a noncontiguous pinned-post example, and snapshot traversal with concurrent appends.

### What didn't work

- Initial file discovery assumed `pkg/`: `rg: pkg: No such file or directory (os error 2)`. Actual packages are in `internal/`.
- A brace/glob command targeted nonexistent date/ticket combinations: `zsh:1: no matches found: ttmp/2026/09/03/AGENTFORUM-004*/reference/*.md`. Switched to recursive rg discovery.
- Large batched reads were truncated by tool output limits; relevant source and diary sections were read in smaller focused calls.

### What I learned

- `AckEvents` is not monotonic despite the draft's description.
- Replay records are outside content transactions, and their actual primary key is global.
- Protobuf event types are enums; adding SQL event strings alone would emit UNSPECIFIED on the wire.
- The browser's saved stream cursor and volatile inbox rows have different durability, separate from content progress.

### What was tricky to build

The probes deliberately assert current defects. Passing them is evidence that the observations are reproducible, not a claim the application is correct. The experiment document states that future regression tests must reverse these expectations. Trigger-based replay-save failure avoids a flaky concurrency experiment while exposing a real failure boundary.

### What warrants a second pair of eyes

- Catch-up prefix coverage, especially when pins are loaded out of order.
- Atomic replay migration and request digest semantics.
- Static browser findings, which were not reproduced in a browser during this review.

### What should be done in the future

Implement the review's Phase 0 regressions and foundational ordering/replay changes before content features. This turn does not implement them.

### Code review instructions

- Begin with primary review sections 2–4 and `reference/02-experiment-results.md`.
- Run the ticket-local Go probe command from the experiment record.
- Inspect `internal/store/posts.go:76`, `events.go:67`, `idempotency.go:40`, and the service create methods.

### Technical details

The experiments use `testing.T.TempDir` and HTTP recorders, with no listening server, no live database, no additional Go module, and no production edits. No commit was created.

## Step 2: Research fundamentals and synthesize the design

After establishing the review findings, collected primary references on relational data, logical clocks, join-based state convergence, application acknowledgment, SQLite isolation, protobuf encoding, and HTTP semantics. The resulting review uses those ideas to explain a small set of concrete operations instead of adding a general distributed-systems framework.

The main document covers current architecture, ranked findings, a mathematical model, scope and read semantics, typed service views, migrations, catch-up page/ack contracts, attachments, pinning, browser state, phases, tests, tradeoffs, and proposed decision records. The annotated guide connects each foundational resource to an observed problem.

### Prompt Context

**User prompt (verbatim):** "also look for resources on the fundamentals, once you have properly established your review."

**Assistant interpretation:** Ground the research in the code review, then find and explain foundational resources relevant to its conclusions.

**Inferred user intent:** Obtain a technically rigorous design informed by mathematics and computer science rather than a collection of refactoring preferences.

The user later asked, **User prompt (verbatim):** "Is the pdftotext taking so long?"

**Assistant interpretation:** Provide a concrete status check on the apparent delay while preserving the review task.

**Inferred user intent:** Ensure the task is progressing and is not stuck on a slow extraction tool.

### What I did

- Archived HTML with Defuddle via scripts 03 and 04; downloaded author-hosted Lamport and Saltzer/Reed/Clark PDFs and extracted searchable text.
- Confirmed both pdftotext conversions had completed and no collector remained running when asked.
- Read the relevant extracted passages and wrote `reference/03-fundamentals-reading-guide.md` plus `sources/README.md`.
- Wrote the main review, about 10,200 words, with ASCII diagrams that also render in a PDF, API/schema sketches, SQL/pseudocode, and eight proposed decision records.
- Refined page acknowledgment to derive a per-thread lower frontier from the incoming page cursor, preventing a later page from silently acknowledging skipped earlier content.

### Why

The same handful of invariants connect the features: immutable order supports pagination and read progress; scope is a union of independent relationships; max merge tolerates retries; transaction boundaries protect replay; explicit acknowledgment prevents output failure from hiding work.

### What worked

- All technical and foundational references were saved locally.
- Defuddle's CRDT extract contains the synchronization-model conditions despite parser noise.
- The design preserves the existing relational core and identifies narrow places to share code.

### What didn't work

- The first sandboxed fetch failed: `Error loading content: getaddrinfo ENOTFOUND www.sqlite.org`. Repeated with network access allowed.
- The RFC extraction reported `Error: Could not parse CSS stylesheet`, but wrote the markdown and exited successfully.
- The arXiv extraction reported `Defuddle Error processing document: DOMException [SyntaxError]` for a MathJax selector, but retained the article body. Both warnings are documented in source provenance.
- No attempt was made to patch Defuddle or the application during this review.

### What I learned

- The useful CRDT connection is the max-join law, not a need for replication infrastructure.
- Logical-clock literature explains why physical timestamp precision does not establish the required content order.
- A stable multi-page catch-up needs frozen selection inputs or an explicitly weaker live-query contract.
- A successful output processor AddRow may precede actual stdout flush, so CLI mark-read needs a verified output boundary.

### What was tricky to build

The catch-up plan must be bounded and resumable without holding a transaction across network requests. The proposed plan stores selection metadata, then uses short page transactions and separate progress receipts. Its cost and alternatives are explicit. Mutable profile names and present-day pin context are not mislabeled as a historical database snapshot.

### What warrants a second pair of eyes

- Product defaults: visited scope, initial subforum baseline, pin permissions, and limits.
- Receipt gap validation and plan expiry behavior.
- Whether the simpler live-query alternative is preferable to stable plan storage for the expected workload.

### What should be done in the future

Turn proposed decisions into implementation contracts when AGENTFORUM-006 starts. Measure query plans, memory, and contention before increasing initial limits.

### Code review instructions

- Read sections 5–8 of the primary review alongside the fundamentals guide.
- Check the source archive qualifications before treating extracts as exact mathematical reproductions.
- Review Phase 1 migration policy before applying it to user data.

### Technical details

The source archive contains seven markdown extracts, two original PDFs, two extracted text files, and a provenance README. Third-party full texts are not included in the reMarkable bundle; the bundle contains original review prose and references.

## Step 3: Validate ticket and deliver the review bundle

Completed file relations, source provenance, tasks, and changelog. The ticket doctor passed cleanly. The reMarkable dry run selected the intended four documents, and the real upload succeeded.

This delivery entry is local bookkeeping added after upload. The uploaded diary contains investigation Steps 1–2; the uploaded primary review, reading guide, and experiment findings are complete. No production implementation or commit was made.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Validate the review artifacts and complete the requested reMarkable delivery.

**Inferred user intent:** A usable document on the device plus a traceable local ticket, not an unfinished upload plan.

### What I did

- Related core source files to the design, experiment scripts to their evidence record, and collection scripts/provenance to the diary and reading guide.
- Added architecture/frontend vocabulary entries used by this ticket.
- Ran `docmgr doctor --ticket AGENTFORUM-007 --stale-after 30`: all checks passed.
- Ran `git diff --check`: clean.
- Ran `remarquee upload bundle --dry-run` with the four deliverables, the chosen name/destination, `--toc-depth 2`, and `--non-interactive`.
- Ran the real bundle upload with the same files and options.

### Why

The research skill requires organized evidence, a clean doctor result, and a dry run. The specialized upload skill treats the successful upload response as confirmation, so no redundant account/status/listing requests were made.

### What worked

The uploader returned:

~~~text
OK: uploaded AGENTFORUM-007 Architecture and Content Design Review.pdf -> /ai/2026/09/04/AGENTFORUM-007
~~~

### What didn't work

No validation or upload failures.

### What I learned

The ASCII diagrams and ordinary fenced pseudocode render through the existing markdown-to-PDF pipeline without needing a separate Mermaid renderer.

### What was tricky to build

Keeping the uploaded document self-contained while retaining longer third-party sources locally. The bundle includes the annotated guide and links, not the full source archive.

### What warrants a second pair of eyes

The proposed scope/baseline defaults and materialized catch-up plan remain design choices for implementation review. The verified existing defects are independently reproducible.

### What should be done in the future

Use the main review's phased plan when starting AGENTFORUM-006; do not treat the review ticket's checked tasks as feature implementation.

### Code review instructions

Start with the ticket index and primary review. Follow the experiment links for evidence and the reading guide for foundational context.

### Technical details

Destination: `/ai/2026/09/04/AGENTFORUM-007`.

Bundle name: `AGENTFORUM-007 Architecture and Content Design Review`.

Local changes: new review ticket plus two vocabulary entries. Production source unchanged.
