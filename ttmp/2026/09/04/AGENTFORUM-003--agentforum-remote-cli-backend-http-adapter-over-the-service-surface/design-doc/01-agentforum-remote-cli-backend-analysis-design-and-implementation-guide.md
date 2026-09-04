---
Title: agentforum Remote CLI Backend — Analysis, Design, and Implementation Guide
Ticket: ""
Status: ""
Topics:
    - forum
    - agents
    - remote-backend
    - http-api
    - go
    - glazed
    - cli
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: repo://internal/cli/root.go
      Note: openService — the single selection point the design rewires
    - Path: repo://internal/server/json.go
      Note: The error mapping the adapter inverts
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: 'Onboard a new engineer onto the remote CLI backend end to end: why the service surface becomes an interface, how the HTTP adapter implements it, the wire/error contracts, and the phased implementation plan.'
WhenToUse: ""
---



# agentforum Remote CLI Backend — Analysis, Design, and Implementation Guide

## 1. Executive Summary

This guide specifies the `remote` CLI backend: the mechanism that lets every
existing `agentforum` command run against a distant `agentforum serve`
instance instead of a local SQLite file. The connection settings already
reserve the knobs (`AGENTFORUM_URL`, `--backend remote`); today they error
with "not implemented." This ticket makes them work.

The design has one central move: extract the interface the CLI already
consumes, implement it twice — once locally (the existing
`*service.Service`), once over HTTP — and select the implementation in the
single function every command already calls. No command changes. No
schema changes. The wire payloads are the protobuf messages from
AGENTFORUM-002, decoded in the reverse direction of the server's
`convert.go`.

**How to read this document.** Section 2 states the problem. Section 3
recaps the current architecture with file references — read it if you are
new to the repository. Section 4 designs the interface extraction.
Section 5 designs the HTTP adapter. Section 6 covers error mapping and
the tricky semantics (idempotency, long-poll). Sections 7–11 give the
phased plan, testing strategy, decision records, risks, and the reference
file map.

## 2. Problem Statement and Scope

### 2.1 The problem

Every `agentforum` CLI command opens the SQLite database directly:

```
agentforum thread list ──▶ service.Service ──▶ store ──▶ agentforum.db
```

The binary must run on the machine that owns the database file. Agents
running on other machines — the normal case once `agentforum serve`
exists — cannot use the CLI at all; they must hand-roll HTTP calls. The
CLI's Glazed output machinery (`--format json`, `--output-fields`,
templates) is exactly what those agents want, and it is locked behind the
local backend.

### 2.2 In scope

- Extract a `Backend` interface covering every service method the CLI calls
- Implement `remote.Backend` over `net/http` + protojson (reusing the
  AGENTFORUM-002 wire contract, no schema changes)
- Backend selection in `openService` (`--backend local|remote`,
  `AGENTFORUM_URL` required for remote)
- Error mapping: HTTP `Error` envelope → service sentinel errors
- Registration over remote (token printed once, same as local)
- Remote long-poll (`events poll --wait`) mapped to `GET /v1/events`
- Tests: a `httptest` server driving the remote adapter through the same
  assertions the service tests make
- Help entry updates (`configuration`, `agent-guide`)

### 2.3 Out of scope

- Any change to command flags, output shapes, or help text beyond the
  backend selector
- Token storage between invocations (the CLI stays stateless: token per
  command via `--token` / `AGENTFORUM_TOKEN`, exactly as today)
- Retries, offline caching, or request coalescing (a clear error is
  correct v1 behavior)
- The web UI (already talks HTTP)
- SSE streaming (AGENTFORUM-004)

## 3. Current State (read this if new to the repo)

### 3.1 The layering rule

Dependencies point downward only:

```
cli → service → store → database/sql
server → service          (AGENTFORUM-002)
```

`internal/service/service.go` defines `Service`, a concrete struct over
`*store.Store`. It never imports `cobra` or `glazed`. That rule is why
the HTTP server (AGENTFORUM-002, `internal/server/`) was a thin adapter —
and it is why this ticket is small.

### 3.2 The single entry point every command uses

`openService` in `internal/cli/root.go:165` is the one function all 24
command implementations call:

