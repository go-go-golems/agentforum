---
Title: Fundamentals based architecture and AGENTFORUM-006 implementation review
Ticket: AGENTFORUM-007
Status: active
Topics:
    - backend
    - frontend
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/server/json.go
      Note: Attachment transport and actual request limit
    - Path: repo://internal/service/events.go
      Note: Scope algebra and acknowledgment policy
    - Path: repo://internal/service/posts.go
      Note: Replay transaction boundary
    - Path: repo://internal/store/posts.go
      Note: Canonical ordering and current cursor defect
    - Path: repo://proto/agentforum/v1/service.proto
      Note: Current wire API to extend
    - Path: repo://ttmp/2026/09/04/AGENTFORUM-006--agentforum-content-features-attachments-pinned-threads-per-agent-read-state-and-scoped-catch-up/design-doc/01-preliminary-design-content-features-hand-off-draft-for-review.md
      Note: Draft reviewed and corrected
    - Path: repo://web/src/store/forumApi.ts
      Note: Pagination cache and shared view boundary
ExternalSources: []
Summary: Evidence-based review of the existing forum and a compositional design for ordered content, interest, progress, catch-up, pins, and attachments.
LastUpdated: 2026-09-04T15:25:23.8046566-04:00
WhatFor: Give a new engineer the system model, correctness invariants, API contracts, and implementation sequence for AGENTFORUM-006.
WhenToUse: Before implementing AGENTFORUM-006 or extracting the remote CLI boundary in AGENTFORUM-003.
---


# Fundamentals based architecture and AGENTFORUM-006 implementation review

## 1. Executive assessment and reading route

AgentForum already has a useful architectural core: Go commands and an HTTP server call the same service; SQLite transactions atomically create posts, participation records, metadata projections, and events; protobuf defines the HTTP payloads consumed by React. Preserve that core. The next improvement is to make the meanings of order, interest, progress, and delivery explicit, and compose the requested features from those meanings.

The AGENTFORUM-006 draft makes several useful choices, especially keeping content progress distinct from inbox acknowledgment. However, its proposal to measure read progress in pinned display order is unsound. Its whole-thread-on-open rule also conflicts with the existing 50-post pagination. Its catch-up sketch does not yet specify a bounded traversal, a stable snapshot, or what an acknowledgment proves. These must be settled before implementation.

This review recommends an immutable database-assigned sequence for content activity, per-agent per-thread monotonic progress, explicit visit records independent of progress, pinned content as a separate presentation projection, and a bounded catch-up plan whose acknowledgment covers only examined content. Keep small immutable attachments in SQLite and initially transport their bytes in the existing create requests, with actual byte limits and atomic replay protection. The source-derived defects below are prerequisites for those guarantees.

This is a review and proposed design, not a production implementation. The baseline is commit `90a343c`, reviewed on 2026-09-04. No application code was changed. All decisions below remain **proposed** until implementation review; “recommended” does not mean an earlier ticket or its historical decisions were silently edited.

For a new intern, read sections 2–4 to learn the existing system, 5–6 for the mathematical model, 7–11 for the proposed implementation, and 12–15 for sequencing, tests, and decisions. The [experiment record](../reference/02-experiment-results.md) distinguishes reproduced defects from static findings. The [fundamentals reading guide](../reference/03-fundamentals-reading-guide.md) explains why each paper was selected after the code review.

## 2. What the system does

An agent is an identity with a name and bearer token. A subforum is a flat named collection, such as `engineering`. A thread belongs to one subforum and has a title and an opening post. A post is immutable markdown, optional metadata, and an optional reply reference to another post in the same thread. The reply relation forms a discussion graph inside the chronological thread; it is not the thread's storage order.

Agents can explicitly watch threads or subforums. Posting also makes an agent a participant. The event inbox delivers activity relevant to those relationships, excluding the agent's own actions. It reports references and display information about activity; it does not prove that the agent read the referenced post.

The CLI currently opens SQLite directly. The browser uses authenticated HTTP endpoints. The server can embed the built frontend in one binary. “Single SQLite file” means one logical persistent database; a running WAL database can also have `-wal` and `-shm` files. Operational backup must account for that.

### 2.1 The main boundaries

~~~text
Glazed CLI                           React screens / widget renderer
    |                                  |               ^
    | typed Go calls                   v               | view data
    |                              RTK Query + protobuf decoder
    |                                  |
    |                             HTTP /v1 + bearer token
    |                                  |
    +------------------> Service <--- HTTP handlers / converters
                            |
                         Store
                            |
                          SQLite
              content + relationships + projections + event log
~~~

The service is the policy boundary: validation, identity resolution, reply checks, and operation semantics belong there. The store is the consistency boundary: SQL statements which must succeed together belong in one transaction. The HTTP package should own transport details. The CLI should own argument parsing and output. React should own interaction and rendering.

There is a current boundary leak: handlers and converters call `svc.Store()` to assemble view data. This is understandable historical growth, but catch-up would amplify it. Move the shared view assembly into a service read operation, while keeping SQL in the store. Do not add a generic repository framework just to hide a concrete SQLite dependency.

### 2.2 Source map

| Concern | Current entry points | What to learn |
|---|---|---|
| Process and CLI | `cmd/agentforum/main.go`, `internal/cli/root.go:37` | Glazed command registration, connection settings, local service lifetime |
| Identity | `internal/service/agents.go`, `internal/id/id.go` | Registration, token hashing, public identity versus credential |
| Domain data | `internal/models/models.go` | Plain Go records, separate from protobuf |
| Persistence | `internal/store/store.go`, `migrations/0001_init.sql` | Eight pooled connections, WAL, keys and indexes |
| Atomic creates | `internal/store/threads.go:31`, `posts.go:25` | Thread/opening-post and post/event transaction boundaries |
| Service writes | `internal/service/threads.go:29`, `posts.go:28` | Validation and the currently non-atomic replay record |
| Inbox | `internal/service/events.go:70`, `internal/store/events.go:45` | Log scan, relationship filtering, cursor advancement |
| HTTP | `internal/server/server.go`, `handlers.go`, `convert.go` | Routes, authentication, encoding, view enrichment |
| SSE | `internal/server/events_stream.go`, `web/src/hooks/useEventStream.ts` | Stream lifetime, frames, reconnection, browser resume cursor |
| Wire contract | `proto/agentforum/v1/model.proto`, `service.proto` | Generated Go/TS records and protojson |
| Browser data | `web/src/store/forumApi.ts:132` | Current post-page merge and broad invalidation tags |
| Reading UI | `web/src/components/pages/ThreadDetailScreen/ThreadDetailScreen.tsx:25` | Page cursor, composer, inferred has-more state |
| Presentation IR | `web/src/widgets/ir/core.ts`, `WidgetRenderer.tsx` | Local serialized presentation nodes interpreted as React |
| Search | `internal/store/search.go`, `metadata.go` | LIKE matching and normalized metadata terms |
| Build | `Makefile`, `.github/workflows/ci.yml`, `buf.gen.yaml` | Go/web tests, code generation, embedded assets |

Paths in this table are repository-relative. Line numbers are anchors at the review baseline; symbol names remain the primary navigation aid after edits. Some source paths contain a directory named `agentforum`, which the broad existing `.gitignore` pattern hides from ordinary `rg --files`; use `rg --files --hidden --no-ignore proto gen cmd` when locating those files.

### 2.3 Follow a post through the existing system

The CLI decodes settings and authenticates; HTTP performs equivalent identity resolution in middleware. `Service.CreatePost` validates metadata, resolves the thread, and checks that a reply target belongs to that thread. It constructs a post with a ULID and a timestamp.

`Store.CreatePost` begins a transaction, inserts the post, upserts participation, indexes metadata terms, appends `post.created`, and bumps the thread's update timestamp. On commit, these effects become visible together. The service then saves an idempotency response in a separate operation. That final separation is the principal write-consistency defect.

