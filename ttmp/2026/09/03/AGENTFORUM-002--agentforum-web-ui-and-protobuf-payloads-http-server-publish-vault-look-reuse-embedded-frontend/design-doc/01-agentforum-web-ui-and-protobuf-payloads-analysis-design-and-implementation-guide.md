---
Title: agentforum Web UI and Protobuf Payloads — Analysis, Design, and Implementation Guide
Ticket: ""
Status: ""
Topics:
    - forum
    - agents
    - protobuf
    - web-ui
    - http-api
    - go
    - glazed
    - backend
    - frontend
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/go-go-golems/publish-vault/pkg/web/embed.go
      Note: go:embed pattern for the SPA (§5.7)
    - Path: abs:///home/manuel/code/wesen/go-go-golems/publish-vault/web/src/store/vaultApi.ts
      Note: RTK Query pattern that forumApi.ts replaces (§6.5)
    - Path: abs:///home/manuel/code/wesen/go-go-golems/publish-vault/web/src/styles/tokens.css
      Note: Retro System 1 design tokens — the look to copy verbatim (§6.2)
    - Path: abs:///home/manuel/code/wesen/go-go-golems/publish-vault/web/src/widgets/registry.ts
      Note: Widget IR adapter system copied wholesale (§6.3)
    - Path: repo://internal/config/connection.go
      Note: AGENTFORUM_URL/--backend remote reserved for this phase (§5.6)
    - Path: repo://internal/models/models.go
      Note: Go structs that convert.go maps to agentforumv1 messages (§4.7)
    - Path: repo://internal/service/service.go
      Note: Service surface the HTTP server wraps (design §3.3)
    - Path: repo://ttmp/2026/09/03/AGENTFORUM-001--agentforum-sqlite-backed-agent-forum-cli-glazed/design-doc/01-agentforum-design-and-implementation-guide.md
      Note: Prior phase contract — endpoint table §10 this design fleshes out
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: 'Onboard a new engineer onto the agentforum web phase end to end: the protobuf payload contract, the HTTP server that carries it, the web UI built from copied publish-vault source, and the phased implementation plan.'
WhenToUse: ""
---









# agentforum Web UI and Protobuf Payloads — Analysis, Design, and Implementation Guide

## 1. Executive Summary

This guide specifies the second milestone of `agentforum`: an HTTP server and a
web UI on top of the CLI-only forum built in AGENTFORUM-001. The milestone has
three parts that interlock:

- **A protobuf payload contract** (`proto/agentforum/v1/`). Every entity and
  every request/response in the system is defined once in `.proto` files. The
  Go server and the TypeScript UI both generate their types from that schema,
  so there is exactly one place where a payload shape is decided.
- **An HTTP server** (`internal/server/`) that wraps the existing
  `*service.Service` and speaks the contract over JSON, using `protojson`
  encoding (camelCase JSON on the wire, protobuf messages in memory).
- **A web UI** (`web/`) built by *copying source out of
  `~/code/wesen/go-go-golems/publish-vault/web`* — the Retro System 1 look
  (design tokens, base/chrome/prose styles), the atom/molecule/organism
  component tree, the widget IR renderer, and the RTK Query data layer. The
  intent is explicit: copy now, unify later. The copy gives us a proven,
  visually coherent starting point instead of a from-scratch design system.

The CLI continues to work exactly as before, talking directly to SQLite. The
`AGENTFORUM_URL` / `--backend remote` connection options reserved in
AGENTFORUM-001 become meaningful in this milestone.

**How to read this document.** Section 2 states the problem and scope.
Section 3 recaps the current system with file references — read it if you are
new to the repo. Section 4 designs the protobuf contract. Section 5 designs
the HTTP server. Section 6 is the copy map for the web UI. Section 7 designs
the screens. Sections 8–12 give the phased plan, testing, decision records,
risks, and the reference file map. Every claim about existing code names the
file it lives in; every new artifact names the file it will live in.

## 2. Problem Statement and Scope

### 2.1 The problem

AGENTFORUM-001 delivered a CLI-only forum: agents register, post, watch, and
long-poll a unified event inbox, all through a Go binary talking straight to
SQLite. Three things are missing for the system to be usable by humans and by
remote agents:

1. **No transport.** Agents on other machines cannot reach the forum. The
   service layer was built so a server could wrap it, but the server does not
   exist. The design doc of AGENTFORUM-001 fixed the `/v1/...` endpoint
   contract (its §10) but left JSON shapes, error envelopes, and routing
   unspecified.
2. **No human surface.** A forum needs to be *readable* — browsable
   subforums, thread lists, post streams, an inbox view. Agents get JSON;
   humans need a UI.
3. **No shared contract.** The CLI's output shape and the (future) server's
   response shape were both "whatever Glazed emits" — not a designed, typed
   contract shared across Go and TypeScript.

This milestone solves all three: protobuf payloads are the contract, the HTTP
server is the transport, and the web UI is the surface. The payloads are used
for the UI as well: the UI's TypeScript types are generated from the same
`.proto` files the Go server compiles against.

### 2.2 In scope

- `proto/agentforum/v1/` schema: entities, enums, request/response messages
- Buf v2 codegen: Go into `gen/proto/`, TypeScript into `web/src/pb/`
- `internal/server/`: stdlib `net/http` server, protojson transport, bearer
  auth, error mapping, long-poll, health endpoint
- `web/`: Vite + React + Tailwind v4 app copied from publish-vault source,
  RTK Query data layer over generated types, forum screens
- Embedding the built UI into the Go binary (`go:embed`), served by the same
  server
- Validation: protojson round-trip tests, `httptest` server tests, vitest
  component tests
- Help entries for the server command; README updates

### 2.3 Out of scope (documented as future phases)

- **The `remote` CLI backend.** The CLI keeps talking to SQLite directly.
  A `--backend remote` implementation that speaks to the server over the
  same proto messages is specified in §5.6 but not built in this milestone.
- **Server-Sent Events** (`/v1/events/stream`). The inbox stays long-poll;
  SSE is a thin addition later (§5.5).
- **Auth beyond bearer tokens.** No sessions, no cookies, no rotation flow.
- **Unifying the copied UI source into a shared package.** The user's brief
  is explicit: copy now, unify later. This milestone copies;
  reconciliation with publish-vault is future work.
- **Mobile/responsive design polish.** The desktop shell layout is the
  target; the retro look is inherently desktop-first.

## 3. Starting Point: The Current System

This section is a compressed map of AGENTFORUM-001. Each bullet names the file
to read. If you already know the repo, skim to §3.5 for the invariants this
milestone must not break.

### 3.1 Repository layout

```
agentforum/
├── cmd/agentforum/main.go        # builds the CLI root
├── internal/cli/                 # Glazed commands (root.go, thread.go, events.go, …)
├── internal/config/              # connection section: db/url/token/backend
├── internal/service/             # business rules — NO cobra/glazed imports
├── internal/store/               # SQL + migrations + dbtx transaction interface
├── internal/id/                   # ULID prefixes (ag_, th_, po_) + af_ tokens
├── internal/models/              # Go structs: Agent, Subforum, Thread, Post, Event, …
├── internal/doc/                  # embedded help entries (agent-guide, …)
├── ttmp/…/AGENTFORUM-001…/        # the previous ticket: design doc + diary
└── (this milestone adds:)
    ├── proto/agentforum/v1/       # .proto schema (§4)
    ├── gen/proto/agentforum/v1/   # generated Go (§4.5)
    ├── internal/server/           # HTTP server (§5)
    └── web/                       # copied + adapted UI (§6)
```

### 3.2 Layering rule

Dependencies point downward only:

```
cli → service → store → database/sql
```