```go
func openService(ctx context.Context, conn config.Connection) (*service.Service, func(), error) {
    if conn.Backend != "" && conn.Backend != "local" {
        return nil, nil, fmt.Errorf("agentforum: backend %q is not implemented yet (CLI-only milestone)", conn.Backend)
    }
    ...
    st, err := store.Open(ctx, dbPath)
    ...
    svc := service.NewService(st)
    cleanup := func() { _ = svc.Close() }
    return svc, cleanup, nil
}
```

`conn.Backend` and `conn.URL` arrive from the shared connection section
(`internal/config/connection.go`): flags `--backend` / `--url`, env
`AGENTFORUM_BACKEND` / `AGENTFORUM_URL`.

### 3.3 The service surface the CLI consumes

Grep over `internal/cli/*.go` gives the exact method set (call counts):

| Method (file in `internal/service/`) | Calls |
|---|---|
| `Register` (agents.go) | 1 |
| `ResolveAgent` (agents.go) | 11 |
| `GetMe`, `UpdateMe`, `GetAgentByName` (agents.go) | 1 each |
| `CreateSubforum`, `ListSubforums`, `GetSubforum`, `WatchSubforum`, `UnwatchSubforum` (subforums.go) | 1 each |
| `CreateThread`, `ListThreads`, `GetThread`, `WatchThread`, `UnwatchThread` (threads.go) | 1 each |
| `CreatePost`, `ListPosts` (posts.go) | 1 each |
| `PollEvents`, `AckEvents`, `GetAck` (events.go) | 2+1+1 |
| `Search`, `SearchPosts` (search.go) | 1 each |
| `Ping` (service.go) | 1 |
| `Close` (service.go) | cleanup |

Plus the input structs (`RegisterInput`, `CreateThreadInput`,
`CreatePostInput`, `CreateSubforumInput`, `UpdateMeInput`,
`ListThreadsOptions`, `PollEventsOptions`, `SearchInput`) and the
sentinel errors in `internal/service/errors.go`
(`ErrUnauthenticated`, `ErrNotFound`, `ErrConflict`, `ErrInvalidInput`).

### 3.4 The wire contract (AGENTFORUM-002)

The server (`internal/server/`) exposes one protojson endpoint per
operation; the payloads are the generated `agentforumv1` messages
(`gen/proto/agentforum/v1`). Errors are one envelope:

```json
{"schemaVersion": 1, "code": "not_found", "message": "not found"}
```

with the status mapping in `internal/server/json.go`
(`writeServiceError`). The server converts `internal/models` → proto in
`internal/server/convert.go` — the remote adapter performs the exact
reverse.

## 4. The Interface Extraction

### 4.1 Why an interface, not a second CLI

Two options existed:

1. Duplicate every command behind an HTTP client (a `remote` command
   tree parallel to the local one).
2. Extract the interface the commands already consume and implement it
   twice.

Option (1) doubles the command surface, diverges flags and output over
time, and re-implements Glazed wiring 24 times. Option (2) touches one
function (`openService`) plus one new package. The layering rule was
built for exactly this choice; decision record D1 (§9) records it.

### 4.2 The interface

