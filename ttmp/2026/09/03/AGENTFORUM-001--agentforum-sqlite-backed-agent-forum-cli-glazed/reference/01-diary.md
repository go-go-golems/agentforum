---
Title: Diary
Ticket: AGENTFORUM-001
Status: active
Topics:
    - go
    - cli
    - sqlite
    - forum
    - agents
    - glazed
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://README.md
      Note: Quickstart + layout + status (P7)
    - Path: repo://internal/cli/events.go
      Note: events poll/follow/ack commands (commit 5744e9a)
    - Path: repo://internal/cli/profile.go
      Note: profile register/show/update commands (commit dbf44e4)
    - Path: repo://internal/cli/root.go
      Note: Glazed root + AppName env loading + openService (commit cbdc6a6)
    - Path: repo://internal/cli/search.go
      Note: cross-entity search command (commit 37f0c85)
    - Path: repo://internal/cli/subforum.go
      Note: subforum list/create/show/watch/unwatch (commit a01d81c)
    - Path: repo://internal/cli/thread.go
      Note: thread create/list/show/watch/unwatch (commit b8f9ea2)
    - Path: repo://internal/service/agents.go
      Note: Register/ResolveAgent/UpdateMe business rules (commit dbf44e4)
    - Path: repo://internal/service/events.go
      Note: PollEvents long-poll + reason computation (commit 5744e9a)
    - Path: repo://internal/service/subforums.go
      Note: subforum CRUD + watch rules (commit a01d81c)
    - Path: repo://internal/service/threads.go
      Note: idempotency replay (commit 37f0c85)
    - Path: repo://internal/store/agents.go
      Note: agent CRUD + token-hash lookup (commit dbf44e4)
    - Path: repo://internal/store/migrations.go
      Note: embedded migration runner (commit cbdc6a6)
    - Path: repo://internal/store/posts.go
      Note: atomic CreatePost + ListPosts (commit b8f9ea2)
    - Path: repo://internal/store/search.go
      Note: metadata_terms query path (commit 37f0c85)
    - Path: repo://internal/store/store.go
      Note: SQLite open with WAL/busy_timeout/FK (commit cbdc6a6)
    - Path: repo://internal/store/threads.go
      Note: atomic CreateThreadWithPost (commit b8f9ea2)
ExternalSources: []
Summary: 'Implementation diary for agentforum: chronological record of each phase, what changed, what failed, and how to validate.'
LastUpdated: 2026-09-03T16:40:29.043612152-04:00
WhatFor: Record the implementation journey so review and continuation are straightforward.
WhenToUse: Read before resuming work on AGENTFORUM-001.
---








# Diary

## Goal

Build `agentforum`, a SQLite-backed forum for AI agents, CLI-first using the Glazed
command framework: profiles with token auth, subforums, threads/posts, a unified
cursor-based event inbox with long-polling, flexible JSON metadata with a terms
index, and idempotency. Capture the journey phase by phase with failures and
verification.

## Step 1: Ticket, design guide, and phased plan

This step scoped the whole effort and produced the artifacts the rest of the work
stands on: a docmgr ticket, an intern-facing design/implementation guide, a phased
task list, and a printed plan slip. The impact is that implementation can now
proceed phase by phase with a stable contract to check against.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create a go forum for agents, using sqlite. First ,CLI only, using the galzed command framework to get env variables + CLI flags. Create a detailed analysis / design / implementation guide that is for a new intern … Store in the ticket and the nupload to remarkable. Then implement, commit at appropriate intervals and keep a detailed diary as you work … Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done." (followed by the full agentforum spec: env vars, CLI shape, HTTP API, data model, subforums, metadata, search, events.)

**Assistant interpretation:** Set up a docmgr ticket for a Go+SQLite agent forum, write a thorough intern-facing design/implementation guide and upload it to reMarkable, define phases, print a brutalist plan slip, then implement phase by phase with pre/post slips and a diary.

**Inferred user intent:** A repeatable, documented, slip-tracked build of `agentforum` that a new engineer can pick up, with physical paper checkpoints at each phase boundary.

### What I did
- Inspected the repo (greenfield, only docmgr skeleton + "Initial commit").
- Read the `glazed-command-authoring`, `docmgr`, `diary`, `brutalist-work-slip`, and `ticket-research-docmgr-remarkable` skills and their references.
- Studied the real Glazed source at `/home/manuel/code/wesen/go-go-golems/glazed` and the `refactorio` consumer to confirm: `cli.BuildCobraCommandFromCommand`, `WithParserConfig(CobraParserConfig{AppName})` env loading (prefix `AGENTFORUM_`), `schema.NewSection`, `cmds.WithSections/WithFlags/WithParents`, `types.NewRow`+`MRP`.
- Confirmed Glazed v1.4.3 auto-fetches the `go1.26.8` toolchain under `GOTOOLCHAIN=auto`.
- Created ticket `AGENTFORUM-001`; added vocabulary topics `go,cli,sqlite,forum,agents,glazed`.
- Added the design doc `design-doc/01-agentforum-design-and-implementation-guide.md` and this diary.
- Added 7 phase tasks (P1–P7) to `tasks.md`.

