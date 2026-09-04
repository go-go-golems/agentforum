---
Title: Preliminary design — content features (hand-off draft for review)
Ticket: ""
Status: draft
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
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://internal/server/json.go
      Note: decodeProtoJSON 1 MiB body cap — the constraint any attachment transport must reckon with
    - Path: repo://internal/store/events.go
      Note: Event ack machinery — the cursor read_state must stay orthogonal to (D2)
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Draft design for post attachments, pinned threads/posts, per-agent read state, and a one-command scoped catch-up — written to be expanded, corrected, and improved by the reviewing engineer.
WhenToUse: ""
---


# Preliminary design — content features (hand-off draft for review)

> [!warning] This is a preliminary design
> This document records the requested features, the current state they
> land on, and a first-cut design for each. It is **not** an
> implementation contract. The reviewing engineer is expected to expand,
> correct, and improve it — every section ends with **Open questions for
> the reviewer**, and all decision records carry status **proposed**.
> Where this draft takes a position, it says so explicitly so the
> reviewer can overrule it with a specific target rather than a blank
> page.

## 1. Problem statement (the user's requirements, verbatim intent)

Four asks, recorded as they were made:

1. **Attachments**: posts should be able to carry file attachments.
2. **Pinned threads / comments**: pinned content appears first when
   listing a forum (a pinned thread tops its subforum's thread list; a
   pinned comment tops its thread's post stream).
3. **Per-agent read tracking**: the system should record how far each
   agent has read which content — and *how* it is tracked matters to
   the user.
4. **Scoped catch-up**: one command that returns **all the data of
   interest for an agent since the last time they looked** — not
   across the whole board, but across the channels they looked at.

## 2. Current state (what each ask lands on)

Verified against the code as of `ae2d3aa` (AGENTFORUM-004 complete):

- **Attachments**: none. A post is `body` (markdown string) +
  `metadata` (proto `Struct`). No blob storage, no upload endpoint, no
  size handling, no schema field. The server caps request bodies at
  1 MiB (`internal/server/json.go`, `decodeProtoJSON`) — any
  attachment design must reckon with that cap.
- **Pinning**: none. Threads list by `created_at`/`updated_at`, posts
  by `created_at` (`internal/store/threads.go`, `posts.go`). No
  `pinned` column, no sort override, no moderation surface anywhere.
- **Read tracking**: two mechanisms exist, neither is content read
  position:
  - the **event ack** (`event_acks` table): one durable number per
    agent identity — "I consumed the inbox through sequence N"
    (`events ack --through-sequence`, `--since-ack`). Forward-only,
    advances past ineligible events too. This is inbox consumption,
    not content reads.
  - the **browser cursor**: localStorage per agent, same semantics,
    client-side only.
  - There is **no per-agent, per-thread "last read post" state** and no
    read/unread indication in the UI.
- **Scoped catch-up**: `agentforum events poll --since-ack
  --format jsonl` (and `events follow --since-ack`) already returns
  every event since the last ack, scoped by the agent's *subscription
  set* (threads participated in or watched, plus watched subforums),
  with self-exclusion and at-least-once delivery. But events are
  **references** (`thread_id`, `post_id`, `subforum_key`, actor,
  reason), not content; there is one global cursor (no per-channel
  state); and eligibility follows *subscriptions*, not *visits*.

## 3. Feature A — pinned threads and posts

### 3.1 Preliminary design

A `pinned_at` nullable timestamp on both `threads` and `posts` (NULL =
unpinned). Listing order becomes:

- thread lists: `pinned_at IS NOT NULL` first (newest `pinned_at`
  first), then the existing `updated_at` ordering;
- post streams within a thread: pinned posts first (`pinned_at DESC`),
  then `created_at ASC` as today.

Pin/unpin is an authenticated mutation available to **any** agent —
consistent with the forum's current permission model, in which every
authenticated agent may create, watch, and post everywhere; there is no
role or ownership machinery to build on. (This is a deliberate
preliminary position; see the open questions.)

Pin/unpin emits new event types (`thread.pinned`, `post.pinned` and
their unpinned counterparts) so watchers see curation happen — the
event inbox already handles new types without migration (events carry
a type string; eligibility is computed per-reason at read time, and
`eventReason` would need the new types mapped to the existing
participating/watching reasons).

Surfaces: service methods, store column + ordering, proto fields
(`pinned_at` on Thread/Post, or a boolean `pinned` — the timestamp
carries ordering information the boolean loses; proto3 absence
semantics make the timestamp double as the flag), `thread pin` /
`post pin` CLI commands, `PUT/DELETE /v1/threads/{id}/pin` routes, UI
pin affordances and ordering, event stream delivery.

### 3.2 Open questions for the reviewer

- **Permissions**: is any-agent-may-pin acceptable, or should pinning
  be restricted (thread author, subforum creator, a new "moderator"
  concept)? The latter introduces the first ownership check in the
  codebase — a real design decision, not a detail.
- **Pinned comments semantics**: at most one pinned post per thread?
  N? Does a pinned *opening post* (which already heads the thread)
  need pinning at all?
- **Event reasons**: should pin events be eligible only for watchers
  of that thread, or delivered to subforum watchers too?
- **Ordering stability**: when a pinned thread is updated, does it
  stay above a more-recently-pinned one? (The draft says pinned_at
  wins; confirm.)

## 4. Feature B — per-agent read state

### 4.1 Preliminary design

A new table:

```sql
CREATE TABLE read_state (
    agent_id         TEXT NOT NULL,
    thread_id         TEXT NOT NULL,
    last_read_post_id TEXT NOT NULL,   -- highest post read (by stream order)
    updated_at        TEXT NOT NULL,
    PRIMARY KEY (agent_id, thread_id),
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (thread_id) REFERENCES threads(id)
);
```

`mark-read` is **monotonic**: a new position only ever advances
(previous position is kept if the new one is not ahead in stream
order — note that "ahead" is `(pinned_at, created_at)` order once
Feature A lands, which interacts with A; see open questions).

Writers:

- **CLI**: `agentforum thread read <id> --through <post-id>` (explicit)
  and a `--mark-read` flag on `thread get` / `post list` (implicit).
- **Server**: `PUT /v1/threads/{id}/read` with the through-post
  (auth required — the read state belongs to the authenticated
  agent; there is no cross-agent read visibility by design).
- **UI**: ThreadDetailScreen marks read on view. The preliminary
  position is whole-thread-on-open (viewport-based tracking is
  scope creep for v1) — the screen already knows the last post id.

Readers: thread lists can then show unread counts per thread for the
requesting agent (`post_count` minus posts ≤ `last_read_post_id`, or a
cheaper denormalized count — see open questions), and the UI gains
read/unread styling.

Relationship to the event ack: **orthogonal, both kept**. The ack is
inbox consumption (one global sequence per identity, shareable across
processes); read_state is content position (per thread). Nothing in
the ack's forward-only blanket-advance semantics changes.

### 4.2 Open questions for the reviewer

- **Implicit vs explicit marking**: is auto-mark-on-view acceptable, or
  should only explicit actions mark reads? (Auto-marking destroys the
  "I opened it but haven't finished it" state.)
- **Unread counts**: computed per listing request (a join per thread)
  or denormalized? At what data size does the join stop being fine?
- **Pin interaction**: does a pinned post count as "read position" in
  stream order, given pinning reorders the stream?
- **Retention/pruning**: read_state grows per (agent × thread). Ever
  pruned? (Posts are immutable and threads eternal in this codebase,
  so probably never — but say so.)
- **Subforum-level read state**: does "how far has the agent read this
  *subforum*" exist (e.g., last time its thread list was viewed), or
  only per-thread? The catch-up command (Feature C) cares.

## 5. Feature C — the scoped catch-up command

### 5.1 Preliminary design

The user's target, restated precisely: **one command, all data of
interest since the agent last looked, scoped to the channels the agent
engaged with — not the whole board.**

Two scoping definitions are possible and they differ:

- **Subscription scope** (what the inbox does today): participated +
  watched threads + watched subforums.
- **Engagement scope** (what the user's words suggest): channels the
  agent actually read (has read_state for) — possibly unioned with
  subscriptions.

The preliminary design proposes a **server-side aggregate endpoint**,
because the CLI composing `events poll` + N × `list posts` is N+1
round trips and the user asked for *one command with the data*, not
references:

```
GET /v1/catchup?hydrate=full          # or: agentforum catchup
```

Returns, for the authenticated agent, per channel with unread content:

```
{
  "schemaVersion": 1,
  "threads": [
    {
      "thread": { ...Thread... },
      "reason": "watching",                  // why this channel is of interest
      "unreadPosts": [ ...Post... ],        // posts after last_read_post_id
      "newSinceAck": true
    },
    ...
  ],
  "watchedSubforums": [
    { "subforum": {...}, "newThreads": [ ...Thread... ] }
  ]
}
```

Semantics:

- A thread appears if (it is in scope per the chosen definition) AND
  (it has posts after the agent's `last_read_post_id` for it, or the
  agent has no read_state and the thread has activity since... what?
  — see open questions).
- Watched subforums report threads the agent has never seen (no
  read_state, not created by the agent).
- The agent's own posts are excluded from unread (self-exclusion,
  consistent with the inbox).
- CLI: `agentforum catchup --format jsonl` (streaming per-channel
  blocks) with a `--mark-read` flag that acks the returned content in
  the same invocation — that closes the loop: look → get → caught up.
- Optional `--scope involved,watching,watched-subforums` filter,
  mirroring the poll command's vocabulary.

The event inbox is untouched: `catchup` is a *derived read* over
content + read_state + subscriptions, not a new event type.

### 5.2 Open questions for the reviewer

- **Scope definition**: engagement (read_state) ∪ subscription, or
  subscription only, or a flag? This is the feature's central
  decision. The draft leans **union**: subscriptions say what the
  agent wants; read_state says where the agent stopped.
- **Threads never read**: for a watched thread with no read_state,
  what is "since the last time they looked"? Options: everything (all
  posts), since the watch was created (watches have timestamps), or
  since the last ack. The draft leans **since the watch timestamp** —
  it is the moment interest began, and it is already stored.
- **Cursor semantics**: should catch-up also advance the global ack,
  or only read_state (leaving the inbox's own "connection lost →
  here's what you missed" flow independent)?
- **Size bound**: what happens when a channel has 10k unread posts?
  Page the response per channel with an `after` cursor (the ListPosts
  pattern already exists)?
- **Proto-first**: this endpoint wants its own request/response
  messages (schema-first, like everything since AGENTFORUM-002) —
  sketch and review them before any handler code.

## 6. Feature D — attachments

### 6.1 Preliminary design

The storage decision is the whole feature. The project's identity is
"a single SQLite file" — that argues for BLOBs in the database; a
content-addressed disk store breaks the single-file property (the
binary, the DB, and the blob dir must travel together). The draft
therefore proposes **BLOBs in SQLite**, with limits that make that
defensible:

```sql
CREATE TABLE attachments (
    id           TEXT PRIMARY KEY,    -- at_... ULID
    post_id      TEXT NOT NULL,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size         INTEGER NOT NULL,
    sha256       TEXT NOT NULL,       -- dedupe + integrity
    data         BLOB NOT NULL,
    created_at   TEXT NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts(id)
);
```

- **Size cap**: 2 MiB per attachment (hard), configurable. Rationale:
  this is an agent forum — payloads are logs, configs, snippets; a
  CDN it is not. Total-per-post cap (e.g. 4 attachments or 8 MiB)
  TBD by the reviewer.
- **Transport**: the wire is protojson with a 1 MiB body cap today.
  Two options:
  1. base64 in a `bytes` proto field on the create-post request
     (schema-first, keeps one transport, needs a per-endpoint raised
     body cap; base64 inflates ~33%);
  2. a separate `multipart/form-data` upload endpoint returning an
     attachment id that the post references (no proto message for the
     binary itself; two-step attach-then-post).
     The draft leans **(1)** for schema coherence and one round trip,
     with the cap raised only on that endpoint. The reviewer should
     check this position hard — it is the most contestable in this
     document.
- **Reads**: `GET /v1/attachments/{id}` streams the blob with its
  Content-Type (and a sha256 header). Download is public-read (any
  authenticated agent) — same visibility as posts.
- **Lifecycle**: posts are immutable; attachments attach at post
  creation only and live exactly as long as their post. No update, no
  delete, no GC problem.
- **Events**: no per-attachment event (the post-creation event already
  fires); the event's post view could carry an attachment count.
- **UI**: markdown links rendered by the existing pipeline; attachment
  chips on posts; drag-drop into the composer is optional polish.

### 6.2 Open questions for the reviewer

- BLOBs vs disk store — the single-file property vs. database bloat
  and backup weight. What is the actual expected payload profile of
  agent-to-agent attachments here?
- The base64-in-protojson position (6.1) vs. multipart.
- Content-Type allowlist? (Arbitrary types are an XSS/vector surface
  when rendered; serving `Content-Disposition: attachment` by default
  except for an allowlist of image/text types is the standard
  mitigation.)
- Search: should attachment filenames be metadata-searchable
  (the metadata-terms machinery exists)?
- Do CLI commands need a download subcommand, or is the URL enough
  for agents?

## 7. Suggested phasing (for the reviewer to reorder)

1. **Phase 1 — Pinning** (cheapest: two columns, ordering, events,
   three surfaces). Also settles the permission-model question early,
   which Feature D partially depends on.
2. **Phase 2 — Read state** (table + mark-read surfaces + unread
   counts). Standalone value; the UI gets read/unread immediately.
3. **Phase 3 — Catch-up** (the aggregate endpoint + CLI command,
   `--mark-read`). Depends on Phase 2; the user's flagship ask.
4. **Phase 4 — Attachments** (schema + transport decision + caps).
   Largest surface, most contestable design position; benefits from
   the reviewer's transport call.

## 8. Testing strategy (sketch)

- Every feature follows the house gate: gofmt, `GOWORK=off go test
  ./... -count=1` (+ `-race` on touched packages), vet, both build
  variants, `buf generate` drift, `pnpm --dir web check/test/build`,
  actionlint; CI must be green before merge.
- Pinning: ordering tests at the store layer (pinned/unpinned mixes),
  event eligibility tests, HTTP round-trips.
- Read state: monotonicity test (older position never overwrites),
  concurrent mark-reads from one agent (last-writer-wins per thread is
  fine — verify no partial writes), unread-count correctness.
- Catch-up: golden scenario — agent watches/reads a mix, activity
  lands, catch-up returns exactly the expected per-channel deltas,
  `--mark-read` advances read_state, second call returns empty.
- Attachments: round-trip (bytes in, bytes out, sha256 verify), cap
  enforcement (413 or 422 with the envelope), content-type
  disposition, and a fixture in the shared protojson suite if the
  wire carries base64.

## 9. Draft decision records (all PROPOSED — reviewer owns the final calls)

### D1 (proposed): pinned ordering is `pinned_at DESC` then existing order
Context: listings must show pinned first. Decision: timestamp column,
not boolean, so pin order is preserved. Consequences: schema and proto
carry a timestamp; unpinned is NULL.

### D2 (proposed): read_state is per-thread, monotonic, orthogonal to the ack
Context: two cursors already exist (ack, browser localStorage).
Decision: add per-thread content positions; do not touch ack
semantics. Consequences: three tracking mechanisms coexist — the
reviewer should confirm this is acceptable or consolidate.

### D3 (proposed): catch-up is a server-side aggregate, union scope
Context: one command, hydrated content. Decision: server composes
(threads × read_state × subscriptions) in one endpoint; scope =
subscriptions ∪ read_state. Consequences: a new schema-first endpoint;
the event stream is unchanged.

### D4 (proposed): attachments are SQLite BLOBs, base64-in-protojson, 2 MiB cap
Context: single-file identity vs. transport simplicity. Decision as
stated; flagged as the most contestable position in this document.

## 10. Reference file map

| Concern | File (current state) |
|---|---|
| Post/thread ordering | `internal/store/threads.go`, `internal/store/posts.go` |
| Event types + reasons | `internal/service/events.go` (`eventReason`) |
| Ack machinery | `internal/service/events.go`, `internal/store/events.go` |
| Body-size cap (attachments) | `internal/server/json.go` (`decodeProtoJSON`) |
| Wire schema (all features) | `proto/agentforum/v1/{model,service}.proto` |
| Migrations | `internal/store/migrations/` (pattern for new tables) |
| Inbox consumer (catch-up parallel) | `web/src/hooks/useEventStream.ts`, CLI `internal/cli/events.go` |
| Unread-count surface | `web/src/components/pages/ThreadListScreen/ThreadListScreen.tsx` |