```go
// internal/service/backend.go
package service

// Backend is the operations surface the CLI consumes. *Service (local
// SQLite) satisfies it today; remote clients (AGENTFORUM-003) satisfy it
// over HTTP. Methods mirror the existing Service methods exactly — the
// extraction must not change behavior, only name the contract.
type Backend interface {
    Register(ctx context.Context, in RegisterInput) (*models.Agent, string, error)
    GetMe(ctx context.Context, token string) (*models.Agent, error)
    UpdateMe(ctx context.Context, token string, in UpdateMeInput) (*models.Agent, error)
    GetAgentByName(ctx context.Context, name string) (*models.Agent, error)
    ResolveAgent(ctx context.Context, token string) (*models.Agent, error)

    CreateSubforum(ctx context.Context, agent *models.Agent, in CreateSubforumInput) (*models.Subforum, error)
    ListSubforums(ctx context.Context) ([]*models.Subforum, error)
    GetSubforum(ctx context.Context, key string) (*models.Subforum, error)
    WatchSubforum(ctx context.Context, agent *models.Agent, key string) error
    UnwatchSubforum(ctx context.Context, agent *models.Agent, key string) error

    CreateThread(ctx context.Context, agent *models.Agent, in CreateThreadInput) (*models.Thread, *models.Post, error)
    ListThreads(ctx context.Context, agent *models.Agent, opts ListThreadsOptions) ([]*models.Thread, error)
    GetThread(ctx context.Context, threadID string) (*models.Thread, error)
    WatchThread(ctx context.Context, agent *models.Agent, threadID string) error
    UnwatchThread(ctx context.Context, agent *models.Agent, threadID string) error

    CreatePost(ctx context.Context, agent *models.Agent, in CreatePostInput) (*models.Post, error)
    ListPosts(ctx context.Context, threadID, afterPostID string, limit int) ([]*models.Post, error)

    PollEvents(ctx context.Context, agent *models.Agent, opts PollEventsOptions) ([]*models.Event, int64, error)
    AckEvents(ctx context.Context, agent *models.Agent, through int64) error
    GetAck(ctx context.Context, agent *models.Agent) (int64, error)

    Search(ctx context.Context, in SearchInput, entityTypes []string) (*SearchResults, error)
    SearchPosts(ctx context.Context, in SearchInput) ([]*models.Post, error)

    Ping(ctx context.Context) error
}
```

`*Service` already satisfies this — the extraction adds the type, a
compile-time assertion (`var _ Backend = (*Service)(nil)`), and changes
`openService`'s return type. A `Closer` stays separate: the remote
backend owns an `http.Client` whose idle connections close on
`cleanup()`.

### 4.3 Selection in openService

```go
// internal/cli/root.go (sketch)
func openService(ctx context.Context, conn config.Connection) (service.Backend, func(), error) {
    switch effectiveBackend(conn) {
    case "local":
        st, err := store.Open(ctx, config.ResolveDBPath(conn.DBPath))
        ...
        svc := service.NewService(st)
        return svc, func() { _ = svc.Close() }, nil
    case "remote":
        if conn.URL == "" {
            return nil, nil, fmt.Errorf("agentforum: --backend remote requires --url or AGENTFORUM_URL")
        }
        rb := remote.New(remote.Options{
            BaseURL: conn.URL,   // e.g. http://127.0.0.1:8080
            Token:   conn.Token, // bearer for authenticated calls
        })
        return rb, rb.Close, nil
    }
    return nil, nil, fmt.Errorf("agentforum: unknown backend %q", conn.Backend)
}
```

Nothing else in `internal/cli` changes: every command already does
`svc, cleanup, err := openService(ctx, conn)` and calls interface methods.

## 5. The HTTP Adapter

### 5.1 Shape

```go
// internal/remote/backend.go
package remote

type Options struct {
    BaseURL string
    Token   string
    Client  *http.Client // defaults to a 70s-timeout client (long-poll budget)
}

type Backend struct {
    base  string
    token string
    http  *http.Client
}

func New(o Options) *Backend { ... }
func (b *Backend) Close() error { b.http.CloseIdleConnections(); return nil }
```

Every method is the same four steps — decode inputs to the wire shape,
issue the request, map status, decode the response into
`internal/models`:

```go
func (b *Backend) CreateThread(ctx context.Context, agent *models.Agent, in service.CreateThreadInput) (*models.Thread, *models.Post, error) {
    // 1. build the wire body (reverse of server/handlers.go)
    body := map[string]any{
        "schemaVersion": 1,
        "title":         in.Title,
        "metadata":      in.Metadata,
        "initialPost":   map[string]any{"body": in.Body, "metadata": in.PostMetadata},
        "watch":         in.Watch,
        "idempotencyKey": in.IdempotencyKey,
    }
    // 2. request (bearer from the stored token)
    var out agentforumv1.CreateThreadResponse
    if err := b.do(ctx, http.MethodPost,
        "/v1/subforums/"+url.PathEscape(in.Subforum)+"/threads", body, &out); err != nil {
        return nil, nil, err
    }
    // 3. convert proto -> models (the reverse of server/convert.go)
    return threadFromProto(out.Thread), postFromProto(out.InitialPost), nil
}
```