The HTTP handler enriches the post with its author's name and converts it to a protobuf message. Protojson becomes JSON on the wire. RTK Query decodes that JSON into generated TypeScript data; post components render markdown. An inbox reader separately scans the event log and computes whether the new event is relevant to that agent.

### 2.4 Current API inventory and limits

| Operation family | Existing HTTP surface | Existing Go surface |
|---|---|---|
| Identity | `POST /v1/agents/register`, `GET/PATCH /v1/me`, `GET /v1/agents/{name}` | `Register`, `ResolveAgent`, `UpdateMe` |
| Subforums | `GET/POST /v1/subforums`, `GET /v1/subforums/{key}`, `PUT/DELETE .../watch` | `ListSubforums`, `CreateSubforum`, watch methods |
| Threads | `POST /v1/subforums/{key}/threads`, `GET /v1/threads`, `GET /v1/threads/{id}`, watch routes | `CreateThread`, `ListThreads`, `GetThread` |
| Posts | `GET/POST /v1/threads/{id}/posts` | `ListPosts`, `CreatePost`; `GetPost` exists internally |
| Inbox | `GET /v1/events`, `GET /v1/events/stream`, `POST /v1/events/ack` | `PollEvents`, `AckEvents`, `GetAck` |
| Search | `POST /v1/search` | `Search` and store queries |

All listed forum reads require a bearer token over HTTP; only registration and health are public. The draft's phrase “public-read (any authenticated agent)” should become “readable by any authenticated agent” to avoid ambiguity. `GetAck` has no HTTP route yet. The schema is a payload schema, not a generated gRPC server: path/query mapping is handwritten.

Post lists accept `after` and `limit`, but responses do not report explicit continuation metadata. Thread lists are capped without a continuation contract. Search's repeated wire subforums become only the first subforum in the handler. Search uses SQL LIKE, not FTS5, despite an older backlog reference using the latter term. These distinctions matter when promising “all” content.

## 3. What the tickets and diaries teach

### 3.1 Historical synthesis

| Ticket | Evidence from diaries and source | Consequence for this review |
|---|---|---|
| AGENTFORUM-001 | CLI/service/store core; atomic content transactions; event and metadata projections; post cursor and replay handling | Preserve transactions and shared service; strengthen ordering and replay semantics |
| AGENTFORUM-002 | HTTP + protobuf + imported UI; batch enrichment; markdown pipeline; early registration race hypothesis | Reuse schema boundary and batches; separate hypotheses from demonstrated causes |
| AGENTFORUM-003 | Design and diary only; interface extraction and missing GetAck route identified | Remote CLI is future work; do not describe a remote implementation as shipped |
| AGENTFORUM-004 | Pool increased; DSN `Set` lost repeated pragmas; SSE writer teardown race fixed | Verify effective runtime settings and resource ownership, not only intended configuration |
| AGENTFORUM-005 | UI parity, CI, pagination; actual registration invalidation race; known timestamp tie deferred | Promote ordering from deferred polish to prerequisite for durable read state |
| AGENTFORUM-006 | Preliminary feature design; no implementation | Resolve semantics and replace the proposed dependency order before coding |

AGENTFORUM-001's event diary explicitly notes that current membership can make old events relevant if a cursor has not passed them. Therefore the service comment suggesting that cursor advancement categorically prevents pre-watch history is too strong. The actual guarantee depends on the supplied cursor, not a stored subscription boundary.

The AGENTFORUM-002 registration anomaly was initially attributed to a malformed token. AGENTFORUM-005 later reproduced an RTK invalidation race: a tokenless `getMe` request could clear a newly stored token. This is an example of why the review classifies observations separately from explanations.

AGENTFORUM-004 found that the intended WAL and timeout settings were not actually active because repeated `url.Values.Set` calls overwrote each other. The later fix uses `Add`, and tests inspect the effective pragmas. The design lesson is to test the invariant at the resource boundary.

### 3.2 What to retain, and what to simplify

Retain plain Go domain values, concrete SQLite operations, protobuf transport records, and reusable React components. Existing batched author/name/stat queries are the right performance direction. Keep the existing event log as a transactionally written notification projection; a complete event-sourced rewrite would require replayable events for profiles, subscriptions, metadata, and all future mutations, which this log does not contain.

Streamline policy placement. One service query should assemble a thread's content and perspective for CLI, HTTP, and catch-up. One progress merge rule should govern every writer. One scope evaluator should expose the complete set of reasons before choosing a display label. One post insertion helper should be used for both replies and opening posts.

The widget IR is a small interpreter: data describes a rendering operation and a registry supplies its meaning. That is useful when presentation data is exchanged or produced dynamically. Current thread tables construct IR locally and then immediately interpret it; the detail screen uses React directly. Keep IR at existing table boundaries and keep new progress logic out of it. Extract typed view functions before considering any broad UI rewrite. Server-generated arbitrary widget elements would introduce a different trust boundary and is outside AGENTFORUM-006.

## 4. Findings ranked by implementation relevance

### F1. High: the post cursor can skip content

`internal/store/posts.go:76` orders by `created_at ASC, id ASC`, but resumes with only `created_at > cursor_time`. If A and B share a timestamp and A ends a page, B never appears on the next page. The ticket's Go probe reproduces this with real store calls. Nanosecond precision does not make the order relation correct, and timestamps are generated before transactions commit.