### Why
- A written contract prevents drift across phases and makes the brutalist slips meaningful (each slip maps to a real, defined phase).
- Confirming the Glazed env mechanism before coding avoids the `AGENT_NAME`-vs-`AGENTFORUM_` trap documented in the design (§4.3, Decision Records).

### What worked
- Glazed env loading is exactly `UPPER(REPLACE(field,"-","_"))` with the app prefix; a no-prefix `connection` section yields `AGENTFORUM_DB/URL/TOKEN` directly.
- The toolchain auto-download means `go 1.26.1` (Glazed's floor) is not a blocker.

### What didn't work
- (Nothing yet — pure planning step.)

### What I learned
- `cli.BuildCobraCommand` is a thin alias for `BuildCobraCommandFromCommand`; `AddCommandsToRootCommand` builds all commands before mounting (collision-safe). The tutorial's `WithSectionsList` is stale; v1.4.3 uses `cmds.WithSections`.
- `modernc.org/sqlite` (pure Go) keeps `go build` C-free; JSON1 is built in.

### What was tricky to build
- Reconciling the brief's exact env var `AGENT_NAME` (no prefix, display hint) with Glazed's prefix-based env loading. Resolved by letting Glazed own `AGENTFORUM_*` and adding one documented `os.Getenv("AGENT_NAME")` fallback in `profile register` (design §4.3).

### What warrants a second pair of eyes
- The layering rule (§3.1): keep `service` free of `cobra`/`glazed`/`os` so the future HTTP server is a pure adapter.
- The cursor invariant (§8.2): `next_cursor = max(seq) or cursor` — getting this wrong causes replays or skips.

### What should be done in the future
- Extract a `Backend` interface (local/remote) when the HTTP server phase starts, so the CLI gains `--backend remote` without command changes.
- Add SSE at `/v1/events/stream` reusing the exact event format.

### Code review instructions
- Read the design doc end to end (it is the contract for every phase).
- Verify the planned file layout in §3 matches what P1 creates.
- Check the 7 decision records in §11 for any you'd re-litigate before code is written.

### Technical details
- Ticket path: `ttmp/2026/09/03/AGENTFORUM-001--agentforum-sqlite-backed-agent-forum-cli-glazed`
- Tasks: `docmgr task list --ticket AGENTFORUM-001`
- Slip script: `~/.pi/agent/skills/brutalist-work-slip/scripts/work_slip.py plan …`
- Glazed version: v1.4.3 (toolchain go1.26.8); consumer reference: refactorio.

## Step 2: P1 scaffold — module, glazed root, store, migrations

This phase stood up the runnable skeleton: a Go module, a Glazed root command that
loads `AGENTFORUM_*` env vars, a reusable connection section, a pure-Go SQLite
store with an embedded migration runner, and a `db init` command that proves the
whole stack works. The impact is that every later phase starts from a binary
that opens/migrates a database and prints structured output.

**Commit (code):** cbdc6a6 — "feat(agentforum): P1 scaffold …"

### What I did
- `go mod init github.com/go-go-golems/agentforum`; pulled glazed v1.4.3,
  modernc.org/sqlite v1.58.0, oklog/ulid/v2 v2.1.2 (toolchain auto-fetched go1.26.8).
- `internal/id`: prefixed ULID IDs (`ag_/th_/po_`) + `af_` token + `HashToken` (sha256).
- `internal/models`: Agent/Subforum/Thread/Post/Event/Participant/MetadataTerm + event reasons.
- `internal/config`: `Connection` struct + `ConnectionSection()` (no-prefix section →
  `AGENTFORUM_DB/URL/TOKEN/BACKEND`) + `DefaultDBPath/ResolveDBPath` (XDG-aware).
- `internal/store`: `Open` (WAL + busy_timeout + FK on, single conn), embedded
  `migrations/*.sql` runner tracking `schema_migrations`, `0001_init.sql` with all 12 tables.
- `internal/service`: thin `Service` wrapping the store (methods added in later phases).
- `internal/cli`: root wired with `WithParserConfig(AppName:"agentforum")` via
  `AddCommandsToRootCommand`; `db init` command exercising the stack.
- `cmd/agentforum/main.go`, `Makefile`, `.gitignore`, `store_test.go`.

### Why
- One connection section + `AppName` env prefix is the cleanest way to satisfy the
  brief's "glazed for env vars + CLI flags" without hand-rolled `os.Getenv` in the hot path.
- `modernc.org/sqlite` keeps `go build` CGO-free; WAL + single connection
  serializes writes simply for the CLI-only milestone.

### What worked
- `agentforum --help` prints the tree; `db init` creates the DB with all 12 tables;
  `AGENTFORUM_DB=/tmp/af-test.db agentforum db init --format json` works; re-running
  is idempotent (exactly 1 row in `schema_migrations`).
- `go test ./...`, `go vet ./...`, `git diff --check` all clean.

### What didn't work
- (No failures.)

### What I learned
- `cmds.WithSections` (not the tutorial's stale `WithSectionsList`) is the v1.4.3 API.
- The connection section has no prefix on purpose: with `AppName` set, a prefixed
  section would double the env namespace (`AGENTFORUM_AGENTFORUM_DB`).

### What was tricky to build
- Keeping `service` free of `cobra`/`glazed`/`os` while the CLI still owns "open the
  DB + resolve defaults" — resolved by putting `openService` in `internal/cli` and
  passing a plain `config.Connection` into the service.

### What warrants a second pair of eyes
- The single-connection pool (`SetMaxOpenConns(1)`) is correct for write
  serialization but means long-poll reads in P5 share the writer's lock; revisit if
  P5 polls block writes. WAL + busy_timeout should make reads non-blocking, but verify.
- `0001_init.sql` is `IF NOT EXISTS`-guarded *and* tracked in `schema_migrations`, so a
  future migration must not also use `IF NOT EXISTS` blindly or it can mask drift.

### What should be done in the future
- When adding a second migration, ensure it is not idempotent-by-accident so the
  runner actually detects partial applies.
- Consider a `db status`/`db migrate --list` command for diagnostics.

### Code review instructions
- Start in `internal/cli/root.go` (`openService`, `parserOpts`) and `internal/store/store.go` (`buildDSN`).
- Validate: `make build && AGENTFORUM_DB=/tmp/x.db ./agentforum db init --format json` then `sqlite3 /tmp/x.db '.tables'`.
- Run `make test`.

### Technical details
- DSN: `file:<path>?_pragma=journal_mode=WAL&_pragma=busy_timeout=5000&_pragma=foreign_keys=on`.
- Env mapping: `AGENTFORUM_<UPPER(FIELD_WITH_UNDERSCORES)>`.
- Tables: agents, subforums, threads, posts, participants, watches, subforum_watches, events, event_acks, metadata_terms, idempotency_keys, schema_migrations.

## Step 3: P2 profiles & token auth

This phase gave agents identities. Registration mints an `af_…` token, stores only
its SHA-256 hash, and returns the plaintext once. `profile show`/`update`
authenticate by token; a duplicate name returns a conflict; `AGENT_NAME` is honoured
as a display-name hint only. The impact is that every later command can call
`svc.ResolveAgent(token)` to get the acting agent.

**Commit (code):** dbf44e4 — "feat(agentforum): P2 profiles & token auth …"

### What I did
- `internal/store/agents.go`: Create/Get (by name/id/token-hash)/Update + `scanAgent` + `ErrNoRows`; `internal/store/json.go` + `parseTime` for JSON/time columns.
- `internal/service/errors.go`: `ErrUnauthenticated/ErrNotFound/ErrConflict/ErrInvalidInput`.
- `internal/service/metadata.go`: validateMetadata (size/depth/reserved-`_`/key shape).
- `internal/service/agents.go`: `Register` (409 on dup name via pre-check + UNIQUE fallback), `GetMe`, `UpdateMe`, `GetAgentByName`, `ResolveAgent` (401 on bad/missing).
- `internal/cli/profile.go`: register/show/update; `agentRow` helper; `AGENT_NAME` fallback for `--name`.
- `internal/cli/helpers.go`: `buildMetadata` (file + `--meta k=v`), `readBodyFile`, `metadataJSON`.
- Root: error-collecting `add()` builder; `SilenceErrors+SilenceUsage` for clean one-line errors.
- `internal/service/agents_test.go`: register→auth→conflict→unauth→invalid→reserved-key→update.

### Why
- Token-as-hash means a DB leak grants nothing; `AGENT_NAME` stays a casual hint, not auth (Decision Records).
- One `ResolveAgent` entry point keeps every authenticated command uniform and the future HTTP server a thin adapter.

### What worked
- Full roundtrip green: register→show→update; duplicate→`conflict` (exit 1); `AGENT_NAME` fallback→name=reviewer; bad token→`unauthenticated` (exit 1).
- After `SilenceErrors/SilenceUsage`, errors print as one clean line (no usage dump).
- `go test ./...` + `go vet` clean.

### What didn't work
- `types.Row` is an `*orderedmap.OrderedMap`, not a slice, so my first `append(row, …)` for the optional token field failed to compile; fixed by building a `[]types.MapRowPair` and `types.NewRow(pairs...)`.
- A path typo (`wensen-2026…` vs `wesen/2026…`) created a bogus directory tree; the `write` tool auto-makes parents. Caught immediately, `mv`'d the file, `rm -rf`'d the bogus tree.

### What I learned
- `TypeStringList` binds as a cobra `StringSlice` flag → `--meta k=v --meta k2=v2` collects cleanly into `[]string`.
- `models.Agent.TokenHash` has `json:"-"`, so public agent output never leaks the hash even though the struct carries it.

### What was tricky to build
- Mapping a SQLite UNIQUE violation to `ErrConflict` portably: `modernc.org/sqlite` errors contain `"UNIQUE constraint"`, so `isUniqueViolation` string-matches. This is fragile if the driver changes wording; flagged for a future typed-error check.
- `UpdateMe` "non-empty updates, empty unchanged" semantics: simple and correct for the milestone, but cannot clear a field. Documented in help.

### What warrants a second pair of eyes
- `isUniqueViolation` string matching — confirm it covers both `agents.name` and `agents.token_hash` UNIQUE failures (token collisions are astronomically unlikely but should still map to conflict, not a raw error).
- `ResolveAgent` trims whitespace from tokens; a token with leading/trailing spaces in env would silently work — acceptable, but note it.

### What should be done in the future
- Replace `isUniqueViolation` string matching with a typed sqlite error code check if the driver exposes one.
- Add `profile rotate-token` and field-clearing semantics if agents need them.

### Code review instructions
- Start in `internal/service/agents.go` (`Register`, `ResolveAgent`) and `internal/store/agents.go` (`scanAgent`).
- Validate: `make test`; then the roundtrip in the diary's P2 verification (register→show→update→duplicate→bad token).

### Technical details
- Token: `af_` + 32 random bytes base64url; stored `sha256` hex. ID: `ag_` + ULID.
- Errors map to future HTTP: 401 `ErrUnauthenticated`, 404 `ErrNotFound`, 409 `ErrConflict`, 422 `ErrInvalidInput`.

## Step 4: P3 subforums + watch

Subforums are now first-class: a user-chosen `key` plus title/description/metadata,
with list/create/show and explicit subforum watch/unwatch. The impact is that
threads (P4) have a home and the watched-subforum event reason (P5) has a basis.

**Commit (code):** a01d81c — "feat(agentforum): P3 subforums …"

### What I did
- `internal/store/subforums.go`: Create/List/Get/Update + `scanSubforum`; subforum_watches Watch/Unwatch/IsWatching/WatchedSubforumKeys.
- `internal/service/subforums.go`: `CreateSubforum` (key regex `^[a-z0-9][a-z0-9-]{0,62}$`, metadata validation, 409 on dup), `List/Get` (404), `Watch/Unwatch` (404 if missing, idempotent).
- `internal/cli/subforum.go`: `subforum list/create/show/watch/unwatch`; positional `key` arg via `fields.WithIsArgument(true)`; `subforumRow` + `statusRow` helpers.
- Wired 5 commands into the root `add()` builder.
- `internal/service/subforums_test.go`: create/conflict/bad-key/list/get-404/watch-idempotent/unwatch-idempotent.

### Why
- Keyed-by-`key` (not ULID) keeps subforums ergonomic in the CLI (`subforum show engineering`) and URL-safe.
- Watch is independent of participation, matching the brief's distinction; idempotent so retried agents don't double-subscribe.

### What worked
- Roundtrip green: create (with `--meta`), list (key-ordered), show, watch→`watching`, unwatch→`not_watching`; dup key→`conflict`; bad key→`invalid input`; watch missing→`not found`; unauth create→`unauthenticated`.
- `go test ./...` + `go vet` clean.

### What didn't work
- First `scanSubforum` matched `sql.ErrNoRows` by error string; replaced with `errors.Is(err, sql.ErrNoRows)` for robustness.

### What I learned
- Positional args land in the default section under their field name, so `s.Key` (`glazed:"key"`) just works alongside flags.
- `INSERT OR IGNORE` makes watch idempotent at the SQL layer without a read-then-write race.

### What was tricky to build
- Key validation regex: must reject leading hyphen and spaces/uppercase so keys stay URL- and label-safe; `Bad Key!` and `-leading` both correctly rejected.

### What warrants a second pair of eyes
- `CreateSubforum` allows any authenticated agent to create subforums (design §2.3). Confirm that's acceptable before any public exposure.
- `WatchSubforum` does a GetSubforum then an INSERT OR IGNORE — two statements, not one transaction. A subforum deleted between them would leave a dangling watch; FK constraints prevent the delete-of-subforum-with-watches case, so this is safe today.

### What should be done in the future
- Add `subforum update` (PATCH) now that the store has UpdateSubforum.
- Enforce a creation permission policy if the forum is exposed beyond a trusted agent mesh.

### Code review instructions
- Start in `internal/service/subforums.go` (`CreateSubforum`, `WatchSubforum`) and `internal/cli/subforum.go` (positional `key`).
- Validate: `make test`; then the P3 roundtrip above.

### Technical details
- Subforum key regex: `^[a-z0-9][a-z0-9-]{0,62}$` (max 63 chars). Watch table PK (agent_id, subforum_key).

## Step 5: P4 threads & posts

This phase made the forum usable: opening a thread creates the thread and its
first post atomically, posting replies makes the author a participant, threads
list/show, and thread watch/unwatch. Crucially, the write path for `events` and
`metadata_terms` lives here so P5 (events read) and P6 (metadata query) never
need a backfill.

**Commit (code):** b8f9ea2 — "feat(agentforum): P4 threads & posts …"

### What I did
- `internal/store/dbtx.go`: small interface satisfied by `*sql.DB`/`*sql.Tx` so atomic writers share helpers.
- `internal/store/metadata.go`: `flattenMetadata` (dotted keys, array→repeated-key) + `indexMetadataTermsTx` (delete-then-insert) + exported `IndexMetadataTerms`.
- `internal/store/events.go`: `appendEventTx` (autoincrement sequence) + `AppendEvent`.
- `internal/store/participants.go`: `upsertParticipantTx` (ON CONFLICT DO UPDATE), `IsParticipant`, `ListParticipantThreadIDs`.
- `internal/store/watches.go`: `watchThreadTx`/`WatchThread`/`UnwatchThread`/`IsWatchingThread`/`ListWatchingThreadIDs`.
- `internal/store/threads.go`: `CreateThreadWithPost` (atomic: thread+post+participant+terms(thread,post)+events(thread.created,post.created)+optional watch), `GetThread`, `ListThreads` (subforum/involved/watching/limit), `BumpThreadUpdatedAt`, `scanThread`.
- `internal/store/posts.go`: `CreatePost` (atomic: post+participant+terms+event+bump updated_at via `UPDATE … RETURNING`), `GetPost`, `ListPosts` (after-post by created_at), `scanPost`.
- `internal/service/threads.go`: `CreateThread` (validates title/subforum/metadata, resolves subforum), `ListThreads` (maps involved/watching to agent), `GetThread`, `WatchThread`/`UnwatchThread`.
- `internal/service/posts.go`: `CreatePost` (reply_to same-thread check), `ListPosts`, `GetPost`.
- `internal/cli/thread.go`: create/list/show/watch/unwatch; `threadRow`/`postRow` with `kind` discriminator; `thread show` emits thread then posts (--after-post/--limit).
- `internal/cli/post.go`: `post create` (--body/--body-file/--reply-to/--metadata-file/--meta/--keyword).
- `internal/cli/helpers.go`: `applyKeywords` (append to metadata.keywords).
- `internal/service/threads_test.go`: atomic create (asserts 2 events + thread terms), involved/watching listing, reply_to cross-thread/missing, missing-thread.

### Why
- Atomicity in the store layer (not the service) guarantees no partial thread is ever visible even if a statement mid-bundle fails.
- Writing events + metadata_terms at creation avoids any P5/P6 backfill migration — the read layers are additive.

### What worked
- Roundtrip green: alice creates (metadata `transcript_id`/`ticket`/2 keywords`+watch) → thread+post atomically; `list --involved` shows it with full metadata JSON; bob (0 involved) posts → bob now involved; `thread show` → 3 rows (thread+2 posts); reply_to missing→`not found`, cross-thread→`invalid input`; `metadata_terms` flattened correctly (array→multiple rows); 3 `events` in sequence; participants + watch recorded.
- `go test ./...` + `go vet` clean.

### What didn't work
- Struct literal typo: wrote `models.MetadataTerm{Key: prefix, fmt.Sprintf(...)}` (missing `Value:`) on 5 lines → "mixture of field:value and value elements". The string case (with `Value: t`) compiled, which made the others' missing `Value:` obvious once I `cat -A`'d the lines.
- Tried to mount P4 commands with a `[]struct{cmd;err}` literal containing multi-return calls — Go can't unpack multi-return in a composite literal. Reverted to explicit `add()` calls.
- First P4 manual test used `--ticket` (a P6 *list* filter) on `thread create`; create takes `--meta ticket=…`. Fixed the test.

### What I learned
- `UPDATE … RETURNING 1` keeps the thread updated_at bump inside the post tx and surfaces a missing thread as a real error instead of a silent no-op.
- ULID `created_at` ordering + `id ASC` tiebreaker makes `--after-post` pagination stable.

### What was tricky to build
- Keeping `CreateThreadWithPost` and `CreatePost` fully transactional while reusing `upsertParticipantTx`/`indexMetadataTermsTx`/`appendEventTx`/`watchThreadTx` — solved with the `dbtx` interface so the same helpers run on the tx.
- `ListThreads` "involved" = creator OR participant OR watcher, assembled as one parameterized `WHERE` with three `?` for the same agent id.

### What warrants a second pair of eyes
- The atomic writers use a single connection (`SetMaxOpenConns(1)` from P1). A long P5 long-poll read holds the connection; verify it does not block a concurrent writer (WAL should make reads non-blocking, but the single-conn pool defeats that — flag for P5).
- `ListPosts` after-post filters by `created_at > after.created_at`; two posts with identical nanosecond timestamps could skip the after-post itself but include its peers. ULIDs make this near-impossible; document it.

### What should be done in the future
- Revisit `SetMaxOpenConns` for read concurrency once P5 long-poll lands (maybe read-only pool + writer connection).
- Add `thread update` (title/metadata) reusing the store's bump + reindex.

### Code review instructions
- Start in `internal/store/threads.go` (`CreateThreadWithPost`) and `internal/store/posts.go` (`CreatePost`) for the atomicity guarantees.
- Validate: `make test`; then the P4 roundtrip (create→list-involved→reply→show→metadata_terms→events).

### Technical details
- Atomic bundle (create thread): thread + post + participant + 2 terms indexes + 2 events + optional watch, one `BEGIN/COMMIT`.
- `kind` column distinguishes thread vs post rows in `thread show`/`thread create` output.

## Step 6: P5 events + long-poll

This phase built the unified inbox: `events poll` long-polls a monotonic cursor,
computing per-agent reasons (participating/watching/watched_subforum) at query
time and excluding the agent's own actions; `events follow` streams; `events ack`
records durable progress. The impact is the brief's central feature — one stream
across all threads an agent cares about.

**Commit (code):** 5744e9a — "feat(agentforum): P5 events …"

### What I did
- `internal/store/events.go`: `ListEventsAfter(cursor, limit)` (asc sequence), `AckEvents` (upsert `event_acks`), `GetAck`, `scanEvent`.
- `internal/service/events.go`: `PollEvents` (long-poll loop with `pollPageSize`/`pollInterval`), `parseScope` (involved→participating, watching, watched-subforums), `membershipSets` (batched participant/watched-thread/subforum sets), `eventReason` (precedence participating>watching>watched_subforum, self-excluded by caller), `AckEvents`/`GetAck`, `sleepCtx`.
- `internal/cli/events.go`: `events poll --cursor/--wait/--scope/--since-ack`, `events follow --wait/--scope/--since-ack` (loops until SIGINT), `events ack --through-sequence`; `eventRow` carries `next_cursor` per row; `emptyPollRow` for empty polls.
- `internal/service/events_test.go`: reasons + self-exclusion + scope + ack; a long-poll concurrency test (poller blocks, poster fires, poller returns within deadline).

### Why
- Reason computed at query time (not stored) keeps writes O(1) and lets watching start/stop without rewriting history (Decision Records).
- Forward-only cursor (advance past every examined event) avoids busy-loops on self-events while delivering eligible events before advancing past them.

### What worked
- bob (watches thread) → 3 events `watching`; carol (watches subforum) → 3 `watched_subforum`; alice (all self) → 0 but `next_cursor=3`; `--scope involved` (bob not a participant) → 0; `ack 3` then `--since-ack` → empty.
- Long-poll: bob `poll --since-ack --wait 4` in background, alice posts after 1s → poll returns event seq 4 `post.created` within the window (proven both via CLI and a Go concurrency test).
- `go test ./...` + `go vet` clean.

### What didn't work
- Duplicate `case "watched-subforum"` in `parseScope` (I listed the token twice); fixed to one combined case.
- First events test referenced `service.Event` (no such type); events are `models.Event`. Fixed import + channel type.

### What I learned
- The single-connection pool from P1 does NOT block the long-poll concurrency: the test's poller and poster run on the same `*sql.DB` and SQLite WAL + the poller sleeping (not holding a tx) let the post land. So P1's `SetMaxOpenConns(1)` concern (flagged in P4) is fine for long-poll reads, which never hold an open transaction while waiting.

### What was tricky to build
- Cursor advancement vs. retroactive eligibility: a thread you watch NOW makes its past events eligible (reason-at-query-time). I deliver them if your cursor hasn't passed them; once advanced, they're gone. This is intentional forward-only behavior, documented.
- Keeping the wait loop from busy-spinning: a full page of ineligible events advances the cursor and loops immediately (no sleep); only when caught up (page < limit) does it sleep until the deadline.

### What warrants a second pair of eyes
- `eventReason` precedence: a thread you both participate in AND watch returns `participating` (not `watching`). Confirm that's the desired single `reason` field; the brief shows one reason per event.
- `events follow` loops forever and only stops on ctx cancellation (SIGINT) or error; a transient store error would terminate the stream. Acceptable for a CLI loop but note for a future server SSE path (retry/backoff).

### What should be done in the future
- Enrich event rows with post body + author display name (batched per page) so the inbox is self-contained, matching the brief's event shape.
- Add SSE `/v1/events/stream` reusing `PollEvents` with the same cursor/next_cursor contract.

### Code review instructions
- Start in `internal/service/events.go` (`PollEvents`, `eventReason`, `parseScope`) and `internal/cli/events.go` (`EventsFollowCommand` loop).
- Validate: `make test` (includes the long-poll concurrency test); then the P5 CLI roundtrip above.

### Technical details
- `pollPageSize=500`, `pollInterval=200ms`. Scope default `involved,watching,watched-subforums`. `event_acks` PK agent_id.
- `next_cursor` is emitted on every event row (and on the empty-poll row) so JSONL consumers can resume from any line.

## Step 7: P6 metadata & search + idempotency

This phase made metadata queryable and writes replay-safe. The `metadata_terms`
index written since P4 is now queried via `--meta/--keyword/--ticket`; `post search`
and cross-entity `search` work; and `--idempotency-key` collapses retried
thread/post creates to the first result. The impact: agents can find threads by
transcript/ticket/keyword and retry writes without duplicates.

**Commit (code):** 37f0c85 — "feat(agentforum): P6 metadata & search …"

### What I did
- `internal/store/idempotency.go`: `GetIdempotencyRecord`/`SaveIdempotencyRecord` + `IdempotencyRecord`.
- `internal/store/search.go`: `TermFilter` (multi-key OR), `SearchFilter`, `appendTermExists`/`metadataWhere` helpers, `SearchThreads` (title LIKE + terms), `SearchPosts` (join threads for subforum + body LIKE + terms), `postColumnsQualified` to avoid ambiguous `id`.
- `internal/store/threads.go`: `ListThreadsFilter.Terms` + EXISTS clauses in `ListThreads`.
- `internal/service/search.go`: `SearchInput`, `SearchThreads`/`SearchPosts`/`Search` (combined, entity-type selection); re-export `TermFilter`.
- `internal/service/threads.go`+`posts.go`: `IdempotencyKey` on inputs; replay via cached JSON (`{thread,initial_post}` / `{post}`), save after create.
- `internal/cli/helpers.go`: `buildTerms` (--meta k=v → key=k; --keyword → key=keywords; --ticket → keys=[ticket,external_refs.value]).
- `internal/cli`: `thread create/list` + `post create` get `--idempotency-key`/`--meta`/`--keyword`/`--ticket`; new `post search` and top-level `search <text>` commands.
- `internal/service/search_test.go`: idempotent thread+post replay (asserts 1 thread / 2 posts), metadata/keyword/ticket/AND/text search, combined search.

### Why
- Idempotency at the service boundary (check→create→save) means the store's atomic writers stay simple; the cached JSON response makes replay return the *exact* first result.
- `--ticket` matching either `ticket` or `external_refs.value` shows the multi-key OR in `TermFilter` is worth the small extra generality.

### What worked
- Idempotency: create with `run_204-thread`, retry with different args → same `id`, 1 thread in DB; post idempotency likewise (1 reply, not 2).
- `thread list --meta transcript_id=tr_892` / `--keyword caching` / `--ticket PLAT-431` → 1; `--meta ticket=NOPE` → 0.
- `post search --meta turn_id=turn_21` → 1 post (right body); `search "locale" --entity post` → 1; `search "" --ticket PLAT-431` → thread; `search "" --subforum eng --keyword root-cause` → post.
- `go test ./...` + `go vet` clean.

### What didn't work
- `SearchPosts` joined `threads` and selected bare `id` → `ambiguous column name: id`. Fixed with `postColumnsQualified` (`posts.id, …`).
- The edit tool refused edits whose `oldText`/`newText` contained Go string literals with embedded double quotes ("must be object"); switched those edits to a Python patch. A known sharp edge to remember.
- Test typo `Tems:` (for `Terms:`); fixed with sed.
- Combined-search test asserted `threads=1` for text `"cache"`, but "Caching" ≠ contains "cache"; fixed by putting "Caching" in the post body and searching "caching".

### What I learned
- SQLite `LIKE` is case-insensitive for ASCII by default, so `search "caching"` matches title `Caching investigation`.
- Qualifying columns in any JOIN query is mandatory even when the non-joined table isn't selected; `id` collides because `threads.id` exists.

### What was tricky to build
- Idempotency replay must unmarshal the cached JSON back into `*models.Thread`/`*models.Post` with their `map[string]any` metadata; JSON round-trips `map[string]any` fine, so replay returns the original metadata too.
- AND-combined term filters each need their own `EXISTS` subquery with distinct placeholders; the `entity_type` placeholder is added once per filter before its keys.

### What warrants a second pair of eyes
- `SaveIdempotencyRecord` uses `INSERT OR REPLACE`; if a key collision happened between two *different* agents (keys are per-agent in the lookup but the PK is just `key`), one would overwrite the other's record. Keys should be globally unique (they're opaque/random per the brief); confirm or change the PK to `(key, agent_id)`.
- `search` text uses unanchored `LIKE %text%`; no escaping of `%`/`_` in user text. Low risk for a trusted CLI but note it.

### What should be done in the future
- Escape `%`/`_` in search text or switch to a real FTS5 index for large datasets.
- Add a `search --created-after` / `--limit` wired to `CreatedAfter` (store supports it; CLI flag not added yet).
- Consider per-subforum JSON-Schema metadata validation (reserved `_` keys already enforced).

### Code review instructions
- Start in `internal/store/search.go` (`metadataWhere`, `SearchPosts`) and `internal/service/threads.go` (idempotency replay).
- Validate: `make test`; then the P6 roundtrip (idempotent create, --meta/--keyword/--ticket, post search, combined search).

### Technical details
- `TermFilter{Keys, Value}`: OR over keys, AND over filters. Idempotency cached JSON: `{"thread":…,"initial_post":…}` and `{"post":…}`.

## Step 8: P7 hardening & ship

This final phase closed the milestone: a README quickstart, the full validation
gate, a reMarkable re-upload of the now-complete diary, and the final task/slip
bookkeeping. The impact is that the project is reproducible (`make build/test`),
documented, and deliverable.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish P7 — README, validation, reMarkable diary upload, final slip.

**Inferred user intent:** A shipped, validated, documented CLI-only agentforum with a paper trail.

### What I did
- Wrote `README.md` (layout, config table, quickstart, key concepts, validation, status).
- Ran the full gate: `gofmt -l` (clean), `go test ./...` (store+service green, incl. long-poll concurrency), `go vet` (clean), `go build ./...` (clean), `git diff --check` (clean).
- Smoke-tested a fresh DB: db init → register → subforum → thread (--watch) → post → `events poll` (self-exclusion → empty-poll row, as designed).
- Re-uploaded the design doc + complete diary bundle to reMarkable.
- Checked the P7 task; ran `docmgr doctor` (clean).

### Why
- The validation gate is the contract: every phase ends with `make test/vet/build`, so the milestone close is just running it once more on the whole tree.
- Re-uploading the diary gives the tablet the complete journey, not just the planning snapshot.

### What worked
- Full gate green on the first close attempt (no new failures surfaced).
- Smoke confirmed self-exclusion: a sole agent's `events poll` returns the empty-poll marker row (`events: 0`), not its own activity.

### What didn't work
- (Nothing.)

### What I learned
- The empty-poll row is a useful invariant to assert: it proves the cursor advanced past self-events without delivering them.

### What was tricky to build
- Resisting scope creep at the finish: the HTTP server, SSE, and per-subforum JSON-Schema are deliberately documented as future phases, not implemented now.

### What warrants a second pair of eyes
- The CLI has no `cli`-layer tests yet (only store+service). The design's test strategy calls for a couple of end-to-end `Execute` tests; that's the top follow-up.
- `idempotency_keys` PK is `key` only (per-agent in lookup but global PK); flagged in Step 7 — confirm global uniqueness or switch PK to `(key, agent_id)`.

### What should be done in the future
- Add CLI-layer tests (build each command, assert Glazed universal flags, end-to-end run on a temp DB).
- Implement the HTTP server phase reusing `*service.Service` (design §10).
- Add SSE `/v1/events/stream` reusing `PollEvents`.
- Escape `%%`/`_` in search text or move to FTS5.

### Code review instructions
- Run `make fmt test vet build`; read `README.md` and re-run the quickstart.
- Confirm `docmgr doctor --ticket AGENTFORUM-001` is clean and all 7 tasks are checked.

### Technical details
- reMarkable: `/ai/2026/09/03/AGENTFORUM-001` (design doc + diary bundle).
- Final commit graph: P1 cbdc6a6 → P2 dbf44e4 → P3 a01d81c → P4 b8f9ea2 → P5 5744e9a → P6 37f0c85 → P7 (this step).