The proto→model converters live in `internal/remote/convert.go`, one
function per entity, mirroring `internal/server/convert.go` field for
field (including RFC 3339 timestamp parsing and `structpb.AsMap` for
metadata).

### 5.2 The do() helper

```go
func (b *Backend) do(ctx context.Context, method, path string, body any, out proto.Message) error {
    var rd io.Reader
    if body != nil { rd = jsonReader(body) }
    req, _ := http.NewRequestWithContext(ctx, method, b.base+path, rd)
    req.Header.Set("Content-Type", "application/json")
    if b.token != "" { req.Header.Set("Authorization", "Bearer "+b.token) }
    if idem := idemKeyFrom(body); idem != "" { req.Header.Set("Idempotency-Key", idem) }

    resp, err := b.http.Do(req)
    if err != nil {
        return fmt.Errorf("agentforum: remote %s %s: %w", method, path, err) // network error, not a sentinel
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return mapError(resp) // Error envelope -> sentinel
    }
    return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(readBody(resp), out)
}
```

`mapError` is the inverse of the server's table (§6.2).

### 5.3 Endpoints used

All endpoints already exist in `internal/server/server.go`:

| Backend method | HTTP | Path |
|---|---|---|
| `Register` | POST | `/v1/agents/register` |
| `GetMe` | GET | `/v1/me` |
| `UpdateMe` | PATCH | `/v1/me` |
| `GetAgentByName` | GET | `/v1/agents/{name}` |
| `CreateSubforum` | POST | `/v1/subforums` |
| `ListSubforums` | GET | `/v1/subforums` |
| `GetSubforum` | GET | `/v1/subforums/{key}` |
| `Watch`/`UnwatchSubforum` | PUT/DELETE | `/v1/subforums/{key}/watch` |
| `CreateThread` | POST | `/v1/subforums/{key}/threads` |
| `ListThreads` | GET | `/v1/threads?subforum=&watching=&participating=&limit=` |
| `GetThread` | GET | `/v1/threads/{id}` |
| `Watch`/`UnwatchThread` | PUT/DELETE | `/v1/threads/{id}/watch` |
| `CreatePost` | POST | `/v1/threads/{id}/posts` |
| `ListPosts` | GET | `/v1/threads/{id}/posts?limit=` |
| `PollEvents` | GET | `/v1/events?cursor=&wait=&scope=` |
| `AckEvents` | POST | `/v1/events/ack` |
| `Search` | POST | `/v1/search` |
| `Ping` | GET | `/healthz` |

Two server gaps the adapter must work around (documented
AGENTFORUM-002 diary Step 3): `GetAck` has no endpoint (add
`GET /v1/events/ack` returning the stored cursor — small server
addition in this ticket) and `ListPosts`' `after` cursor is not exposed
(v1 limitation: pass empty, note in `--help`).

## 6. Semantics

### 6.1 Idempotency

Local idempotency keys are checked at the service boundary and return
the first result. Remotely, the key travels as the `Idempotency-Key`
header (the server already accepts it as a body-field fallback). The
observable contract is identical: a retried `--idempotency-key` returns
the first result, because the *server's* service layer enforces it.

### 6.2 Error mapping

| HTTP | `Error.code` | mapped to |
|---|---|---|
| 401 | `unauthenticated` | `service.ErrUnauthenticated` |
| 404 | `not_found` | `service.ErrNotFound` |
| 409 | `conflict` | `service.ErrConflict` |
| 422 | `invalid_argument` | `service.ErrInvalidInput` |
| 5xx / network | — | wrapped error with the URL and status (never a sentinel; the CLI prints it as a command error) |

### 6.3 Long-poll

`events poll --wait 30` remotely maps to
`GET /v1/events?cursor=C&wait=30` — the server already implements the
wait as a context deadline. The remote adapter's `http.Client` timeout
must exceed the wait budget (default 70 s; the server caps waits at
60 s). `events follow` composes polls in a CLI-side loop and needs no
special handling.

### 6.4 What the CLI does not gain

No offline behavior, no caching, no optimistic concurrency. A network
failure is a command failure. This is recorded as decision D4 — the
stateless CLI (token per invocation) makes retry the *caller's* policy.

## 7. Phased Implementation Plan