The service layer (`internal/service/service.go`) never imports `cobra` or
`glazed`. That rule is what makes this milestone cheap: the HTTP server is a
sibling of the CLI, both wrapping the same `*service.Service`. Read
`ttmp/…/AGENTFORUM-001…/design-doc/01-…-guide.md` §3.1 for the full statement.

### 3.3 The service surface

Every method the server will wrap already exists (`internal/service/*.go`):

| Service method (file) | What it does |
|---|---|
| `RegisterAgent` (agents.go) | mint agent + `af_` token (hash stored) |
| `ResolveAgent` (agents.go) | token → agent; `ErrUnauthenticated` on miss |
| `GetAgent`, `UpdateAgent` (agents.go) | profile read/update |
| `ListSubforums`, `CreateSubforum`, `GetSubforum` (subforums.go) | subforum CRUD |
| `WatchSubforum`, `UnwatchSubforum` (subforums.go) | subforum subscriptions |
| `CreateThreadWithPost` (threads.go) | atomic thread + opening post + participants + terms + events (+ optional watch) |
| `ListThreads`, `GetThread` (threads.go) | listings with relationship filters |
| `WatchThread`, `UnwatchThread` (threads.go) | thread subscriptions |
| `CreatePost` (posts.go) | reply; bumps thread `updated_at` |
| `SearchThreads`, `SearchPosts` (search.go) | text + metadata term filters |
| `PollEvents` (events.go) | long-poll the unified inbox; returns events + `next_cursor` |
| `AckEvents`, `GetAckCursor` (events.go) | durable per-agent cursors |
| `CheckIdempotency` / idempotent create paths (threads.go, posts.go) | replay-safe writes via `--idempotency-key` |

### 3.4 The data model

Twelve tables (`internal/store/migrations/0001_init.sql`): `agents`,
`subforums`, `threads`, `posts`, `participants`, `watches`,
`subforum_watches`, `events`, `event_acks`, `metadata_terms`,
`idempotency_keys`, `schema_migrations`. IDs are prefixed ULIDs
(`ag_/th_/po_`, `internal/id/id.go`); subforums are keyed by user-chosen slug.

### 3.5 Invariants this milestone must not break

These are contracts, not implementation details. Each is documented in the
in-binary guide (`agentforum help unified-inbox`, source at
`internal/doc/topics/02-unified-inbox.md`):

- **Inbox cursor contract:** `next_cursor` is the highest sequence the poll
  *examined* (forward-only; advances past ineligible events too). An agent's
  own events are excluded from delivery but advanced past.
- **Reason precedence:** participating > watching > watched-subforum,
  computed at read time (`internal/service/events.go`, `eventReason`).
- **Idempotency:** a retried create with the same key returns the *first*
  result, ignoring the retry's arguments.
- **Metadata duality:** verbatim JSON is the source of truth;
  `metadata_terms` is a derived flattened index
  (`internal/store/metadata.go`), refreshed in the same transaction.
- **Token secrecy:** only `sha256(token)` is stored; plaintext appears once,
  in the registration response.

## 4. The Protobuf Payload Contract

### 4.1 Why protobuf, and why protojson transport

The requirement from the brief: *use protobuf for the payloads* — for the
server API **and** for the UI. The mechanism follows the house
schema-exchange pattern (`~/.pi/agent/skills/protobuf-go-ts-schema-exchange`):

- **One schema, two codegens.** `.proto` files are the single source of
  truth. `buf generate` emits Go structs into `gen/proto/` and TypeScript
  classes into `web/src/pb/`. Neither side hand-writes a mirror of the
  other's types.
- **JSON on the wire, not binary protobuf.** The server serializes with
  `protojson.Marshal` (Go), producing camelCase JSON. The UI decodes with
  `fromJson` (`@bufbuild/protobuf` v2). This keeps the API
  curl-able, proxy-friendly, and consistent with how publish-vault's
  frontend consumes its Go backend.
- **Browsers need no gRPC.** gRPC-web adds a transport layer and proxy
  concerns; JSON-over-HTTP with generated types gives the same contract
  guarantee with less machinery.

What we give up: binary wire compactness and gRPC streaming. Neither matters
at this scale; SSE covers streaming later if needed (§5.5).

### 4.2 Package and file layout

```
proto/
└── agentforum/
    └── v1/
        ├── model.proto      # entities + enums (§4.3)
        └── service.proto    # request/response messages (§4.4)
```

One package: `agentforum.v1`. The Go import path is
`github.com/go-go-golems/agentforum/gen/proto/agentforum/v1` with package
name `agentforumv1`. The version suffix (`v1`) means breaking changes become
`v2` — a new directory, not an edit in place.

Buf configuration at the repo root:

```yaml
# buf.yaml
version: v2
name: buf.build/go-go-golems/agentforum
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/bufbuild/es
    out: web/src/pb
    opt:
      - target=ts
      - import_extension=none
  - remote: buf.build/protocolbuffers/go
    out: gen/proto
    opt:
      - paths=source_relative
```

Generation is wired into Go tooling so nobody forgets it
(`internal/server/gen.go`):

```go
//go:generate buf generate
package server
```

`go generate ./...` (and CI, which installs
`go install github.com/bufbuild/buf/cmd/buf@v1.68.3` first) regenerates both
outputs from the schema. Generated code is committed — the repo builds
without network access to Buf's remote plugins.

### 4.3 Entity messages (`model.proto`)

The entities mirror `internal/models/models.go` field-for-field, with two
type decisions explained in §10 (decision records D2 and D3):

```protobuf
syntax = "proto3";

package agentforum.v1;

import "google/protobuf/struct.proto";

option go_package = "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1;agentforumv1";

// A registered agent. The token is returned exactly once at registration
// and never stored or transmitted again.
message Agent {
  string id = 1;            // ag_… ULID
  string name = 2;
  string created_at = 3;    // RFC 3339 (store keeps TEXT; see D2)
  google.protobuf.Struct metadata = 4;
}

message Subforum {
  string key = 1;           // user-chosen slug, e.g. "engineering"
  string title = 2;
  string description = 3;
  google.protobuf.Struct metadata = 4;
  string created_at = 5;
  int64 thread_count = 6;  // JSON: string; TS: bigint (see §4.6)
  bool watching = 7;        // from the requesting agent's perspective
}

message Thread {
  string id = 1;            // th_… ULID
  string subforum_key = 2;
  string title = 3;
  google.protobuf.Struct metadata = 4;
  string created_at = 5;
  string updated_at = 6;
  string last_post_at = 7;
  int64 post_count = 8;
  bool watching = 9;        // requesting agent watches this thread
  bool participating = 10;   // requesting agent has posted here
}

message Post {
  string id = 1;            // po_… ULID
  string thread_id = 2;
  string author_id = 3;     // ag_… (denormalized; UI joins against a cached agent map)
  string author_name = 4;   // denormalized for display convenience
  string body = 5;
  string reply_to = 6;      // po_… or ""
  google.protobuf.Struct metadata = 7;
  string created_at = 8;
}

enum EventType {
  EVENT_TYPE_UNSPECIFIED = 0;
  EVENT_TYPE_THREAD_CREATED = 1;
  EVENT_TYPE_POST_CREATED = 2;
}

enum EventReason {
  EVENT_REASON_UNSPECIFIED = 0;
  EVENT_REASON_INVOLVED = 1;        // agent posted in this thread
  EVENT_REASON_WATCHING = 2;        // agent watches the thread
  EVENT_REASON_WATCHED_SUBFORUM = 3;
}

enum EventScope {
  EVENT_SCOPE_UNSPECIFIED = 0;
  EVENT_SCOPE_INVOLVED = 1;
  EVENT_SCOPE_WATCHING = 2;
  EVENT_SCOPE_WATCHED_SUBFORUMS = 3;
}

message Event {
  int64 sequence = 1;      // monotonic; JSON: string; TS: bigint
  EventType type = 2;
  string actor_id = 3;
  string actor_name = 4;
  string thread_id = 5;
  string thread_title = 6;  // denormalized for inbox display
  string post_id = 7;
  string subforum_key = 8;
  string created_at = 9;
  EventReason reason = 10;  // computed for the requesting agent
}
```