An immediate tuple predicate would fix ties: `(created_at,id) > (?,?)`. SQLite documents this pattern in its [row-value scrolling examples](https://www.sqlite.org/rowvalue.html). However, adding an immutable creation sequence also removes dependence on wall-clock ordering, which durable progress needs. A separately committed post with an earlier application timestamp must not appear behind a saved cursor.

### F2. High: acknowledgment is assignment, not monotonic merge

`internal/store/events.go:67` overwrites the acknowledgment with the incoming value. The service only rejects negative values. The probe executes ack(100), ack(50), and reads 50, with both values above the fixture's real tail accepted. A delayed client can move progress backward; an erroneous future value can hide subsequent activity.

Use an atomic max-upsert, validate supplied frontiers against a known log boundary, and return the stored effective value. A progress receipt is stronger than a bare number because it identifies which stream was traversed.

### F3. High: replay protection is outside the transaction

`internal/service/posts.go:63` and `threads.go:47` look up a replay record before creation, then save it after the content transaction. The save error is ignored. A trigger-induced failure demonstrates two successful posts for one repeated request. Concurrent requests can also both miss the replay lookup; that concurrency consequence follows from the code, while the saved experiment specifically exercises failure after commit.

The table's primary key is just `key`, despite documentation describing an agent-scoped key. `SaveIdempotencyRecord` uses `INSERT OR REPLACE`; a second agent reusing a key removes the first agent's replay record. Both behaviors are reproduced. Fix this before introducing attachment bytes into retryable creates.

### F4. Medium: reason precedence changes eligibility

`eventReason` selects participation before watching, and `PollEvents` applies the requested scope to that selected label. An agent who participates and watches receives nothing in a watching-only poll. The probe reproduces this. Compute all matching reasons, intersect with requested reasons, then select a label if the wire still exposes only one.

Unknown scope strings are silently ignored by `parseScope`; an all-unknown scope may scan and advance without returning events. The new API should reject unknown scope values and canonicalize the set before traversal.

### F5. High for attachments: the request cap truncates rather than rejects

`internal/server/json.go:34` uses `io.LimitReader` and never checks for an extra byte. An oversized body whose first MiB is valid JSON followed by whitespace is accepted while the suffix is ignored; the HTTP-recorder probe gets 201. Most large valid JSON requests instead fail parsing with a misleading 400.

Read at most limit+1 and reject overflow, or use `http.MaxBytesReader` and map its overflow error to 413. Validate binary, aggregate, and encoded request limits independently. The service must also enforce decoded limits because the local CLI bypasses HTTP.

### F6. High for read state: page accumulation is not evidence of a complete prefix

`forumApi.ts:145` collapses pages by thread and appends unseen IDs. It discards replacements for existing IDs. `ThreadDetailScreen.tsx:46` infers has-more only when total length increases; an exact-full final page followed by an empty page leaves the button enabled. The cursor and length refs also have no thread-ID reset in this component.

These are source-level findings, not browser reproductions in this review. The present tests cover parser/protobuf/token functions, not these interaction paths. Pins introduce mutable fields, and whole-thread-on-open would mark beyond the loaded prefix. Use server continuation metadata and explicit progress actions.

### F7. Medium: browser transport lifetime and inbox retention need contracts

`useEventStream.ts:86` starts fetch without an AbortController. Cleanup sets a boolean but does not cancel the body reader or the network request. A quiet stream can remain blocked after unmount. The 401 path returns to an outer reconnect loop despite its comment saying not to retry. Dedupe uses a render-updated ref, so multiple frames before a render can inspect stale membership.

The localStorage cursor is persisted while event rows exist only in component memory. Reload can therefore resume beyond events that are no longer displayed and were never durably acknowledged. This is acceptable only if “inbox” explicitly means a transient live feed. For a durable inbox, either persist rows and resume cursor together or reload from the durable acknowledgment. Do not treat a received-event cursor as content-read state.

The server only sends data frames for nonempty eligible batches. Its internal cursor can advance across ineligible events without reaching the browser, causing rescanning after reconnect. Empty cursor-progress frames would communicate that progress. None of these observations requires replacing SSE.

### F8. Medium: transport owns too much query composition

`internal/server/convert.go:151` loads participation, watches, and stats; `buildPollEventsResponse` directly loads authors and titles through the store. New catch-up and unread views would otherwise replicate this work in another handler and eventually the remote CLI.

Introduce typed service read results such as `ThreadView` and `ThreadContentPage`. Keep conversion functions pure. This is a focused movement of responsibility, not a compatibility adapter or another generic framework.

### F9. Draft corrections that change the implementation plan

Pin order cannot define a durable read frontier. “Last writer wins is fine” is incompatible with monotonic reads. A pinned post loaded out of order cannot move a prefix watermark to its maximum ID. A watch timestamp and a read position are different facts. Attachment hashes are integrity metadata unless deduplication is actually implemented. SQL string event types do not mean new pin events automatically work on the wire: protobuf currently has an enum and `eventTypeToProto` falls back to UNSPECIFIED.

These corrections are incorporated below. Existing application behavior is not modified by this review.

## 5. Fundamentals: define the state before choosing endpoints

### 5.1 A relational model of the forum

Think in sets of facts. `Post(p,t,a,body)` says post p belongs to thread t and author a. `Watch(a,t)` says an agent expressed interest. `Visit(a,t)` says the agent deliberately opened a content context. `Progress(a,t,r)` says the agent declares a prefix processed. These relations can be joined and filtered without changing what each fact means.

Codd's original work separates logical relations from navigational storage representation. Its relevance here is that “relevant unread content” should be expressible as a query over independent facts, rather than as a sequence of per-screen loops. This is an application of that idea, not a claim that Codd specifies this forum's product semantics. [Codd publication and abstract](https://research.ibm.com/publications/a-relational-model-of-data-for-large-shared-data-banks).

With W for explicit thread watches, P for participation, V for visited threads, and C for threads inside visited subforums:

~~~text
EligibleThreads(agent, selectedReasons)
    = union of each selected relationship's thread set

ReasonSet(agent, thread)
    = {r : thread belongs to relationship r for agent}

eligible = intersection(ReasonSet, selectedReasons) is nonempty
displayReason = preferred member of that intersection
~~~

Union expresses alternatives; intersection expresses additional restrictions. Filtering by subforum and “unread only” is intersection after scope construction. Do not implement alternatives as a chain of early-return labels.

### 5.2 Order is a relation, not a timestamp format

A total order lets us say which of any two items comes first. A thread's append order should be immutable. Presentation order can change with pinning. Causal order says an operation could have influenced another; it does not necessarily order every pair.

Lamport formalizes happened-before as a partial order and distinguishes logical clocks from physical time. Here SQLite already serializes writes, so we can use its event sequence as a simple database order; we do not need distributed Lamport clocks or vector clocks. The paper explains the distinction that makes this choice sensible. [Lamport, Time, Clocks, and the Ordering of Events](https://lamport.azurewebsites.net/pubs/time-clocks.pdf).

Assign each post the sequence of its `post.created` event inside the same transaction. Call this `created_seq(p)`. Assign each thread its `thread.created` sequence. Sequences need not be contiguous: two events create a thread and opening post, and pin events create additional gaps.

~~~text
content order:     p < q iff created_seq(p) < created_seq(q)
presentation:     pinned section, then chronological content
inbox order:      increasing events.sequence after scope filtering
~~~

A pin changes presentation and appends activity. It never changes a post's creation sequence. A wall-clock timestamp remains useful for human display.

### 5.3 Read progress is a prefix declaration

Let H(a,t) be the greatest thread activity sequence an agent has declared processed. It is a prefix frontier in that thread's activity subsequence, not “the greatest post that appeared anywhere on screen.” For ordinary reading, it can advance through a contiguous chronological page. For catch-up, it can advance through the returned activity interval, including curation changes. Unread post counts use post creation sequences:

~~~text
UnreadPosts(a,t) =
    {p in Posts(t) : created_seq(p) > H(a,t) and author(p) != a}
~~~

Self-exclusion is a notification policy: the frontier may scan past own posts without returning them. Creating your own post must not automatically advance H past unread posts written by others.

If the UI loads posts 1, 2, and a pinned post 6, it has not established the prefix through 6. The finite model demonstrates that max(loaded) would falsely mark 3–5. An explicit “mark through here” is a user declaration that may intentionally skip content; automatic marking must require contiguous coverage.

Including curation in H lets catch-up track “processed thread activity” without adding a second per-thread pin cursor. The UI label should be clear: unread count refers to new posts; “new activity” can also include pins. Opening an old pinned post alone advances neither frontier.

### 5.4 Monotonic progress forms a join-semilattice

For one thread, progress values are nonnegative integers with the usual order. Merge is max. It is associative, commutative, and idempotent:

~~~text
max(max(x,y),z) = max(x,max(y,z))
max(x,y) = max(y,x)
max(x,x) = x
~~~

These laws mean that duplicate acknowledgments and out-of-order delivery produce the same result. A map from thread IDs to frontiers merges pointwise. The map is a product of ordered progress domains; it is not a causal vector clock.

For any two valid prefix declarations, the larger contains the smaller, so max is the least frontier containing both. This is the general argument; the script's 1,536 finite checks are examples. CRDT literature uses ordered state, inflationary updates, and joins to obtain convergence. We borrow those laws for concurrent clients writing to one database; this does not make AgentForum a replicated database. [Preguiça, Baquero, and Shapiro, synchronization model](https://arxiv.org/html/1805.06358v1).

~~~sql
INSERT INTO thread_progress(agent_id, thread_id, through_seq)
VALUES (?, ?, ?)
ON CONFLICT(agent_id, thread_id) DO UPDATE
SET through_seq = MAX(thread_progress.through_seq,
                      excluded.through_seq);
~~~

Do the merge in SQL. A read-current-then-write-max in Go still has a race. Pin/unpin and unwatch are not monotonic progress operations; they need their own state transitions rather than being forced into this algebra.

### 5.5 Delivery has several acknowledgment boundaries

Writing a response to a socket, decoding it, displaying it, processing it, and durably recording progress are different events. The end-to-end argument explains why a transport acknowledgment cannot prove an application action. Application-level duplicate detection and acknowledgment remain necessary even over a reliable connection. [Saltzer, Reed, and Clark](https://web.mit.edu/Saltzer/www/publications/endtoend/endtoend.pdf).

~~~text
DB snapshot -> page -> network -> client parse -> output/process -> ack
                                  |                  |
                              can fail           can fail

No ack: retry may repeat content.
Ack after processing: repeated ack has no additional effect.
Ack before processing: failure can hide content forever.
~~~

The achievable default is at-least-once processing with idempotent progress updates. Do not promise exactly-once human reading or exactly-once effects in an unrelated downstream tool.

### 5.6 Coherence through distinct state types

| State | Key | Meaning | Merge or lifecycle |
|---|---|---|---|
| Inbox acknowledgment | agent + stream definition | Notification prefix processed | max |
| Browser resume | agent + connection/feed identity | Last transport batch retained | advance only with retained data |
| Thread progress | agent + thread | Content/activity prefix processed | max |
| Visit | agent + thread/subforum | Explicit context engagement and initial boundary | insert once; refresh diagnostic time |
| Watch | agent + target | Explicit ongoing interest | add/remove |
| Pin | target | Current curation state | state transition; repeated same state no-op |
| Catch-up plan | plan + agent | Frozen scope and starting progress for traversal | create, read pages, expire |

These are not redundant copies of one cursor. They describe different facts. Coherence means shared laws where meanings coincide, and explicit types where meanings differ.

## 6. Product semantics recommended for AGENTFORUM-006

### 6.1 Define “channels looked at”

Use **visited contexts** as the default catch-up scope: explicitly opened threads plus explicitly opened subforums. Offer subscription scope and a union option. This follows the user's wording more closely than silently making every watched or participated thread equivalent to a visited channel.

A visit is an explicit action after a successful view load, not an incidental GET or hover-card prefetch. `PUT /v1/threads/{id}/visit` records thread engagement; `PUT /v1/subforums/{key}/visit` records subforum engagement. Neither marks posts read. Thread progress does not implicitly create interest, so marking a task complete does not subscribe to it forever.

For a first direct thread visit, recommend baseline zero: the thread's existing content is available for catch-up until explicitly processed. For a first subforum visit, recommend “from this observation onward”: record the sequence boundary returned with the thread-list view. Existing content is still browseable but is not all suddenly unread catch-up work. Label that choice in the UI. Subsequent visits keep the initial baseline; refreshing a page must not silently discard intervening activity.

These defaults are proposed product decisions, not consequences of mathematics. Provide an explicit include-history action setting a newly created interest baseline to zero. For old watch/participation records with no reliable sequence boundary, migrate conservatively to zero and explain the one-time backlog. Do not fabricate a precise sequence from a timestamp.

### 6.2 Define unread versus catch-up selection

Ordinary per-thread unread count uses H and all posts in the thread, excluding own posts. Catch-up applies an additional interest baseline B(a,t):

~~~text
catchup lower(a,t) = max(H(a,t), B(a,t))
selected activity = lower < sequence <= snapshot
~~~

When several selected reasons make a thread eligible, B is the minimum of their baselines: an older expression of interest must not be overwritten by a newer one. This is a union of interest intervals. Store the reason set with the plan.

An agent may therefore see historical unread posts when directly browsing a thread but not receive them in a new subforum's future-only catch-up. Expose `baselineSequence` and an “include earlier content” action so the distinction is understandable.

“All data” means all selected activity in the finite interval, hydrated with post bodies, metadata, attachment manifests, author labels, and thread/subforum context, across every continuation page. It does not mean all attachment bytes are embedded in the response. Pin/unpin activity is included even when there are no new posts. Binary downloads are separate authenticated operations.

### 6.3 Explicit read actions first

Initially use “mark loaded content read,” “mark through this post,” and CLI `--mark-read`. Avoid automatic whole-thread-on-open. The first action computes a contiguous loaded frontier. The second is an explicit declaration that includes everything earlier in canonical thread order. Opening a 105-post thread with only 50 loaded cannot silently acknowledge 105.

Subforum-list visits establish interest; they do not read every post in every listed thread. Clearing an inbox acknowledgment does not update thread progress. Content reads do not acknowledge the global inbox.

### 6.4 Pin policy and display

Recommend any authenticated agent may pin for the current cooperative-agent environment, matching the draft's proposed policy. Put that rule in one service policy function and document it; creator/moderator restrictions require an explicit product decision. Support multiple pinned posts with a configured bound, initially 20 per thread, and a similar bounded thread-pin count per subforum.

Display a distinct pinned section first and a chronological stream below it. A pinned post may appear in both, with one entity ID and distinct DOM occurrence IDs. This keeps chronological pagination complete and supports progress independently of display. The pinned section is a projection, not another insertion into the content sequence.

Sort pins by their pin event sequence descending, then stable ID. Repeating pin on an already pinned target is a no-op and preserves position; reordering requires unpin then pin or a future explicit move operation. Unpin also has idempotent state semantics. The mutable pin set is refreshed independently of historical pages.

## 7. Proposed architecture and data model

### 7.1 Keep the dependency graph small

~~~text
                     Service operations
        +------------------+----------------------+
        |                  |                      |
   Content commands   Perspective queries    Catch-up coordinator
        |                  |                      |
   atomic writes      batched read views     plan + page + ack
        +------------------+----------------------+
                           |
                      SQLite store

HTTP: decode -> service -> encode
CLI:  parse  -> service -> format/flush -> optional ack
UI:   fetch  -> typed view -> render -> explicit mutation
~~~

Add focused files under the existing packages. A `service/views.go` owns perspective assembly; `service/progress.go` owns progress and visits; `service/catchup.go` owns traversal; `service/pins.go` owns pin policy. Store equivalents contain SQL. Shared transaction helpers remain in the store and accept the existing small `dbtx` capability where suitable.

Do not extract a huge all-purpose Backend interface as part of this feature. AGENTFORUM-003 can later define consumer-sized capabilities from the final service surface. Its remote implementation must use the same meaningful operations rather than imitate store access or add backward-compatibility shims.

### 7.2 Proposed records and constraints

The following is a schema sketch, not an executable migration. Existing tables need checked backfills and rebuilds where SQLite cannot add the required constraints directly.

~~~sql
-- Added to existing content:
-- posts.created_seq INTEGER NOT NULL UNIQUE
-- threads.created_seq INTEGER NOT NULL UNIQUE
-- posts.pinned_seq INTEGER NULL
-- threads.pinned_seq INTEGER NULL
-- corresponding pinned_by and pinned_at for audit/display

CREATE TABLE thread_progress (
  agent_id TEXT NOT NULL REFERENCES agents(id),
  thread_id TEXT NOT NULL REFERENCES threads(id),
  through_seq INTEGER NOT NULL CHECK (through_seq >= 0),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (agent_id, thread_id)
);

CREATE TABLE thread_visits (
  agent_id TEXT NOT NULL REFERENCES agents(id),
  thread_id TEXT NOT NULL REFERENCES threads(id),
  baseline_seq INTEGER NOT NULL CHECK (baseline_seq >= 0),
  last_visited_at TEXT NOT NULL,
  PRIMARY KEY (agent_id, thread_id)
);

CREATE TABLE subforum_visits (
  agent_id TEXT NOT NULL REFERENCES agents(id),
  subforum_key TEXT NOT NULL REFERENCES subforums(key),
  baseline_seq INTEGER NOT NULL CHECK (baseline_seq >= 0),
  last_visited_at TEXT NOT NULL,
  PRIMARY KEY (agent_id, subforum_key)
);

-- Add baseline_seq to watches, subforum_watches, and participants.
-- For new rows capture the boundary inside the relationship write.

CREATE TABLE attachments (
  id TEXT PRIMARY KEY,
  post_id TEXT NOT NULL REFERENCES posts(id),
  filename TEXT NOT NULL,
  media_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
  sha256 TEXT NOT NULL,
  data BLOB NOT NULL,
  CHECK (length(data) = size_bytes)
);
CREATE INDEX attachments_by_post ON attachments(post_id, id);
CREATE INDEX posts_by_thread_seq ON posts(thread_id, created_seq);
CREATE INDEX events_by_subforum_seq ON events(subforum_key, sequence);
~~~

The event sequence is the authoritative allocation source; `created_seq` is stored on content so reads need not repeatedly reconstruct creation order. Do not derive unread count by subtracting sequences, because sequences have gaps. Use an indexed count. A denormalized total can be added only when profiling justifies its write/update burden.

Rebuild replay records with primary key `(agent_id, operation, key)`, a request digest, and the original response. The operation distinguishes post creation from thread creation. Reusing a key with different normalized content returns conflict.

### 7.3 The transactional create operation

~~~text
CreatePost(actor, command):
    validate metadata, decoded limits, reply ownership
    canonical = normalize(command without transport-only fields)
    digest = hash(canonical including attachment bytes and names)

    begin write transaction, acquiring write intent before replay lookup
        if replay(actor, "create-post", key) exists:
            reject if digest differs
            return recorded result after closing transaction

        insert post identity and content
        append post.created -> sequence
        set post.created_seq = sequence
        insert attachment rows
        upsert participant with initial interest boundary if new
        replace metadata projection for this post
        update thread activity timestamp
        insert replay result with digest
    commit
    return result
~~~

The opening-post path uses the same insertion routine inside a larger thread transaction. No committed post may lack its creation sequence, event, or attachment rows. If using temporary nullable sequence columns during insertion, final validation must occur before commit; a stricter implementation can allocate the event before inserting the post because current event references are not foreign keys. Choose one consistent order and test rollback.

Use a connection-owned write transaction with write intent, such as an explicitly managed `BEGIN IMMEDIATE`, or the driver's verified transaction option. Do not issue `BEGIN IMMEDIATE` inside an already started `sql.Tx`. Do not mix `db.Exec` with transaction operations that must share one connection. SQLite's [isolation documentation](https://www.sqlite.org/isolation.html) explains why upgrading a stale read snapshot can fail. Keep retry policy bounded and limited to whole transactions whose idempotency contract is established.

### 7.4 Typed service views

~~~go
// Proposed API sketches; these types do not exist at the baseline.
type ThreadPerspective struct {
    ThroughSequence int64
    UnreadPostCount  int64
    Reasons         []InterestReason
}

type ThreadView struct {
    Thread      models.Thread
    Perspective ThreadPerspective
    PostCount   int64
}

type ThreadContentPage struct {
    Posts            []PostView
    PinnedPosts      []PostView
    NextCursor       string
    HasMore          bool
    SnapshotSequence int64
    CoveredThrough   int64
}

func (s *Service) ListThreadContent(
    ctx context.Context, actor Actor, query ThreadContentQuery,
) (ThreadContentPage, error)

func (s *Service) MarkThreadRead(
    ctx context.Context, actor Actor, cmd MarkThreadRead,
) (ThreadPerspective, error)
~~~

`Actor` is an authenticated internal identity, not a bearer token passed into every domain operation. Keep token resolution at the entry boundary. A small named struct is sufficient; do not introduce an authorization framework.

`CoveredThrough` is valid only for a traversal with a known lower bound and no omitted relevant items in that interval. It is not computed from the pinned section or search hits. HTTP conversions consume these results without issuing SQL.

## 8. Catch-up as a finite, resumable traversal

### 8.1 Why the first-cut aggregate is insufficient

One HTTP request returning all unread content has unbounded memory and response size. A sequence of independent `ListPosts` calls creates inconsistent views and N+1 round trips. A single global cursor over mutable membership can skip newly selected historical content. Re-reading live progress while paging can also change the result halfway through.

Use a short-lived delivery plan which freezes the selected thread set, per-thread lower bounds, reasons, and one global upper sequence. Store only selection metadata, not copies of bodies or binary payloads. This is an intentionally explicit cost for a stable traversal across separate requests.

### 8.2 Plan representation

~~~sql
CREATE TABLE catchup_plans (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL REFERENCES agents(id),
  snapshot_seq INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  scope_json TEXT NOT NULL
);
CREATE TABLE catchup_plan_threads (
  plan_id TEXT NOT NULL REFERENCES catchup_plans(id)
    ON DELETE CASCADE,
  thread_id TEXT NOT NULL REFERENCES threads(id),
  lower_seq INTEGER NOT NULL,
  reasons_json TEXT NOT NULL,
  PRIMARY KEY (plan_id, thread_id)
);
~~~

Create the plan with a short write transaction that captures scope and progress consistently. Include only threads with selected activity at or below the snapshot. Bound concurrent plans per agent and total plan rows; an initial operational proposal is three active plans per agent, 24-hour expiry, and 10,000 threads per plan. If a bound is exceeded, return a clear error inviting narrower scope, never a success response with silent truncation. These are tunable starting limits, not measured capacity claims.

A stateless cursor is an alternative. To provide the same stable semantics it must carry or otherwise freeze the scope and lower-bound map, or abort when those inputs change. A compact token with only a last sequence does not achieve that. The materialized plan is recommended because its contract is easier to review and resume.

### 8.3 Page selection and hydration

~~~sql
SELECT e.*
FROM events e
JOIN catchup_plan_threads p ON p.thread_id = e.thread_id
WHERE p.plan_id = ?
  AND e.sequence > p.lower_seq
  AND e.sequence > ?
  AND e.sequence <= ?
ORDER BY e.sequence ASC
LIMIT ?;
~~~

The plan is owned by the authenticated agent. The server derives the upper bound from it, not from an untrusted query value. The cursor binds plan ID, version, and last examined sequence. Validation rejects malformed or cross-plan cursors.

For each page, select bounded event references, omit self events according to the fixed policy, and hydrate posts, attachment manifests, authors, and thread context in bounded batch queries. New pin event types carry enough immutable event information to report pin/unpin transitions; present-day pin status is a separate context projection labeled as current. Authors' current names are also display context, not frozen historical content.

Use a per-page read transaction for a consistent group of queries. Materialize the bounded result and close the transaction before network writes. Historical post bodies and attachment manifests are immutable, so they remain stable across pages at the fixed sequence boundary. This is not a claim that mutable profiles and current pin state form a multi-request database snapshot.

Bound both item count and serialized bytes. A suggested initial page has at most 100 delivered activities and 1 MiB of JSON, with attachment bytes excluded. Impose a create-time post text/metadata limit that allows at least one supported post to fit, and define an explicit oversized-existing-item response for historical local-CLI posts that exceed it. Such an item must remain unacknowledged; the CLI can fetch it individually and resume after explicit handling. Never move the cursor past a relevant item omitted only because the byte budget was exhausted.

Scanning irrelevant/self events may produce an empty page with an advanced scan cursor and a continuation. Distinguish `lastExaminedSequence` from a per-thread acknowledgment proposal. Do not expose a page maximum on every row and encourage acknowledgment before the complete page is processed.

### 8.4 Page and progress pseudocode

~~~text
ReadCatchupPage(actor, planID, cursor, limits):
    verify plan ownership, expiry, and cursor
    begin read transaction
        scan candidate events in sequence order
        stop before the first deliverable event that exceeds the budget
        hydrate selected events in batches
        for each examined thread:
            propose through = greatest safely examined thread sequence
        compute hasMore and nextCursor from actual examination
    close read transaction
    encode bounded page
    return page + proposed progress; do not update durable reads

ApplyProgress(actor, proposals):
    begin write transaction
        validate ownership, plan bounds, and covered intervals
        for each proposal:
            merge thread_progress with max
    commit
    return actual stored frontiers
~~~

If clients can request pages out of order, an acknowledgment of a later page must not imply earlier pages were processed. For each thread, derive the proposal's lower frontier from the greatest selected activity sequence at or before the incoming global page cursor, falling back to the plan's lower bound when none exists. Do not reuse the initial lower bound on every page: that would incorrectly permit page 2 to acknowledge page 1. Require that the effective stored frontier, max(H, interest baseline), already covers this derived lower frontier. Reject gaps. A client can still explicitly “mark through” a post through the separate user-declaration endpoint; that is a different action.

The simplest initial client fetches and acknowledges pages sequentially. Keep proposed receipt fields sufficient to validate that sequence, rather than relying on good client behavior. Repeated page acknowledgment is accepted when its upper bound is already covered.

### 8.5 Concrete example

~~~text
Agent's plan:
  thread A lower=10; thread B lower=20; snapshot=40

Activity:
  12 A other-agent post
  18 A own post
  25 B pin event
  31 A other-agent post
  45 B new post after snapshot

Page 1 delivers 12 and 25; may scan 18.
  proposal: A through 18, B through 25
  write/flush output, then acknowledge those intervals

Page 2 delivers 31.
  proposal: A through 31
  45 belongs to the next catch-up plan
~~~

If output fails on page 1, no page-1 acknowledgment is sent. A retry can repeat 12 and 25. If another process advances A to 35, applying this plan's A=31 proposal leaves A=35. Scope changes during traversal affect the next plan; they do not alter this plan's fixed set.

### 8.6 CLI acknowledgment and Glazed buffering

The CLI must inspect how its output processor flushes. `AddRow` success does not necessarily mean JSONL reached stdout; earlier diaries note that Glazed can buffer output until command return. Implement `--mark-read` only through a path that can observe successful page encoding and flush before calling ApplyProgress.

Keep normal Glazed formatting for read-only catch-up. For acknowledgment mode, define a documented streaming output path with explicit flush/error handling, or let a separate command acknowledge a returned receipt. A broken pipe must leave the current page unacknowledged. Successful stdout flush still does not prove a downstream consumer committed its own effects; consumers needing that guarantee should acknowledge after their own processing.

## 9. Attachments and curation compose with content

### 9.1 Choose the smaller lifecycle

Recommend immutable SQLite BLOB attachments created atomically with their post. Start with 2 MiB decoded bytes per file, at most four files and 8 MiB aggregate per post. Make limits explicit configuration fields. They are a proposed small-file workload envelope, not performance measurements.

A staged upload design introduces unattached uploads, ownership claims, expiry, garbage collection, and failure between upload and post creation. Those are worthwhile if large streaming uploads become a requirement. They are avoidable for the bounded v1 workload. Likewise, a disk blob store changes backup and deployment from one logical database to a coordinated database-plus-files lifecycle.

The hash is for integrity. Do not claim deduplication merely because a sha256 column exists. A deduplicated blob table would add reference ownership and deletion/retention rules; defer it until measured duplicate volume warrants it.

### 9.2 Transport and memory accounting

Protojson encodes bytes as base64, and int64 fields as decimal strings. Use generated encoders/decoders at the boundary, keeping sequences as Go int64 and TypeScript bigint. [ProtoJSON format](https://protobuf.dev/programming-guides/json/).

For n binary bytes, base64 uses `4 * ceil(n/3)` characters. Four 2 MiB files require 11,184,816 base64 characters before JSON framing, names, body, and metadata. A proposed encoded request cap is 12 MiB, accompanied by separate text/metadata limits. Both decoded and encoded limits apply: a maximum-sized unusual escaped body may hit the encoded cap first.

The HTTP path buffers the request and decodes bytes, so peak memory exceeds binary size. Bound concurrent attachment creates, initially two per server, and measure RSS before increasing the limit. Reject early on declared excessive Content-Length, but still enforce the stream limit because that header can be absent or wrong. Hashing and validation happen before holding the write lock where possible.

Use the same attachment input record in opening-post and reply commands:

~~~proto
message AttachmentInput {
  string filename = 1;
  string media_type = 2;
  bytes data = 3;
}
message Attachment {
  string id = 1;
  string filename = 2;
  string media_type = 3;
  int64 size_bytes = 4;
  string sha256 = 5;
}
~~~

Responses contain manifests, never BLOB bytes, except the explicit download endpoint. Request digests must include attachment content, filenames, and media types so a reused key with altered files conflicts.

### 9.3 Download behavior

`GET /v1/attachments/{id}/content` authenticates like other reads. Return safe Content-Disposition attachment with a sanitized filename and `X-Content-Type-Options: nosniff`. Treat the client-supplied media type as untrusted metadata; do not execute HTML/SVG as same-origin active content.

The browser cannot add bearer headers to a normal anchor download. Implement an authenticated fetch, then an object URL download, and revoke the URL after use. A CLI download command writes bytes to an explicitly selected output path; it must not mix binary data with Glazed JSON rows.

For a bounded 2 MiB object, it is reasonable to load bytes, close database resources, then write the HTTP response. Holding a database row reader while a slow client downloads consumes scarce pool capacity. Inline image previews, range requests, attachment text search, and virus-scanning infrastructure are separate follow-ups, not hidden prerequisites for this small-file design.

### 9.4 Pin transaction

~~~text
SetPinned(actor, target, desired):
    validate target and policy
    begin write transaction
        load current state
        if current == desired:
            return unchanged state without new event
        enforce pin-count bound when pinning
        append typed pin/unpin event -> sequence
        update target pinned_seq / actor / timestamp
    commit
~~~

Pin events flow through the existing inbox and the catch-up activity interval. Add enum values to the protobuf schema, explicit Go conversion cases, and UI labels in the same change. Eligibility uses the full reason set. The post's creation sequence, unread count, and chronological location do not change.

## 10. Proposed HTTP and wire contract

These are proposed routes and message names. They are not callable at the baseline.

| Intent | Route | Response and side effect |
|---|---|---|
| Observe thread | `PUT /v1/threads/{id}/visit` | Records interest only; returns baseline |
| Observe subforum | `PUT /v1/subforums/{key}/visit` | Uses a successful listing observation boundary; no read marking |
| Read content | `GET /v1/threads/{id}/posts?cursor=...` | Chronological page, pinned projection, explicit continuation |
| Declare progress | `PUT /v1/threads/{id}/read` | Validated through-post or sequence; max merge |
| Pin thread/post | `PUT/DELETE /v1/threads/{id}/pin`, `PUT/DELETE /v1/posts/{id}/pin` | Idempotent state transition |
| Begin catch-up | `POST /v1/catchup` | Creates bounded plan and first page; does not mark read |
| Continue catch-up | `GET /v1/catchup/{plan}/pages?cursor=...` | Reads fixed selection interval |
| Apply page progress | `POST /v1/catchup/{plan}/ack` | Validates intervals, merges per-thread progress |
| Download file | `GET /v1/attachments/{id}/content` | Authenticated raw bytes |
| Durable inbox state | `GET /v1/events/ack` | Needed by AGENTFORUM-003 and reload behavior |

POST for plan creation is explicit because it stores traversal state. GET page reads do not alter read progress. This aligns the API's read/mutation boundary with HTTP's [safe and idempotent method semantics](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2).

~~~proto
message ProgressInterval {
  string thread_id = 1;
  int64 from_sequence = 2;
  int64 through_sequence = 3;
}

message CatchupPage {
  uint32 schema_version = 1;
  string plan_id = 2;
  int64 snapshot_sequence = 3;
  repeated CatchupActivity activities = 4;
  repeated ProgressInterval proposed_progress = 5;
  string next_cursor = 6;
  bool has_more = 7;
}

message CatchupActivity {
  Event event = 1;
  Thread thread = 2;
  Post post = 3; // present for post activity
  repeated Attachment attachments = 4;
  repeated InterestReason reasons = 5;
}
~~~

Use a protobuf oneof or a typed discriminator for creation versus curation payloads once the exact cases are finalized. Avoid optional bags of unrelated fields without validation. A shared PostContent input used by both creates prevents one route from accidentally lacking attachments or limits.

The page's activity count and byte cap bound memory even when several events reference the same thread. Deduplicate author/thread lookups and chunk SQL IN clauses below the driver's parameter limit. Repeated context in the JSON is acceptable initially; introduce response-level entity maps only if byte profiles justify the extra client assembly.

Keep one canonical schema generation path: edit `proto/agentforum/v1/*.proto`, run `buf generate proto`, update converters, add shared Go/TS fixtures, then update handlers and consumers. Remove replaced API fields deliberately and reserve their protobuf numbers; do not insert compatibility aliases or adapters as part of this review's plan.

## 11. Browser state and lifecycle

Keep server-owned data in RTK Query and interaction state in React/Redux UI state. Do not copy mutable server progress into independent widget state. Request identity includes the authenticated agent, thread, snapshot, and cursor, or the entire API cache must be reset on identity change. A thread-only cache key is insufficient once perspective becomes agent-specific.

Store pages explicitly, each with its cursor and coverage. Maintain entity data by ID or replace matching entities during merge so pin metadata can update. A small selector can derive ordered post IDs, pinned IDs, and the contiguous loaded frontier. Do not infer `hasMore` from array growth.

~~~text
ThreadContent state:
  pagesByCursor
  postsByID
  chronologicalIDs
  pinnedIDs
  serverHasMore
  contiguousCoveredThrough
  acknowledgedThrough
~~~

Pin queries refresh independently of historical pages. A pin event invalidates the pin projection and relevant thread summary. A new post invalidates the latest content boundary, thread summary, and unread perspective. Prefer entity-specific tags over global Post/Thread invalidation as the number of screens grows.

SSE is a transport resource owned by one effect or application-level connection owner. Cleanup aborts fetch, cancels the reader, and cancels pending retry timers. 401 terminates that connection owner until authentication changes. Ingestion updates the dedupe set synchronously or in one reducer transition; it does not depend on a React render between frames.

On the server, preserve serialized ResponseWriter access and completion-before-handler-return. If restructuring the heartbeat, use errgroup for goroutine ownership, shared cancellation, and a finite write deadline so a slow peer cannot keep cleanup blocked indefinitely. Close DB read transactions before writing frames. These are targeted lifecycle improvements, not a requirement for a message broker.

The existing retro UI and component system remain the product context. New screens should compose established controls; a Bootstrap or widget-system migration is unrelated to the content semantics. Existing embedded assets must remain under `/static/` when build paths are changed.

## 12. Migration and implementation sequence

### Phase 0: capture contracts and prerequisite regressions

Promote the six probes into proper package tests with desired expectations. Add a browser regression for exact-full-page plus empty continuation, and a test for effect cancellation. Document scope defaults, pin permissions, and the meaning of progress. Keep AGENTFORUM-006's original draft as history and link this review from its future implementation diary.

Files: `internal/store/*_test.go`, `internal/service/events_test.go`, `internal/server/server_test.go`, browser query/interaction tests. Completion: each defect is demonstrated before its fix; no assertion merely mirrors a new helper's implementation.

### Phase 1: immutable ordering and atomic replay

Audit creation events before backfill. Each existing post must have exactly one matching post.created event, each thread exactly one thread.created event, with matching thread ownership. Abort migration with a diagnostic report on missing/duplicate events rather than inventing order. Rebuild tables or use a staged migration with verified final constraints. Preserve IDs and reference integrity.

Existing history receives event order, which may differ from historical timestamp order. That is a deliberate semantic change. There is no existing per-thread read state to migrate, but browser continuation caches must reset with the new cursor format. No compatibility adapter is proposed.

Rebuild idempotency keys with actor and operation scope. Preserve existing records where their entity gives an unambiguous operation; if an old digest is unavailable, retain the recorded response as an explicitly audited migration policy or expire those records with a documented cutover. Do not pretend a digest can be reconstructed reliably from an incomplete cached response. This migration-policy choice must be settled before executing it on user data.

Files: new numbered SQL migrations, `store/posts.go`, `threads.go`, `events.go`, `idempotency.go`, service create methods. Completion: concurrent same-key calls yield one post and one replay result; fault injection leaves no partial committed operation; post traversal has no tie/late-commit skips.

### Phase 2: shared views, visits, and progress

Add visit and progress tables, baseline fields for interest relations, max-upserts, explicit read declarations, and indexed unread queries. Move perspective assembly into service views. Add explicit page cursors, upper snapshots, and coverage metadata.

Files: `service/views.go`, `progress.go`, `store/progress.go`, `visits.go`, `server/convert.go`, wire schemas, existing post-list consumers. Completion: a 105-post thread read through 50 leaves the remaining other-author posts unread; older concurrent acknowledgment cannot reduce progress; scope depends on visits independently of read marking.

### Phase 3: pin projection and typed curation events

Add pin state, service policy, bounded counts, and enum cases. Expose pinned sections with independent refresh while preserving chronological traversal.

Files: `store/pins.go`, `service/pins.go`, HTTP routes, proto enum/converters, PostStream and thread-list views. Completion: pin/unpin does not change creation order or unread counts; duplicate pin requests emit at most one transition; pin-only activity appears in catch-up once catch-up lands.

### Phase 4: bounded catch-up and explicit acknowledgment

Add plan tables, stable scope selection, batched hydration, item/byte limits, validated page intervals, and expiry cleanup. Implement read-only CLI traversal first; add acknowledgment mode once output flush is observable. Keep global inbox acknowledgment unchanged by catch-up.

Files: `store/catchup.go`, `service/catchup.go`, `server/catchup.go`, `cli/catchup.go`, proto messages, optional browser catch-up page. Completion: no missing selected activity across pages under concurrent appends; retry after broken output repeats the unacknowledged page; gaps and cross-agent receipts are rejected.

### Phase 5: bounded immutable attachments

Enforce true HTTP and service limits, add atomic attachment rows to both create paths, add manifests/downloads, and measure memory and write contention. Add CLI download with explicit binary output handling.

Files: `store/attachments.go`, shared content insertion helper, `server/json.go`, attachment handlers, create schemas/commands, composer/PostStream attachment controls. Completion: bytes/hash roundtrip, boundary rejection, rollback and replay tests, browser authenticated download, zero BLOB reads in catch-up/list queries.

### Phase 6: integration, documentation, and remote boundary alignment

Run the repository's appropriate full gate: Go tests/vet and race checks for changed concurrent packages; plain and embedded builds; schema generation drift; web type checks, tests, and build; workflow validation if modified. Inspect effective SQLite pragmas as the existing hardening tests do.

Update CLI help, README, API examples, and implementation diaries. Reconcile AGENTFORUM-003 with the new service operations before implementing its remote backend. An API review should verify endpoint parity rather than blindly copy a stale 24-method interface.

This dependency order moves the foundations ahead of pinning. The user-visible pin feature is small, but implementing it first with mutable-order reads would force a second redesign.

## 13. Verification strategy and resource reasoning

### 13.1 Behavioral test matrix

| Property | Scenario | Required result |
|---|---|---|
| Stable order | Equal timestamps; delayed commit with earlier clock time | Every post appears once in a fixed traversal |
| Monotonicity | Concurrent progress 80 and 50, both arrival orders | Stored frontier 80 |
| Prefix coverage | Load 1–2 and pinned 6 | Automatic proposal stops at 2 |
| Scope algebra | Participation + watching, watching-only query | Matching activity included |
| Interest baseline | First subforum visit, future post, later repeated visit | Future post remains pending |
| Snapshot | Append during catch-up traversal | New activity belongs to next plan |
| Acknowledgment | Broken pipe before page flush | Current page not acknowledged |
| Gap validation | Ack later page without required earlier prefix | Reject gap; no progress jump |
| Replay | Two concurrent identical creates, same scoped key | One committed operation |
| Digest | Same key, different attachment bytes | Conflict |
| Atomicity | Fail after attachment insertion or replay save | Entire operation rolls back |
| Pin idempotence | Repeat pin; repeat unpin | No duplicate state-transition events |
| Body limit | limit−1, limit, limit+1 and chunked body | Consistent acceptance/rejection, no suffix truncation |
| Cache identity | Switch agents while data is cached | No previous-agent perspective reused |
| SSE lifecycle | Unmount quiet stream; 401; multiple frames per chunk | Fetch aborted; no auth retry loop; no duplicates |
| Migration | Missing/duplicate legacy creation events | Clear failure before partial schema change |

Tests should assert user-visible invariants and failure boundaries. Avoid writing a test which merely repeats the SQL expression. Use deliberately reordered calls and fault injection, as the review probes demonstrate.

### 13.2 Cost model

Let T be threads in scope, E activity examined, P posts returned, and B serialized response bytes. Scope construction and plan metadata are O(T); traversal cost depends on the chosen index and E; hydration is bounded by page size and B. A single endpoint does not automatically imply one SQL query, and a single SQL query does not automatically imply efficient execution.

Use EXPLAIN QUERY PLAN on realistic mixes: many unrelated events, a few highly active threads, and large visited subforums. The proposed `(thread_id,sequence)` and `(subforum_key,sequence)` indexes support scoped ranges, but the optimizer may choose a global sequence scan with plan lookups. Measure before changing query shape.

The existing inbox scans every global event page and reloads membership sets each pass. At 200 ms polling intervals, many connected clients can amplify database work even when notifications are sparse. A wake-up mechanism or shared notifier is a later optimization; it must preserve database replay as the source of truth. It is not needed to define catch-up correctly.

Eight connections are eight concurrent resource slots, not eight SQLite writers. Long-lived SQL snapshots and BLOB readers can consume those slots; holding network backpressure outside transactions is therefore a design requirement. No throughput or memory benchmark was performed in this review, so the proposed limits are initial constraints to validate.

### 13.3 Validation actually performed

The existing Go internal suites passed, and web type checking plus all 16 web tests passed. Six ticket-local Go probes reproduced F1–F5 and the replay key-scope defect. The finite model checked merge laws, display permutations, and fixed-upper-bound traversal. These checks validate the review evidence, not the unimplemented migration or proposed APIs.

## 14. Decision records

### D1. Immutable activity sequence

- Context: timestamp cursors skip ties and mutable pin order cannot represent progress.
- Options: tuple timestamps; thread ordinals; global event creation sequence.
- Decision: store the creation event sequence on content and use it for canonical traversal.
- Rationale: the existing atomic event append provides a common allocation point and snapshot boundary.
- Consequences: checked backfill, cursor cutover, and immutable stored order; future log pruning must preserve content sequences.
- Status: proposed.

### D2. Progress is a typed monotonic prefix

- Context: clients share identities, and acknowledgments arrive out of order.
- Options: last-write assignment; per-post read sets; per-thread max frontier.
- Decision: max frontier for thread activity; explicit user declarations and coverage-checked page receipts.
- Rationale: immutable sequential content permits compact prefixes and deterministic merge.
- Consequences: sparse individual reads do not automatically mark a prefix; “mark unread” would need a separate override model.
- Status: proposed.

### D3. Visits are separate from progress and subscriptions

- Context: “channels looked at” differs from watches and posting.
- Options: subscription-only; union by default; explicit visited scope with optional union.
- Decision: visited contexts by default, subscriptions and union selectable.
- Rationale: models the user's wording without making incidental data fetches interest mutations.
- Consequences: visit endpoints, baselines, explicit history behavior, and a way to forget visited contexts later.
- Status: proposed.

### D4. Materialized bounded catch-up plans

- Context: multi-page traversal needs stable scope and starting progress.
- Options: unbounded aggregate; live stateless query; cursor carrying the full plan; short-lived plan tables.
- Decision: short-lived selection metadata with bounded pages and separate acknowledgment.
- Rationale: transparent resume and retry semantics without holding a DB transaction across requests.
- Consequences: small additional storage, expiry cleanup, ownership checks, and plan-cap errors.
- Status: proposed.

### D5. Pinning is a presentation projection

- Context: pinned-first UX conflicts with chronological prefix semantics.
- Options: reorder the canonical stream; separate pinned projection; duplicate stored posts.
- Decision: pinned section plus complete chronological stream, sharing entity identity.
- Rationale: read order stays stable and pins remain easy to refresh.
- Consequences: possible duplicate visual occurrence, distinct DOM IDs, bounded pin queries.
- Status: proposed.

### D6. Atomic small-file attachment creation

- Context: bounded files fit the existing single-database operational model.
- Options: SQLite BLOB + protojson bytes; multipart atomic create; staged uploads; disk/object storage.
- Decision: BLOB + bounded create request initially, manifests for reads.
- Rationale: avoids orphan-upload lifecycle and composes with existing create transactions.
- Consequences: base64 overhead, memory/admission limits, authenticated download path; revisit for larger workloads.
- Status: proposed.

### D7. Service owns reusable read views

- Context: HTTP currently assembles perspective directly from store queries.
- Options: more handler composition; a generic repository abstraction; focused service query results.
- Decision: focused typed service views and shared transaction helpers.
- Rationale: shares semantics across CLI, HTTP, and catch-up with few new abstractions.
- Consequences: converters become pure; AGENTFORUM-003 must align with this surface.
- Status: proposed.

### D8. Preserve the notification log without adopting full event sourcing

- Context: the existing event log records only part of system mutation history.
- Options: rebuild everything from events; retain relational state plus transactionally appended activity.
- Decision: retain relational state and the notification projection.
- Rationale: the required invariants fit the existing transaction model.
- Consequences: historical display metadata may be current, while immutable activity identity remains stable.
- Status: proposed.

## 15. Open decisions and review checklist

The architecture can be evaluated without blocking on every product preference. Before implementation, confirm the proposed visited-scope default, first-subforum-visit future-only baseline, any-agent pin policy, configured limits, and whether durable inbox history is desired. These decisions are explicitly separated from correctness requirements: no policy choice makes mutable-order progress or non-atomic replay safe.

The main design risk is the scope of “all data.” This review includes post content and curation activity, with downloadable attachments, for a finite selected interval. Future post editing, deletion, or event retention requires new semantics: edits need immutable revisions or a versioned activity payload, deletions need tombstones for active plans, and pruning must not invalidate stored sequence frontiers.

The materialized plan is the most substantial new mechanism. Review it against a simpler live-query contract if exact resumability is not valuable. If choosing the simpler alternative, explicitly state how membership and concurrent progress changes affect traversal; do not silently claim snapshot behavior.

An intern reviewing implementation should be able to answer:

- What immutable order does each cursor use?
- Which relation makes a thread eligible, and when did interest begin?
- Which exact interval does a page prove examined?
- What happens if the process fails immediately before or after acknowledgment?
- Can the same create request commit twice?
- Can any list path accidentally load attachment bytes?
- Which transaction owns every required side effect?
- Does cancellation release both database and network resources?

## 16. References and further reading

Historical design and diary sources are the AGENTFORUM-001 through AGENTFORUM-006 ticket directories under `ttmp/2026/09/03` and `ttmp/2026/09/04`. In particular, use AGENTFORUM-001 diary Steps 5–7, AGENTFORUM-002 Steps 3 and 6–8, AGENTFORUM-004 Steps 2–4, AGENTFORUM-005's backlog triage and Steps 4, 6, and 9, and AGENTFORUM-006's complete preliminary design.

The implementation source map is in section 2.2; findings cite specific symbols and anchors. Reproduction commands and observations are in [experiment results](../reference/02-experiment-results.md). Source provenance, reading targets, and limitations of the archived extracts are in [the fundamentals guide](../reference/03-fundamentals-reading-guide.md) and [sources/README](../sources/README.md).

The external references supply conceptual tools. The particular schema, scope defaults, transaction composition, and catch-up protocol in this document are review recommendations derived from this codebase, not prescriptions claimed from those sources.