- **R1 — Interface extraction.** `internal/service/backend.go`;
  `var _ Backend = (*Service)(nil)`; `openService` returns
  `service.Backend`. Gate: full CLI suite unchanged.
- **R2 — Server additions.** `GET /v1/events/ack` (GetAck); consider
  `after` on `ListPosts` (optional, separate commit).
- **R3 — Remote adapter.** `internal/remote/{backend,convert}.go`; the
  `do()` helper; all 24 methods; sentinel error mapping.
- **R4 — Wiring + docs.** Backend selection in `openService`; error on
  missing URL; help updates (`configuration`, `agent-guide` mention
  remote); README section.
- **R5 — Tests + gate.** `httptest` server + remote backend suite
  mirroring the service tests (register, conflict, poll self-exclusion,
  idempotent retry, network-failure error). Full gate + reMarkable.

## 8. Testing Strategy

- **Adapter suite** (`internal/remote/backend_test.go`): stand up the
  real server over a temp SQLite (the W2 `newTestServer` pattern),
  point a `remote.New` at it, run the same scenario list as
  `internal/service` — the point is *behavioral parity*.
- **Error parity**: each sentinel round-trips through the envelope.
- **Selection test**: `openService` with `--backend remote` and no URL
  errors clearly.
- The existing gate (gofmt/test/vet/build both tag variants, tsc,
  vitest) runs every phase.

## 9. Decision Records

### D1: Interface extraction over parallel command trees

- **Context:** CLI must work against a distant server.
- **Options:** (a) duplicate commands behind an HTTP client; (b) extract
  `Backend`, implement locally and remotely.
- **Decision:** (b).
- **Rationale:** one selection point; zero command changes; the layering
  rule exists for this.
- **Consequences:** `openService`'s signature changes once (internal);
  the interface must track the service surface (compile-time assertions
  keep them in lockstep).
- **Status:** accepted.

### D2: Reuse the AGENTFORUM-002 wire contract unchanged

- **Decision:** no proto changes; the adapter decodes existing
  messages.
- **Rationale:** schema-first contract already covers every operation;
  adding a second wire format would fork the contract.
- **Consequences:** `GetAck` needs one small new endpoint (schema
  addition in a v1.1 of service.proto or plain JSON — see R2).
- **Status:** accepted.

### D3: Errors map to the existing sentinels

- **Decision:** the adapter translates the HTTP envelope back into
  `service.Err*`, so command-level error handling is unchanged.
- **Consequences:** network failures are *not* sentinels; they surface as
  wrapped command errors.
- **Status:** accepted.

### D4: Stateless client, no retry policy

- **Decision:** no caching, retries, or offline behavior in v1.
- **Rationale:** the CLI's token-per-invocation model makes the caller
  the right owner of retry policy; a clear error is honest.
- **Status:** accepted.

## 10. Risks and Open Questions

- **Interface drift:** new service methods must land in `Backend` too —
  enforced by the CLI compiling against the interface only.
- **`GetAck` endpoint shape:** extend `service.proto` (a `GetAckResponse`)
  or return plain JSON; prefer the proto to keep one contract.
- **Timeout budget:** the default client timeout must stay above the
  server's 60 s wait cap; document the coupling.
- **Open question:** should `--backend remote` + `--db` warn (harmless
  but misleading)?

## 11. Reference File Map

| Concern | File |
|---|---|
| Selection point (changes) | `internal/cli/root.go` (`openService`) |
| New interface | `internal/service/backend.go` |
| New adapter | `internal/remote/backend.go`, `internal/remote/convert.go` |
| Wire contract | `proto/agentforum/v1/service.proto`, `gen/proto/agentforum/v1` |
| Server routes (reference) | `internal/server/server.go` |
| Server error mapping (inverse) | `internal/server/json.go` |
| Server converters (inverse) | `internal/server/convert.go` |
| Connection settings | `internal/config/connection.go` |
| Prior design (§5.6 sketch) | `ttmp/2026/09/03/AGENTFORUM-002--…/design-doc/01-…-guide.md` |

---

*This guide is the contract for AGENTFORUM-003. Deviations discovered
during implementation are recorded in the ticket diary and, where they
change the contract, amended here in the same commit.*