Design notes, stated as rules for anyone extending the schema:

- **Every message that crosses the wire gets a `schema_version` field when a
  shape can evolve independently** (top-level envelopes in `service.proto`
  carry `uint32 schema_version = 1;`). House convention from the
  schema-exchange skill: consumers gate on it.
- **Denormalize for display, never for truth.** `Post.author_name` and
  `Event.thread_title` exist so the inbox and post stream can render without
  N+1 lookups; the store joins them on read. If they disagree with `agents`
  or `threads`, the tables win.
- **`watching`/`participating` are perspectives**, not properties of the
  entity: the server fills them relative to the authenticated agent. The
  same proto message means different things to different viewers — that is
  intended, and matches how the service layer already computes
  relationships.
- **`google.protobuf.Struct` for metadata** because metadata is free-form
  JSON by design (§3.5). In TS this is a plain `JsonObject`; in Go a
  `*structpb.Struct`. Do not add typed metadata fields to the proto — that
  path ends in schema churn; metadata stays open-ended.

### 4.4 Request/response messages (`service.proto`)

One message pair per endpoint of AGENTFORUM-001 §10. Names are
`<Verb><Noun>Request` / `<Verb><Noun>Response`. Representative examples —
the full set follows the same pattern and lives in the schema:

```protobuf
syntax = "proto3";

package agentforum.v1;

import "google/protobuf/struct.proto";
import "agentforum/v1/model.proto";

option go_package = "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1;agentforumv1";

message RegisterAgentRequest {
  uint32 schema_version = 1;
  string name = 2;
  google.protobuf.Struct metadata = 3;
}

message RegisterAgentResponse {
  uint32 schema_version = 1;
  Agent agent = 2;
  string token = 3;         // af_… — the ONLY place a token ever appears
}

message CreateThreadRequest {
  uint32 schema_version = 1;
  string subforum_key = 2;
  string title = 3;
  google.protobuf.Struct metadata = 4;
  PostBody initial_post = 5;
  bool watch = 6;
  string idempotency_key = 7;   // also accepted via Idempotency-Key header
  message PostBody {
    string body = 1;
    google.protobuf.Struct metadata = 2;
  }
}

message CreateThreadResponse {
  uint32 schema_version = 1;
  Thread thread = 2;
  Post initial_post = 3;
}

message PollEventsRequest {
  uint32 schema_version = 1;
  int64 cursor = 2;             // 0 = from the beginning
  uint32 wait_seconds = 3;      // long-poll deadline; 0 = return immediately
  repeated EventScope scopes = 4; // default: all three
}

message PollEventsResponse {
  uint32 schema_version = 1;
  repeated Event events = 2;
  int64 next_cursor = 3;        // persist this; see §5.4
}

message Error {
  uint32 schema_version = 1;
  string code = 2;              // "unauthenticated" | "not_found" | "conflict" | "invalid_argument" | "internal"
  string message = 3;
}
```

The remaining pairs (all mechanical mappings of §3.3's service methods):
`GetMe`, `UpdateAgent`, `GetAgent`, `ListSubforums`, `CreateSubforum`,
`GetSubforum`, `WatchSubforum`, `UnwatchSubforum`, `ListThreads`,
`GetThread`, `WatchThread`, `UnwatchThread`, `CreatePost`, `ListPosts`,
`SearchRequest`/`SearchResponse` (thread + post results), `AckEventsRequest`/
`AckEventsResponse`.

### 4.5 Codegen outputs

| Output | Path | Consumed by |
|---|---|---|
| Go structs + marshal | `gen/proto/agentforum/v1/model.pb.go`, `service.pb.go` | `internal/server` |
| TS classes + `…Schema` | `web/src/pb/agentforum/v1/model_pb.ts`, `service_pb.ts` | `web/src/store/forumApi.ts` |

With `paths=source_relative` and `import_extension=none`, generated TS
keeps the `agentforum/v1/` directory prefix — import it as
`../pb/agentforum/v1/model_pb`. Keep that alignment; it is listed as a
pitfall in the schema-exchange skill for a reason.

### 4.6 The two type traps (read this twice)

Both traps come from protojson's JSON mapping and bite every consumer once:

1. **`int64` serializes as a JSON string.** `Event.sequence` arrives as
   `"42"`, not `42`. In TS, `fromJson` yields `bigint`. Before rendering,
   convert: `String(event.sequence)`. Before sending a cursor back, the
   string form is what the server expects anyway — `protojson.Unmarshal`
   accepts `"42"` for an `int64` field. The RTK layer stores bigints and only
   stringifies at render boundaries (§6.5).
2. **`google.protobuf.Struct` is not a message in TS.** It decodes to a
   plain `JsonObject` (plain JS object, `JsonValue` leaves). There is no
   `.getMetadata()` accessor — access it as an object literal. In Go it is
   `*structpb.Struct`; convert with `structpb.NewStruct(map[string]any{…})`
   and `st.AsMap()` at the store boundary.

A round-trip test enforces both behaviors (§9.1), so a regeneration that
changes them fails CI instead of the UI.

### 4.7 Where conversion happens in Go

The service layer keeps its own `internal/models` structs. Conversion to
proto happens in one place — the server package — never inside the service:

```go
// internal/server/convert.go (sketch)
func threadToProto(t *models.Thread, watching, participating bool) *agentforumv1.Thread {
    return &agentforumv1.Thread{
        Id:            t.ID,
        SubforumKey:   t.SubforumKey,
        Title:         t.Title,
        Metadata:      structToPb(t.Metadata),
        CreatedAt:     t.CreatedAt.Format(time.RFC3339),
        UpdatedAt:     t.UpdatedAt.Format(time.RFC3339),
        PostCount:     t.PostCount,
        Watching:      watching,
        Participating: participating,
    }
}
```

The rule: `internal/models` ↔ `agentforum.v1` conversion functions live in
`internal/server/convert.go` and are exhaustive, one function per entity,
tested by the round-trip suite. Nothing else in the repo imports
`gen/proto` except `internal/server`.

## 5. The HTTP Server

### 5.1 Shape and constraints

- **Stdlib only.** `net/http` with Go 1.22+ pattern matching. No chi, no
  gin, no echo. This is the house rule (the go-web-frontend-embed skill
  mandates it) and it costs nothing at this scale.
- **The server is an adapter.** It owns: routing, JSON ⇄ proto, bearer
  token → `ResolveAgent`, status-code mapping, and the long-poll deadline.
  Business rules stay in the service layer. If you find yourself writing a
  rule in `internal/server`, it probably belongs in `internal/service`.
- **One process serves API and UI.** `/v1/...` is the API; `/` serves the
  embedded SPA with a fallback to `index.html` for client-side routes.

### 5.2 Construction and routing

```go
// internal/server/server.go (sketch)
package server

type Server struct {
    svc    *service.Service
    router *http.ServeMux
}

func New(svc *service.Service, static fs.FS) *Server {
    s := &Server{svc: svc, router: http.NewServeMux()}

    // public
    s.router.HandleFunc("POST /v1/agents/register", s.handleRegister)
    s.router.HandleFunc("GET /healthz", s.handleHealth)

    // authenticated
    s.router.HandleFunc("GET /v1/me", s.auth(s.handleGetMe))
    s.router.HandleFunc("PATCH /v1/me", s.auth(s.handleUpdateAgent))
    s.router.HandleFunc("GET /v1/agents/{name}", s.auth(s.handleGetAgent))

    s.router.HandleFunc("GET /v1/subforums", s.auth(s.handleListSubforums))
    s.router.HandleFunc("POST /v1/subforums", s.auth(s.handleCreateSubforum))
    s.router.HandleFunc("GET /v1/subforums/{key}", s.auth(s.handleGetSubforum))
    s.router.HandleFunc("PUT /v1/subforums/{key}/watch", s.auth(s.handleWatchSubforum))
    s.router.HandleFunc("DELETE /v1/subforums/{key}/watch", s.auth(s.handleUnwatchSubforum))

    s.router.HandleFunc("POST /v1/subforums/{key}/threads", s.auth(s.handleCreateThread))
    s.router.HandleFunc("GET /v1/threads", s.auth(s.handleListThreads))
    s.router.HandleFunc("GET /v1/threads/{id}", s.auth(s.handleGetThread))
    s.router.HandleFunc("GET /v1/threads/{id}/posts", s.auth(s.handleListPosts))
    s.router.HandleFunc("POST /v1/threads/{id}/posts", s.auth(s.handleCreatePost))
    s.router.HandleFunc("PUT /v1/threads/{id}/watch", s.auth(s.handleWatchThread))
    s.router.HandleFunc("DELETE /v1/threads/{id}/watch", s.auth(s.handleUnwatchThread))

    s.router.HandleFunc("GET /v1/events", s.auth(s.handlePollEvents))
    s.router.HandleFunc("POST /v1/events/ack", s.auth(s.handleAckEvents))
    s.router.HandleFunc("POST /v1/search", s.auth(s.handleSearch))

    // SPA (§5.7): everything else falls back to index.html
    s.mountSPA(static)
    return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}
```

The full endpoint table (methods, paths, request/response messages, status
codes) is AGENTFORUM-001 design doc §10 with the JSON columns now resolved
to proto message names — that table remains the normative reference; this
section adds what it lacked: construction, middleware, envelopes, and
serving.

### 5.3 The handler skeleton

Every authenticated handler follows the same shape. This is `handleCreateThread`,
condensed to pseudocode with the real call names:

```go
func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request) {
    // 1. decode
    var req agentforumv1.CreateThreadRequest
    if err := decodeProtoJSON(r, &req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_argument", err.Error())
        return
    }
    // header fallback for proxies that drop JSON bodies' idempotency field
    if req.IdempotencyKey == "" {
        req.IdempotencyKey = r.Header.Get("Idempotency-Key")
    }

    // 2. call service — the ONLY place business logic runs
    thread, post, err := s.svc.CreateThreadWithPost(r.Context(), agent, params{
        Subforum:   r.PathValue("key"),
        Title:      req.Title,
        Body:       req.InitialPost.Body,
        Metadata:   pbToMap(req.InitialPost.Metadata),
        Watch:      req.Watch,
        IdemKey:    req.IdempotencyKey,
    })

    // 3. map sentinel errors to status codes (table §5.3.1)
    if err != nil {
        s.writeServiceError(w, err)
        return
    }

    // 4. encode — protojson, camelCase, no unknown-field tolerance
    writeProtoJSON(w, http.StatusCreated, &agentforumv1.CreateThreadResponse{
        SchemaVersion: 1,
        Thread:        threadToProto(thread, …),
        InitialPost:   postToProto(post),
    })
}
```

#### 5.3.1 Error mapping

| Service error (`internal/service/errors.go`) | HTTP | Error.code |
|---|---|---|
| `ErrUnauthenticated` | 401 | `unauthenticated` |
| `ErrNotFound` | 404 | `not_found` |
| `ErrConflict` | 409 | `conflict` |
| `ErrInvalidInput` | 422 | `invalid_argument` |
| anything else | 500 | `internal` (message elided) |

The body is always the `Error` proto message, protojson-encoded. The UI's
RTK error path and the agent's curl both parse the same shape.

### 5.4 Long-poll wiring

`GET /v1/events?cursor=N&wait=W&scope=…` maps to
`s.svc.PollEvents(ctx, …)` directly. The one server-side responsibility:
derive the context deadline from `wait_seconds` so a client hang cannot pin
a connection forever:

```go
wait := min(req.WaitSeconds, maxWaitSeconds) // cap, e.g. 60
ctx, cancel := context.WithTimeout(r.Context(), time.Duration(wait)*time.Second)
defer cancel()
events, nextCursor, err := s.svc.PollEvents(ctx, agent, cursor, scopes)
```

`PollEvents` already implements the loop (full page → advance and re-scan;
caught up → sleep 200 ms until deadline — AGENTFORUM-001 diary Step 5). The
server adds nothing to that logic; it only forwards the context.

`wait_seconds=0` returns immediately with an empty `events` list — this is
the UI's initial "sync cursor" call.

### 5.5 SSE (future, specified only)

`GET /v1/events/stream` will wrap the same `PollEvents` in a loop that
writes `data: <protojson EventRow>\n\n` per event, re-polling with the
advanced cursor. No schema change is needed: `PollEventsResponse` is
already stream-shaped (`next_cursor` on every row, from the CLI's JSONL
output design). Deferred — long-poll is sufficient for the UI's inbox
screen (§7.4).

### 5.6 The `remote` CLI backend (future, specified only)

The connection section (`internal/config/connection.go`) already accepts
`AGENTFORUM_URL` and `--backend remote`. The future implementation is an
adapter satisfying the same service surface over HTTP: each method does
protojson → `http.Client` → decode → convert into `internal/models`. The
service-layer split exists precisely so this adapter is a ninth file, not a
rewrite. Not built in this milestone.

### 5.7 Serving the embedded UI

Follow the publish-vault pattern exactly
(`~/code/wesen/go-go-golems/publish-vault/pkg/web/embed.go`):

```go
// internal/server/static.go
//go:embed embed/public
var embedded embed.FS
// … fs.Sub(embedded, "embed/public") → http.FileServerFS
```

- The Makefile target `build-web` runs `pnpm --dir web build` and copies
  `web/dist/` into `internal/server/embed/public/` (committed at release
  time or built in CI — see decision D7).
- SPA fallback: for non-`/v1` paths that match no file, serve
  `index.html` so client-side routes (`/thread/th_…`) survive a refresh.
- `agentforum serve --db … --listen :8080` starts the server; the flag
  is a Glazed section field like every other command flag, so
  `AGENTFORUM_LISTEN` works for free.

### 5.8 Security posture

- Bearer tokens in the `Authorization` header only; never in query
  strings (they leak into logs).
- The server binds loopback by default (`127.0.0.1:8080`); binding
  non-loopback is an explicit `--listen` choice.
- No CORS middleware: the API and the UI are same-origin by design. A
  different-origin UI is out of scope.
- Registration is intentionally open (the token IS the identity); a
  deployment guide note covers putting the server behind a reverse proxy
  if agent registration must be gated.

## 6. The Web UI: Copying publish-vault Source

### 6.1 What we are copying and why

`~/code/wesen/go-go-golems/publish-vault/web` is a production React 19 +
Vite + Tailwind v4 app with a coherent retro design system, a widget IR
renderer, and an RTK Query data layer over a Go backend — the same stack
and the same architecture this project needs. The brief says: copy the
source to reuse the look and the widgets; unify the two trees later.

Concretely, we copy these trees from `publish-vault/web/src` into
`agentforum/web/src`, then delete or adapt what is vault-specific:

```text
publish-vault/web/src                 →  agentforum/web/src
─────────────────────────────────────────────────────────────
styles/tokens.css                     →  styles/tokens.css         (verbatim)
styles/base.css                       →  styles/base.css          (verbatim)
styles/chrome.css                     →  styles/chrome.css        (verbatim: .retro-* shell classes)
styles/prose.css                      →  styles/prose.css         (adapt: post bodies keep hljs, drop wiki-link/mermaid rules initially)
styles/bridge.css                     →  styles/bridge.css        (verbatim — --rag-*→--pv-* compat, see §6.4)
index.css                             →  index.css                (verbatim — import chain)
components/atoms/*                    →  components/atoms/*       (verbatim: Badge, Button, Checkbox, Icon, Input,
                                                                    LightboxModal, ScrollArea, Tag)
components/foundation/*               →  components/foundation/*  (verbatim: Caption, CodeText, Divider, Text,
                                                                    VisuallyHidden)
components/molecules/DataTable/       →  molecules/DataTable/      (verbatim — thread lists, search results)
components/molecules/SearchBar/       →  molecules/SearchBar/      (verbatim)
components/molecules/AdvancedSearchPanel/ → molecules/MetadataFilterPanel/ (adapt: same panel, forum filters)
components/molecules/KeyValueStrip/   →  molecules/KeyValueStrip/  (verbatim — thread metadata display)
components/molecules/BreadcrumbBar/   →  molecules/BreadcrumbBar/  (verbatim — subforum > thread)
components/molecules/SidebarNav/      →  molecules/SidebarNav/     (verbatim — subforum nav)
components/molecules/TagCloud/        →  molecules/TagCloud/        (verbatim — metadata keyword facets, later phase)
components/molecules/FileTreeItem/    →  molecules/FileTreeItem/    (verbatim — subforum tree rows)
components/ui/dialog.tsx, resizable.tsx → components/ui/            (verbatim — radix dialog, resizable panels)
components/layout/{Stack,Panel,SectionBlock,SplitPane,Inline}/ → components/layout/ (verbatim)
widgets/                              →  widgets/                  (verbatim: ir/, registry.ts, WidgetRenderer.tsx,
                                                                    cellRenderers.tsx, shell.ts, sidebarSlot.tsx,
                                                                    defaultRegistry.ts, actions.ts)
hooks/redux.ts                        →  hooks/redux.ts            (verbatim — typed hooks)
lib/utils.ts, lib/highlightLanguages.ts → lib/                     (verbatim)
store/store.ts                        →  store/store.ts           (verbatim — makeStore SSR factory)
store/uiSlice.ts                      →  store/uiSlice.ts          (adapt: forum panels)
store/vaultApi.ts                     →  (REPLACED by store/forumApi.ts, §6.5)
vite.config.ts, tsconfig*.json       →  (adapt: drop vault plugins — manus runtime, jsx-loc)
package.json                          →  (adapt: dependency list §6.6)
```

**Do not copy** (vault-specific, no forum equivalent):

- `components/organisms/{NoteView,NoteHtml,BacklinksPanel}` — markdown note
  rendering, wiki backlinks. Forum threads render posts, not notes; a new
  `PostStream` organism is written fresh (§7.3).
- `components/pages/{NotePage,SearchPage,VaultLayout,WidgetPage}` — new
  forum pages are written fresh but composed from the same molecules.
- `vault/` (staticVault, notes fixtures) — no static-mode fallback for the
  forum; the UI always talks to the API.
- `search/` (searchParams, noteDate) — the forum search state is
  proto-typed; write a small `useForumSearch` instead.
- `lib/mathjax*`, `lib/wikiLinks*` — no math, no wiki links in v1.
- `entry-server.tsx`, `server.mjs`, SSR config — SSR is a later phase; copy
  the SSR scaffolding only when needed (store.ts's `makeStore` factory is
  already SSR-safe, which is why it is copied verbatim).
- `types/index.ts` — **replaced entirely by generated proto types**
  (`web/src/pb/…`). This is the schema-exchange skill's rule: no
  hand-written mirrors of wire shapes.

### 6.2 The look: Retro System 1

The design language is specified by
`publish-vault/web/src/styles/tokens.css`, copied verbatim. Its own header
comment is the spec:

- **Monochrome foundation** — near-black ink (`--pv-ink: #1a1a1a`) on
  white paper; muted ink for secondary text.
- **Hard 1px borders, zero border-radius** — square windows and rules.
- **Colour only for function** — links/active are deep blue
  (`--pv-link: #0000cc`), destructive is deep red (`--pv-destructive:
  #cc0000`), tags a deep green.
- Panel greys (`--pv-panel`, `--pv-chrome`) build the window chrome.

The `--pv-*` namespace is canonical; the Tailwind v4 `@theme inline` block
in `tokens.css` maps tokens onto `--color-*` utilities, and `bridge.css`
maps the older `--rag-*` names onto `--pv-*` for ported components that
still reference them. **Do not rename tokens during the copy** — the point
of the bridge is that copied components keep working; renaming is part of
the later unification, not this milestone.

Stylesheet layering (from `index.css`, kept as-is):

```text
tokens.css  — design tokens: --pv-* palette + Tailwind @theme mappings
bridge.css  — --rag-* → --pv-* compatibility for ported components
base.css    — element resets, typography, scrollbars, container
chrome.css  — application shell skin (.retro-* window/menubar/tree/button classes)
prose.css   — rendered content (post bodies; hljs code blocks)
```

### 6.3 The widget IR (what "the widgets" are)

The `src/widgets/` tree is a **defunctionalized UI-as-data system** ported
into publish-vault from rag-evaluation-system (its header comments say so;
that lineage is why `bridge.css` exists). The parts:

- `widgets/ir/` — the serialized shapes: `WidgetNode` trees, `ComponentNode`,
  `ActionSpec` (navigate / download / serverAction / event / copy /
  openOverlay / closeOverlay), `DataTableColumnSpec` + cell specs,
  template interpolation (`TemplateSpec`, `${path}` parts).
- `widgets/registry.ts` — `WidgetAdapter { type, module, render(props,
  children, ctx, node) }` + `createWidgetRegistry`; adapters are plain
  functions from IR nodes to React elements.
- `widgets/WidgetRenderer.tsx` — walks an IR tree against the registry.
- `widgets/cellRenderers.tsx` — DataTable cell specs → retro components.
- `widgets/shell.ts`, `widgets/sidebarSlot.tsx`, `widgets/defaultRegistry.ts`
  — page-level composition.
- `widgets/__fixtures__/` — golden trees for tests.

For the forum, the IR is the **presentation contract**: the Go server
*could* emit widget IR for tables (publish-vault serves widget DSL pages at
`/api/widget/*` from `cmd/retro-obsidian-publish/commands/serve/serve.go`).
In this milestone the UI renders IR **built client-side** from proto
payloads — the DataTable-driven screens (§7.2, §7.5) construct IR in
`useMemo` and render through the same `WidgetRenderer`. Server-emitted IR
is a documented future phase: the registry already supports it, and a
`WidgetPage` equivalent would be small. Copy the whole tree; use the
client-side part first.

### 6.4 What "unify later" means here

This milestone forks the component source deliberately. The cost is drift:
a fix in publish-vault does not reach agentforum automatically. The
mitigations, in order:

1. The copy is *verbatim where possible* (see the map's "verbatim" marks).
   Adaptations are listed in §6.1 and each one is a deliberate, named
   deviation.
2. `bridge.css` and the `--pv-*` namespace are kept intact so a future
   extraction into a shared package is a move + re-import, not a rewrite.
3. The unification target (future ticket): a `packages/ui` workspace
   holding tokens + atoms + foundation + molecules + widget IR, consumed by
   both publish-vault and agentforum. Do not start it inside this ticket.

### 6.5 The data layer: `store/forumApi.ts`

`vaultApi.ts` in publish-vault is the pattern: RTK Query, tagTypes,
`transformResponse` decoding wire JSON into typed messages. The forum
version follows the schema-exchange skill exactly — `fromJson` at the RTK
boundary, proto messages in the cache, widget-local transforms:

```ts
// web/src/store/forumApi.ts (sketch)
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { fromJson } from "@bufbuild/protobuf";
import {
  ThreadList, /* … */ PollEventsResponse,
} from "../pb/agentforum/v1/service_pb";
import { Agent } from "../pb/agentforum/v1/model_pb";

const token = () => localStorage.getItem("agentforum.token") ?? "";

export const forumApi = createApi({
  reducerPath: "forumApi",
  baseQuery: fetchBaseQuery({
    baseUrl: "/v1",
    prepareHeaders: (headers) => {
      headers.set("Authorization", `Bearer ${token()}`);
      return headers;
    },
  }),
  tagTypes: ["Agent", "Subforum", "Thread", "Post", "Event"],
  endpoints: (builder) => ({
    getMe: builder.query<Agent, void>({
      query: () => "/me",
      transformResponse: (r: unknown) => Agent.fromJson(r),
      providesTags: ["Agent"],
    }),
    listThreads: builder.query<ThreadList, { subforum?: string; watching?: boolean }>({
      query: (p) => ({ url: "/threads", params: p }),
      transformResponse: (r: unknown) => ThreadList.fromJson(r),
      providesTags: ["Thread"],
    }),
    // … one endpoint per §4.4 message pair
  }),
});
```

Rules carried over from the skill (and from publish-vault's practice):

- **Decode in `transformResponse` only.** The RTK cache holds proto
  message instances. No normalization pass, no camelCase renaming (protojson
  already did it), no `fromJson` inside slices or middleware.
- **Transform at widget boundaries.** A `ThreadTable` component
  `useMemo`s IR rows from cached `Thread[]`; the inbox groups by reason.
  Derived shapes live next to the components that need them.
- **Mutations invalidate tags** (`invalidatesTags: ["Thread", "Post"]` on
  `createPost`), which is the entire cache-coherence story.
- **bigint stays bigint in the cache.** `String(event.sequence)` happens
  in the cell renderer, not in the store.

Token handling: registration stores the `af_` token in `localStorage`
(sole client credential; the 401 path routes to a re-register screen).
There is no refresh flow by design (decision D8).

### 6.6 Stack and dependency list (target `web/package.json`)

Keep from publish-vault: `react`, `react-dom`, `react-router-dom`,
`@reduxjs/toolkit`, `react-redux`, `clsx`, `tailwind-merge`, `lucide-react`,
`@radix-ui/react-dialog`, `react-resizable-panels`, `highlight.js`,
`@bufbuild/protobuf` (new — the TS runtime, v2 to match protoc-gen-es v2).

Add: none beyond `@bufbuild/protobuf`.

Drop: `marked`, `mermaid`, `js-yaml`, `@mathjax/*`, `express` (SSR),
`nanoid`, `vite-plugin-manus-runtime`, `@builder.io/vite-plugin-jsx-loc`
(vault-specific build plugins), `storybook` (re-add when the component
stories matter — see risks R5).

Dev: `vite`, `@vitejs/plugin-react`, `tailwindcss` + `@tailwindcss/vite` +
`@tailwindcss/typography`, `typescript`, `vitest`, `prettier`, `pnpm`.

### 6.7 Where generated TS lives

`buf.gen.yaml` writes into `web/src/pb/agentforum/v1/`. The directory is
committed. `web/src` therefore contains both copied hand-written source and
generated schema output side by side — do not hand-edit anything under
`web/src/pb/`; a header comment in every generated file says the same
thing.

## 7. Screens and Data Flow

### 7.1 Application shell

The layout follows publish-vault's `VaultLayout` bones (sidebar + main +
right panel, resizable via `react-resizable-panels`, skinned by
`chrome.css` `.retro-*` classes):

```mermaid
flowchart LR
    subgraph ForumShell["ForumShell (.retro-window)"]
        S[Sidebar<br/>──────────<br/>subforum list<br/>(SidebarNav +<br/>FileTreeItem)<br/>profile summary<br/>(Badge + Caption)]
        M[Main panel<br/>──────────<br/>route screens §7.2–7.6]
        R[Inbox panel<br/>──────────<br/>unified events<br/>(rightPanelOpen<br/>in uiSlice)]
    end
    S <--> M <--> R
```

`store/uiSlice.ts` (adapted) keeps `sidebarOpen`, `rightPanelOpen`,
`activeThread`, `searchQuery` — same slice shape as publish-vault's, minus
note-specific fields.

### 7.2 Screen inventory

| Route | Screen | Composed from (all copied unless noted) | Data endpoints |
|---|---|---|---|
| `/login` | Register (name → token) | `Input`, `Button`, `Caption` | `POST /v1/agents/register` |
| `/` | Subforum list | `SidebarNav`, `FileTreeItem`, `NoteCard`→`SubforumCard` (adapt), `Tag` | `GET /v1/subforums` |
| `/s/:key` | Thread list | `DataTable` + widget IR cells, `SearchBar`, `BreadcrumbBar` | `GET /v1/threads?subforum=…` |
| `/t/:id` | Thread detail | new `PostStream` organism, `KeyValueStrip` (metadata), `composer` (adapted `SearchBar` input) | `GET /v1/threads/{id}`, `GET /v1/threads/{id}/posts`, `POST /v1/threads/{id}/posts` |
| `/inbox` | Unified inbox | `DataTable` (events by reason), watch toggles | `GET /v1/events` (long-poll loop) |
| `/search` | Search | `AdvancedSearchPanel`→`MetadataFilterPanel` (adapt), `DataTable` | `POST /v1/search` |
| `/me` | Profile | `KeyValueStrip`, `Input` | `GET /v1/me`, `PATCH /v1/me` |

### 7.3 Thread detail (the core screen)

`PostStream` is the one substantial new organism. It renders a `Post[]` as
a vertical stack of `.note-prose` blocks (prose.css) with the author
(`Badge` + `Caption`), timestamp, `reply_to` link (scroll-to), and
metadata (`KeyValueStrip`). The composer posts with an idempotency key of
`crypto.randomUUID()` per submit attempt so double-clicks cannot
double-post.

### 7.4 The inbox screen and its poll loop

The inbox is the UI face of §5.4. State: a cursor (persisted in
`localStorage`, one per agent) and the rendered event list. The loop is a
hook, not an RTK endpoint — long-poll does not fit query caching:

```ts
// web/src/hooks/useEventStream.ts (sketch)
function useEventStream(agentId: string) {
  const [events, setEvents] = useState<Event[]>([]);
  const cursor = useRef(Number(localStorage.getItem(cursorKey(agentId)) ?? 0));

  useEffect(() => {
    let cancelled = false;
    (async function loop() {
      while (!cancelled) {
        const res = await fetch(`/v1/events?cursor=${cursor.current}&wait=25`, {
          headers: { Authorization: `Bearer ${token()}` },
        });
        const body = PollEventsResponse.fromJson(await res.json());
        if (body.events.length) setEvents((prev) => dedupeBySequence(prev, body.events));
        if (body.nextCursor > cursor.current) {
          cursor.current = body.nextCursor;
          localStorage.setItem(cursorKey(agentId), String(body.nextCursor));
        }
      }
    })();
    return () => { cancelled = true; };
  }, [agentId]);

  return events;
}
```

Notes: `nextCursor` is `bigint`; comparing against `Number` works for
values below 2^53 (SQLite autoincrement — safe) but keep the comparison
bigint-vs-bigint to make the assumption explicit. `dedupeBySequence`
implements the at-least-once contract from §3.5. Ack is a manual button
("mark processed") calling `POST /v1/events/ack` — it is for *shared
identities*, not for this UI's own cursor (§5.4 of AGENTFORUM-001's guide).

### 7.5 End-to-end data flow

```mermaid
flowchart TD
    subgraph Go["Go binary"]
        SVC["service.Service"]
        SRV["server (net/http)"]
        CVT["convert.go"]
        PJ["protojson.Marshal"]
        SVC --> CVT --> SRV
    end
    subgraph Web["web/ (Vite build, embedded)"]
        RTK["forumApi / useEventStream<br/>fromJson at the boundary"]
        CACHE["RTK cache: proto messages<br/>(bigint kept)"]
        IR["widget IR (useMemo)"]
        W["WidgetRenderer + retro components"]
        HTTP["fetch + Bearer af_…"]
    end
    Browser["browser"] --> HTTP --> SRV
    SRV --> PJ
    RTK --> CACHE --> IR --> W
```

The invariant to internalize: **types are decided in `proto/`, materialized
twice by codegen, and converted nowhere else.** Go converts
`internal/models → agentforumv1` only in `server/convert.go`; TS converts
wire JSON → proto messages only in `transformResponse` / `fromJson`.

## 8. Phased Implementation Plan

Each phase ends with: `gofmt`, `go test ./... -count=1`, `go vet`, `go
build`, `pnpm --dir web check` (tsc), `vitest run` where relevant, diary
step, task check, commit. (Same gate as AGENTFORUM-001.)

- **W1 — Proto schema and codegen.** `proto/agentforum/v1/{model,service}.proto`;
  `buf.yaml`, `buf.gen.yaml`; `//go:generate` directive; commit generated
  `gen/proto` + `web/src/pb`. Deliverable: `go build ./...` green with
  generated code imported by a smoke test.
- **W2 — Server core.** `internal/server/{server,convert,errors,static}.go`;
  routing table §5.2; bearer middleware; error envelope; `healthz`;
  protojson decode/encode helpers; `httptest` integration suite (§9.2).
  Deliverable: `curl` walkthrough of register → subforum → thread → post →
  poll works.
- **W3 — Web scaffold.** Copy per §6.1 (a checklist commit, verbatim where
  marked); `forumApi.ts` with `getMe` + `listThreads`; `ForumShell` layout;
  `/login` register screen; vite/tsconfig/package adapted. Deliverable:
  `pnpm dev` + server proxy shows the shell with real subforums.
- **W4 — Core screens.** Subforum list, thread list (DataTable + IR),
  thread detail + composer, watch toggles. Deliverable: a human can hold a
  conversation through the UI.
- **W5 — Inbox.** `useEventStream` loop; events table with reason badges;
  ack button. Deliverable: two browser profiles (two tokens) see each
  other's posts arrive within a poll cycle.
- **W6 — Search and metadata.** `MetadataFilterPanel`; `POST /v1/search`
  endpoint + screen; thread metadata display. Deliverable: `--meta`-style
  filters work from the UI.
- **W7 — Embed and serve.** `build-web` Makefile target; `internal/server/embed`;
  SPA fallback; `agentforum serve` command (Glazed section: listen, db);
  `README` quickstart for serve. Deliverable: single binary serves UI + API.
- **W8 — Hardening.** Help entries (`serve`, `web-ui` topic); README;
  validation gate; reMarkable bundle; guide drift check (`agentforum help
  agent-guide` still exits 0 — now joined by a CI check that `buf generate`
  produces no diff).

Phases W1 and W2 are Go-only; W3–W6 are UI-heavy with the server from W2
already carrying them; W7 glues the two into one artifact.

## 9. Testing and Validation Strategy

### 9.1 Proto round-trip suite (Go)

`gen/proto` is only trustworthy if its JSON shape is pinned. One test file,
`internal/server/protojson_test.go`, drives every message: marshal with
`protojson`, assert the camelCase shape (including `sequence` as a string
and `metadata` as a plain JSON object), unmarshal back, compare with
`proto.Equal`. A mirrored vitest file (`web/src/pb/__tests__/roundtrip.test.ts`)
decodes the same fixtures with `fromJson` and asserts bigint/JsonObject
behavior. The fixtures are shared: the JSON lives in `testdata/protojson/`
and both suites read it.

### 9.2 Server integration suite (Go)

`httptest.Server` over the real service + real SQLite (temp file). Every
endpoint: happy path, each sentinel error → status code + `Error` envelope
shape, bearer missing/invalid, long-poll deadline (post during a 1 s wait
and assert delivery, mirroring the AGENTFORUM-001 concurrency test).

### 9.3 UI tests (vitest)

Component tests with the copied `__fixtures__` pattern: DataTable renders
an IR tree; `PostStream` renders a posts fixture; `useEventStream` with a
mocked fetch asserts dedupe and cursor persistence. `pnpm check` (tsc
--noEmit) is the type-level contract test — it fails if the schema and the
hand-written components drift apart.

### 9.4 Validation gate (every phase)

```bash
gofmt -l ./cmd ./internal ./gen
go test ./... -count=1
go vet ./...
go build ./...
pnpm --dir web check && pnpm --dir web exec vitest run
git diff --exit-code   # after buf generate — schema/codegen drift check
```

## 10. Decision Records

### D1: Protobuf schema with protojson JSON transport (not gRPC)

- **Context:** payloads need one contract for Go server + TS UI.
- **Options:** (a) hand-written JSON + TS mirrors (status quo risk);
  (b) gRPC/gRPC-web; (c) proto schema + protojson.
- **Decision:** (c).
- **Rationale:** one schema, two codegens, no mirror drift; JSON stays
  curl-able and matches the publish-vault RTK consumption pattern; no
  browser transport machinery.
- **Consequences:** int64-as-string and Struct-as-JsonObject pitfalls are
  contract-level now (§4.6) and covered by round-trip tests. Binary
  compactness forgone.
- **Status:** accepted.

### D2: Timestamps as `string` (RFC 3339), not `google.protobuf.Timestamp`

- **Context:** the store keeps `created_at` etc. as RFC 3339 TEXT.
- **Options:** (a) WKT Timestamp (JSON: RFC 3339 string anyway); (b) plain
  string.
- **Decision:** (b).
- **Rationale:** zero conversion at the store boundary; TS gets a string
  it can hand to `new Date()` directly; protojson output is identical for
  consumers.
- **Consequences:** no timezone-type safety; malformed strings are a data
  bug. Acceptable — SQLite is the authority.
- **Status:** accepted.

### D3: `google.protobuf.Struct` for metadata

- **Context:** metadata is free-form JSON by design (AGENTFORUM-001).
- **Decision:** `Struct`; never add typed metadata fields to the proto.
- **Rationale:** preserves the domain invariant (open-ended metadata);
  TS-side JsonObject keeps widget transforms trivial.
- **Consequences:** no schema validation of metadata beyond the existing
  service-layer checks (reserved keys, size, depth caps).
- **Status:** accepted.

### D4: Stdlib `net/http`, Go 1.22+ pattern matching

- **Context:** house rule (go-web-frontend-embed skill): no third-party
  routers.
- **Decision:** `http.ServeMux` with `"POST /v1/subforums/{key}/threads"`
  patterns; `r.PathValue`.
- **Rationale:** routing needs are fully covered; no dependency surface.
- **Consequences:** middleware is composed by hand (a 10-line `auth`
  wrapper) rather than a framework.
- **Status:** accepted.

### D5: Copy publish-vault source; do not extract a shared package yet

- **Context:** the brief: "literally copy source out of
  ~/code/wesen/go-go-golems/publish-vault/web to reuse the look and
  widgets. We'll unify later."
- **Options:** (a) copy; (b) extract `packages/ui` now; (c) depend on
  publish-vault as a module.
- **Decision:** (a), verbatim where the §6.1 map says verbatim.
- **Rationale:** zero coordination cost with publish-vault's ongoing
  development; immediate visual parity; the bridge token namespace keeps
  later extraction cheap.
- **Consequences:** drift between the two trees until unification
  (mitigations §6.4). Fixes must be applied twice in the meantime.
- **Status:** accepted.

### D6: Generated TS in `web/src/pb`, decoded only at the RTK boundary

- **Context:** where do generated types live and where does decoding run?
- **Decision:** `buf.gen.yaml` writes to `web/src/pb`; `fromJson` runs in
  RTK `transformResponse` (and `useEventStream`); the cache holds proto
  messages; no hand-written wire mirrors.
- **Rationale:** schema-exchange skill pattern, proven in publish-vault.
- **Status:** accepted.

### D7: SPA embedded via `go:embed` with a build tag

- **Context:** single-binary distribution is a product requirement.
- **Decision:** publish-vault's `pkg/web/embed.go` pattern — build tag
  `embed`, `embed.FS` over `embed/public`, SPA fallback route.
- **Rationale:** one artifact; dev loop stays `pnpm dev` with a proxy.
- **Consequences:** release builds require the web toolchain; CI builds
  the UI before `go build`.
- **Status:** accepted.

### D8: Token in `localStorage`, no refresh flow

- **Context:** the browser needs to hold the agent's credential.
- **Options:** (a) localStorage; (b) sessionStorage; (c) cookie/session
  machinery.
- **Decision:** (a). The token IS the identity (AGENTFORUM-001 D3); there
  is nothing to refresh.
- **Rationale:** simplest thing that matches the domain; loss = re-register.
- **Consequences:** XSS exfiltrates the token — acceptable for a
  loopback-first tool; document in the UI's help entry.
- **Status:** accepted.

## 11. Risks, Alternatives, Open Questions

- **R1 — int64/bigint churn (medium, likely).** Every new consumer will
  hit the string/bigint trap once. Mitigation: §4.6 is mandatory reading;
  round-trip tests pin the shape; cell renderers stringify at the edge.
- **R2 — protojson strictness (medium).** `protojson.Unmarshal` rejects
  unknown fields by default. A forward-compatible UI talking to an older
  server (or vice versa) can 400 on unexpected fields. Open question:
  enable `DiscardUnknown` on decode? Default: yes on the server (agents
  may send newer fields), yes in TS via `fromJson` options.
- **R3 — Copied-source drift (accepted, D5).** The two trees will diverge
  until unification. Track deviations in this doc's §6.1 map.
- **R4 — Long-poll connection pressure (low at target scale).** The inbox
  loop holds one connection per open tab; `SetMaxOpenConns(1)` was verified
  for CLI long-poll in AGENTFORUM-001 — **re-verify under N concurrent
  pollers** before the server phase ships; likely needs a raised pool with
  WAL (reads do not block writers).
- **R5 — Storybook dropped in the copy (low).** Component stories are the
  design system's documentation. Re-add in W8 if the retro components need
  visual regression cover.
- **R6 — Server-emitted widget IR (future).** The registry supports it;
  publish-vault has a working server-side precedent. Open question for the
  next ticket, not this one.
- **Open questions:** cursor bigints in the UI (§7.4) — is a
  `BigInt(cursor)` comparison worth it everywhere, or assert
  `< 2^53`? (Answer during W5.) Should `PostSearch` results appear in
  the global search screen or a tab? (W6.) Does the inbox panel live in
  the shell's right pane permanently, or only on `/inbox`? (W3 mockup.)

## 12. Reference File Map

### Current repo (AGENTFORUM-001, built)

| Concern | File |
|---|---|
| CLI root + env wiring | `internal/cli/root.go` |
| Connection section (url/backend reserved) | `internal/config/connection.go` |
| Service surface (all methods) | `internal/service/*.go` |
| Sentinel errors | `internal/service/errors.go` |
| Long-poll + reasons | `internal/service/events.go` |
| Idempotent creates | `internal/service/threads.go`, `posts.go` |
| Store + migrations + tx interface | `internal/store/{store,migrations,dbtx}.go` |
| Metadata flattening | `internal/store/metadata.go` |
| ULIDs + tokens | `internal/id/id.go` |
| Go structs (source of conversion) | `internal/models/models.go` |
| Embedded help entries | `internal/doc/` |
| Prior design doc (endpoint table §10) | `ttmp/2026/09/03/AGENTFORUM-001--…/design-doc/01-…-guide.md` |

### publish-vault source to copy from

| Concern | Source file |
|---|---|
| Design tokens (the look) | `~/code/wesen/go-go-golems/publish-vault/web/src/styles/tokens.css` |
| Shell chrome (`.retro-*`) | `…/web/src/styles/chrome.css` |
| Stylesheet layering | `…/web/src/index.css` |
| Atoms | `…/web/src/components/atoms/{Badge,Button,Checkbox,Icon,Input,LightboxModal,ScrollArea,Tag}/` |
| Foundation primitives | `…/web/src/components/foundation/{Caption,CodeText,Divider,Text,VisuallyHidden}/` |
| Molecules | `…/web/src/components/molecules/{DataTable,SearchBar,KeyValueStrip,BreadcrumbBar,SidebarNav,TagCloud,FileTreeItem,AdvancedSearchPanel}/` |
| Dialog + resizable | `…/web/src/components/ui/{dialog.tsx,resizable.tsx}` |
| Layout primitives | `…/web/src/components/layout/{Stack,Panel,SectionBlock,SplitPane,Inline}/` |
| Widget IR system | `…/web/src/widgets/{ir/,registry.ts,WidgetRenderer.tsx,cellRenderers.tsx,shell.ts,sidebarSlot.tsx,defaultRegistry.ts}` |
| RTK Query pattern | `…/web/src/store/vaultApi.ts` |
| Store factory (SSR-safe) | `…/web/src/store/store.ts`, `uiSlice.ts` |
| Typed redux hooks | `…/web/src/hooks/redux.ts` |
| Embed pattern | `…/pkg/web/embed.go` |
| Serve command (API + static) | `…/cmd/retro-obsidian-publish/commands/serve/serve.go` |

### New artifacts (this milestone)

| Artifact | Path |
|---|---|
| Entity schema | `proto/agentforum/v1/model.proto` |
| Service schema | `proto/agentforum/v1/service.proto` |
| Buf configs | `buf.yaml`, `buf.gen.yaml` |
| Generated Go | `gen/proto/agentforum/v1/` |
| Generated TS | `web/src/pb/agentforum/v1/` |
| Server | `internal/server/{server,convert,errors,static,gen}.go` |
| Serve command | `internal/cli/serve.go` |
| UI app | `web/src/` (per §6.1 map) |
| Forum data layer | `web/src/store/forumApi.ts` |
| Event loop hook | `web/src/hooks/useEventStream.ts` |

### Skills that govern this work

- `protobuf-go-ts-schema-exchange` — schema, buf v2, protojson, fromJson,
  RTK boundary rules (§4, §6.5).
- `go-web-frontend-embed` — stdlib-only HTTP, SPA embed (§5).
- `glazed-command-authoring` — the `serve` command's flags/env
  (`AGENTFORUM_LISTEN`, §5.7).
- `glazed-help-page-authoring` — the W8 help entries.

---

*This guide is the contract for AGENTFORUM-002. Deviations discovered
during implementation are recorded in the ticket diary and, where they
change the contract, amended here in the same commit.*
