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

## Step 2: W1 — proto schema, buf codegen, protojson round-trip tests

This step implemented the schema-first payload contract. Two `.proto`
files define every entity and request/response message; `buf generate`
produces Go and TypeScript from them; golden fixtures pin the protojson
wire shape in tests. No server or UI code yet — the contract comes first so
every later phase codes against generated types only.

### Prompt Context

**User prompt (verbatim):** "implement, commit at appropriate intervals and keep a detailed diary as you work (using the diary format from the skill). Print out a brutalist work slip with the plan / different phases for the ticket. then before stsarting a phase, plrint a split about the phase, and print one when the phase is done."

**Assistant interpretation:** Implement AGENTFORUM-002 phases W1–W8, committing per phase, keeping the diary current, printing plan + pre/post phase slips on the thermal printer.

**Inferred user intent:** Execute the design doc end to end with the same per-phase discipline as AGENTFORUM-001 (slip before, implement, gate, commit, diary, slip after).

### What I did
- Wrote `proto/agentforum/v1/model.proto` (Agent, Subforum, Thread, Post, Event, SearchHit + EventType/Reason/Scope enums) and `service.proto` (one request/response pair per endpoint, `Error` envelope, `schema_version` on top-level messages).
- Put `buf.yaml` in `proto/` (module root), `buf.gen.yaml` at repo root; `buf generate proto` emits `gen/proto/agentforum/v1/*.pb.go` and `web/src/pb/agentforum/v1/*_pb.ts` via remote plugins.
- Wired `//go:generate buf generate proto` through a root `gen.go` (package agentforum) so plain `go generate ./...` regenerates both outputs.
- Added `testdata/protojson/` golden fixtures (event, thread, create-thread request) and `internal/server/protojson_test.go`: camelCase assertions, int64-as-string (`"sequence":"42"`, `"nextCursor":"43"`), Struct-as-plain-object, byte-stable re-marshal, and int64 accepting both `"44"` and `45` on decode.

### Why
The design (§4) makes `.proto` the single source of truth for Go server and TS UI. W1 exists to pin the contract — including the two type traps — before any consumer exists.

### What worked
- `buf generate` with remote plugins worked first try (buf 1.55.1, network available); lint clean under `STANDARD` ruleset.
- `go mod tidy` pulled `google.golang.org/protobuf v1.36.12` without conflicts.

### What didn't work
- `buf lint` at repo root failed with `import "agentforum/v1/model.proto": file does not exist`. Cause: the buf module root is the directory containing `buf.yaml`; with `buf.yaml` at the root, module paths carry the `proto/` prefix and the import (and generated output paths) would be `proto/agentforum/v1/...`. Fix: moved `buf.yaml` into `proto/` so module paths are `agentforum/v1/...`; generation runs as `buf generate proto` from the root, keeping `gen/proto/agentforum/v1` and `web/src/pb/agentforum/v1` clean (the exact pitfall the schema-exchange skill warns about).
- First test draft used `msg.String()` on the `proto.Message` interface — compile error `type proto.Message has no field or method String` (new protobuf API interface only has `ProtoReflect`). Fixed with `%v` in `t.Errorf` (concrete generated types implement Stringer).
- Left an unused `"fmt"` import after that fix; `go test` caught it, removed.

### What I learned
- `protojson.Unmarshal` accepts both JSON strings and numbers for int64 — encoded into a test (`TestUnmarshalAcceptsInt64AsStringAndNumber`) so the leniency is a pinned contract, not an accident.
- protojson re-marshal of a decoded golden fixture is byte-stable (field order is deterministic), which makes golden-file comparison practical without canonicalization.

### What was tricky to build
- The module-root decision (buf.yaml location) interacts with three things at once: import paths inside `.proto` files, `paths=source_relative` Go output layout, and the TS import prefix. Moving the module root into `proto/` resolved all three consistently; the alternative (root module + `proto/`-prefixed imports) would have produced `gen/proto/proto/...`.

### What warrants a second pair of eyes
- The `DiscardUnknown: true` choice in the test's fixture decode (design R2 said "decide during W1"): the round-trip test opts in so fixtures stay valid when the schema grows. The server decode (W2) must make the same call explicitly.
- `SearchHit.score` is `double` — fine for LIKE-based ranking, but confirm nothing downstream assumes ordering semantics.

### What should be done in the future
- W2: HTTP server on top of the contract.
- The vitest mirror of the round-trip suite lands with the web scaffold (W3) when `web/package.json` exists; fixtures already shared via `testdata/protojson/`.

### Code review instructions
- Start: `proto/agentforum/v1/service.proto` (message inventory), then `internal/server/protojson_test.go`.
- Validate: `buf lint proto && buf generate proto && git diff --exit-code gen web/src`; `GOWORK=off go test ./... -count=1`.

### Technical details
- Commit: 8d161dd.
- Codegen command: `buf generate proto` (from repo root); remote plugins `buf.build/bufbuild/es` (target=ts, import_extension=none) and `buf.build/protocolbuffers/go` (paths=source_relative).

## Step 3: W2 — HTTP server with protojson transport

This step built the `/v1` API as a pure adapter over the existing service
layer: routing, bearer authentication, protojson encoding, error mapping,
and long-poll deadline wiring. The store gained four batched queries for
response denormalization. The full endpoint surface is covered by an
httptest integration suite plus a live curl walkthrough.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Same directive continuing — implement phase W2 with per-phase commit, diary, and slips.

**Inferred user intent:** See Step 2.

### What I did
- `internal/server/server.go`: `Server{svc, router}`, `New()` wiring the full route table (Go 1.22+ patterns like `POST /v1/subforums/{key}/threads`), `auth()` middleware (bearer → `ResolveAgent`, agent+token in context), `healthz`.
- `internal/server/json.go`: `writeProtoJSON` (protojson, camelCase), `decodeProtoJSON` (1 MiB cap, `DiscardUnknown: true` — resolving open question R2), `writeServiceError` (sentinels → 401/404/409/422/500 with the `Error` proto envelope).
- `internal/server/convert.go`: the single `internal/models` → `agentforumv1` boundary; `threadView` batches watching/participating/stats per requesting agent.
- `internal/server/handlers.go`: every endpoint from the design's §5.2 table, including `Idempotency-Key` header fallback on creates and the long-poll cap (`maxWaitSeconds=60`, `context.WithTimeout`).
- Store additions: `ThreadStats`, `ThreadTitles` (threads.go), `AgentNames` (agents.go), `SubforumThreadCounts` (subforums.go) — one grouped query each, never N+1.
- `internal/server/server_test.go`: httptest suite (register/auth flow, full forum flow over HTTP, idempotent retry, concurrent long-poll, unknown-field acceptance).

### Why
Design §5: the server is an adapter; business rules stay in the service. This phase makes the W1 contract reachable over HTTP, which W3–W7 (web UI) build on.

### What worked
- The whole `/v1` surface wrapped the existing service methods with zero service-layer changes — the "server is just an adapter" claim from the design held exactly.
- Live curl walkthrough matched the design's wire contract verbatim: `postCount:"1"`, `nextCursor:"2"` empty-poll marker, `Error` envelope with `code:"unauthenticated"`.
- Long-poll test reuses the service-level pattern (goroutine poll + delayed post + elapsed assertion).

### What didn't work
- `protojson.Message` does not exist — the correct parameter type is `proto.Message` (google.golang.org/protobuf/proto). Also `UnmarshalOptions.UnmarshalReader` does not exist in protobuf v1.36; fixed by reading the body (`io.ReadAll` + `io.LimitReader`) then `Unmarshal`.
- `TermFilter` is `{Keys []string, Value}` (multi-key), not `{Key, Value}` — the search handler builds `Keys: []string{k}` per metadata entry.
- `CreateThreadInput` field is `Metadata`, not `ThreadMetadata`.
- First long-poll test asserted 3 events from cursor 0 — wrong: `PollEvents` returns immediately when *any* eligible events exist (bob had 2 already). Fixed by first draining (wait=0), then long-polling from the returned cursor; the blocked poll then delivers exactly the third event.
- Attempted an out-of-repo smoke binary under `/tmp` — `use of internal package ... not allowed`; internal packages only build inside the module. Fixed with a temporary in-repo `cmd/afsmoke` (deleted before commit).

### What I learned
- A poll that already has eligible events returns immediately by design ("eligible events are always delivered before being advanced past") — long-poll blocking only happens when caught up. Testing long-poll therefore requires draining first; this is the inbox contract, not an implementation quirk.

### What was tricky to build
- Denormalization without N+1: every list/poll handler needs per-row display fields (authorName, threadTitle, postCount, watching, participating). Solved with four batched store queries and a `threadViews` struct computed once per request; the poll handler batches actor ids and thread ids across the returned events.

### What warrants a second pair of eyes
- `decodeProtoJSON`'s `DiscardUnknown: true` (R2) is now a shipped contract: clients may send unknown fields. If strictness is wanted later it is a behavior change, not a bug fix.
- Search maps `subforums[0]` to the single `SearchInput.Subforum` (proto allows repeated; service supports one) — documented in-code, but the proto implies more than the service does.
- `handleListPosts` ignores the proto's `after` cursor (proto `ListPostsRequest.thread_id`/`limit` are taken from path/query instead) — pagination beyond `limit` is not yet exposed over HTTP.

### What should be done in the future
- Expose post pagination (`after` cursor) in the HTTP API when the UI needs it.
- Consider returning `nextCursor` on `GET /v1/threads` (keyset pagination) if thread lists grow.

### Code review instructions
- Start: `internal/server/server.go` (route table + auth), then `handlers.go` (one handler per endpoint), `convert.go` (the only models→proto boundary).
- Validate: `GOWORK=off go test ./internal/server/ -count=1`; `gofmt -l internal; go vet ./...`.

### Technical details
- Commit: ac4c9e6.
- Live walkthrough (loopback :18099): register → `af_…` token; create subforum (201); create thread with `Idempotency-Key` header → 201 with `postCount:"1"`, `watching:true`, `participating:true`, `authorName:"smoke"`; poll self-exclusion → `{"schemaVersion":1,"nextCursor":"2"}`; bad token → `{"code":"unauthenticated", …}`.

## Step 4: W3 — web scaffold (publish-vault copy), forumApi, shell, register flow

This step materialized the design's copy map: 64 files copied from
publish-vault/web (styles, atoms, foundation, molecules, layout, ui, the
whole widget IR tree, hooks, lib, store factory), adapted where the map
says adapt, and extended with the forum data layer and first screens. The
UI was verified end-to-end in a real browser against the W2 server.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Same directive — implement phase W3 with per-phase commit, diary, and slips.

**Inferred user intent:** See Step 2.

### What I did
- Executed the §6.1 copy: `styles/` + `index.css`, `components/{atoms,foundation,ui,layout}`, 7 molecules, `widgets/` whole tree, `hooks/redux.ts`, `lib/{utils,highlightLanguages}.ts` + `vendor/highlight-languages`, `store/store.ts` verbatim.
- Adapted: `uiSlice.ts` (activeNoteSlug → activeThread), `defaultRegistry.ts` (dropped the 4 vault-only adapters: NoteHtml, FrontmatterPanel, NoteCard, BacklinksPanel), `vite.config.ts` (react + tailwind v4, `/v1` proxy to :8080, no manus/jsx-loc plugins), `package.json` (deps per §6.6; `clsx ^2.1.1` — `^2.2.1` does not exist).
- Wrote `store/forumApi.ts` (RTK Query; `fromJson` in transformResponse; token in localStorage), `types/index.ts` shim (FileNode/TagCount widget props only), `main.tsx`, `App.tsx` (auth gate), `ForumShell`, `ForumSidebar`, `RegisterScreen`, `SubforumListScreen`, `ThreadListScreen` (DataTable molecule).
- Added `Server.Router()` accessor (used by the smoke host; W7 embedding will reuse it).
- Browser verification (Playwright): register screen → registered as "browser-agent" → ForumShell with sidebar (Engineering, count 2) → thread list through DataTable (titles, post counts, timestamps, breadcrumbs).
- Deleted vault-fixture tests (`defaultRegistry.test.ts`, `shell.test.ts` — they read reader-page/recent-page fixtures exercising vault widgets).

### Why
Design §6: copy now, unify later. W3 proves the copied look + widgets work against the real API and the generated proto types.

### What worked
- The verbatim-copied components compiled after exactly three mechanical fixes (see below); the retro look and chrome classes needed zero changes.
- The vitest round-trip suite reads the same `testdata/protojson` fixtures as the Go suite — both languages now assert identical wire bytes, closing the W1 follow-up.
- Browser flow worked first try after typecheck: register → token → shell → real subforum/thread data.

### What didn't work
- Page files initially imported atoms with 3× `../` (pages sit at `src/components/pages/X/`, so 2× is correct) — ~25 "Cannot find module" errors, fixed with sed.
- `lib/highlightLanguages.ts` imports `src/vendor/highlight-languages/*` which was not in the copy map — copied it (map gap, noted).
- protoc-gen-es v2.14 with `target=ts` generates pure types + `…Schema` descriptor constants; there is NO static `Message.fromJson()` method. Correct API is the standalone `fromJson(Schema, json)` function (and `toJson(Schema, msg)`). The skill's `ShowList.fromJson(...)` example reflects the older class-based codegen. Both forumApi and the test were switched to the function form.
- `clsx ^2.2.1` does not exist on npm (latest 2.1.1) — `ERR_PNPM_NO_MATCHING_VERSION`.
- tsconfig lacked `"target"` — `Set` iteration in a copied widget test demanded `es2015+`; set `target: "es2022"`.
- `Caption` only accepts `as="span"|"h3"|"h4"|"div"` — not `h1`.
- Playwright clicked a 1Password status div when given a stale snapshot ref; re-finding the button by role fixed it (operator error, not code).

### What I learned
- The v2 codegen split (types vs runtime descriptors) makes `import type {...}` + `import { …Schema }` the standard pattern; `GenMessage` extends `DescMessage`, which is why `fromJson(schema, json)` accepts the generated `…Schema` const.
- publish-vault's component tree is genuinely portable: the only structural work was the pages/organisms layer, which the design already flagged as "write fresh."

### What was tricky to build
- The RTK `getMe` transform: the wire response is `GetMeResponse{agent}` but the hook should expose the `Agent`. Solved by decoding the full response and returning `.agent` inside `transformResponse` (cache still holds a proto message). A 401 there routes to the register screen via `App`'s error check.
- Token lifecycle without a refresh flow: register stores the token and reloads (simplest correct remount); the shell's forget button clears it. Documented per D8.

### What warrants a second pair of eyes
- `forumApi.getMe`'s transformResponse has a defensive fallback (`r as Agent`) if the envelope shape changes — verify it never masks a schema drift (the round-trip tests should catch it first).
- The `types/index.ts` shim keeps copied molecules compiling; if it grows beyond widget props it violates the "no wire mirrors" rule — watch it in review.

### What should be done in the future
- W4: thread detail screen (PostStream organism + composer), IR-driven thread list, watch toggles in the UI.
- Revisit `pollEvents` query once the inbox loop (useEventStream) lands — the query form may be redundant.

### Code review instructions
- Start: `web/src/store/forumApi.ts` (data layer), `web/src/App.tsx` (auth gate), then `ForumShell`/`ForumSidebar`.
- Validate: `cd web && pnpm check && pnpm test && pnpm build`; browser flow per diary above.

### Technical details
- Commit: 5a643c1. Copied 64 source files (+ vendor tree + generated pb already present). Build: 1686 modules, dist ≈ 484 kB JS (152 kB gzip) + 40 kB CSS.

## Step 5: W4 — core screens, markdown + MathJax, flat restyle (user feedback mid-phase)

This step delivered the core screens (thread detail with composer and
watch toggles, IR-driven thread list) and, after user feedback mid-phase,
restyled the UI to flat HN-style austerity inside the retro system and
added markdown + MathJax rendering to post bodies, reusing publish-vault's
MathJax loader and code-enhancement machinery.

### Prompt Context

**User prompt (verbatim):** "no brutalist dropshadows, and more compact / concise (think HN style austerity, but stick with the style overall. make sure markdown and mathjax work). /tmp/pi-clipboard-6a5d8531-4a53-4973-9721-3f1314993b30.png"

Followed by: "remember you can also copy stuff from publish-vault and such"

**Assistant interpretation:** Keep the retro monochrome language but remove all drop shadows, tighten spacing to Hacker-News-like density, and render post bodies as markdown with MathJax math; prefer copying proven publish-vault code over writing new.

**Inferred user intent:** The first browser look was too heavy (boxed posts, hard shadows, generous padding). The user wants an austere, information-dense reading surface that still reads as the same retro system, with full markdown/math support for agent-authored content.

### What I did
- ThreadDetailScreen (PostStream + composer with per-submit idempotency keys, watch/unwatch, breadcrumbs, metadata strip); ThreadListScreen rebuilt as widget IR (cell specs, navigate `ActionSpec` with `${row.id}` interpolation, WidgetRenderer + defaultRegistry); `getThread` endpoint.
- Copied `lib/mathjax.ts` from publish-vault verbatim (TeX→SVG direct API, lazy font-range loading, `typesetTeX`/`ensureMathStyles`).
- Extracted `enhanceCodeBlocks` + `addCopyButtons` from publish-vault's `noteEnhancements.ts` into PostStream's `codeEnhancements.ts` (mermaid/embeds/anchors dropped — vault-specific); styles already in copied prose.css.
- New `lib/markdown.ts` (math extraction → marked → DOMPurify) and `MarkdownBody.tsx` (sanitized HTML + placeholder swap for typeset TeX; memo is load-bearing like NoteBody).
- Restyle: all `box-shadow`s removed from chrome.css (window/inset/btn/btn-primary/search); tighter paddings (btn `2px 8px`/11px, menubar 24px, tree items `1px 6px`/11px, window-title `2px 6px`/11px); PostStream → flat divider-separated list with one-line meta; subforum rows flat with dividers; screen paddings p-3/p-4.
- Browser-verified with a markdown+math post: 2 MathJax SVGs (inline + display), bold/list rendered, Go code block hljs-highlighted with copy button, "Costs $5" left as prose by the delimiter edge rule.

### Why
The feedback arrived between the W4 build and its commit, so W4 landed as one commit containing both the core screens and the restyle (`fdcdd9b`).

### What worked
- Copying publish-vault's MathJax module verbatim: the whole typeset path (lazy font ranges, retry handling, SVG output with `fontCache: "local"`) worked first try against marked-produced HTML.
- The `$5`-stays-prose rule (math delimiters require non-space content edges) held in the live test.
- DOMPurify with `ADD_ATTR: ["data-af-math"]` keeps the placeholder attributes alive through sanitization.

### What didn't work
- The first W4 commit attempt silently died: `pkill -f af-serve2` matched the *shell running the command itself* (the pattern string appears in the shell's own command line), killing the session before the gate/commit ran. Same trap again later with `kill $(pgrep -f afserveW4)`. Fix: `pkill -x <exact-name>`. Recorded here because it will bite again otherwise.
- `@highlight-languages` path alias from publish-vault's tsconfig was dropped in my adapted tsconfig — first used a relative import, which is fine without SSR; the alias only exists for the SSR module swap.
- Two more import-depth slips in new files (PostStream at `src/components/organisms/X/` → lib is 3 up). The rule that finally stuck: count from the file to `src/`, not to `components/`.

### What I learned
- MathJax 4 direct API + `fontCache: "local"` composes cleanly with `dangerouslySetInnerHTML` content as long as the host is memoized (inner re-renders destroy SVG nodes silently).
- marked v18's `parse` is sync when called with `{async: false}`; the default return type is `string | Promise<string>` and needs the cast.

### What was tricky to build
- Ordering inside MarkdownBody's effect: hljs/copy-button enhancement and math typesetting both mutate the same subtree. Resolved by running code enhancement first (placeholders are inert spans that survive hljs) and swapping math nodes after; each pass is idempotent and cancellation-guarded.
- Math inside fenced code blocks is not protected (extraction runs before markdown parsing) — documented as a v1 limitation in `lib/markdown.ts`.

### What warrants a second pair of eyes
- DOMPurify default profile on agent-authored markdown: raw HTML is sanitized but the allowlist is generous. If untrusted agents ever post, tighten the profile.
- The math regexes handle `$…$`/`$$…$$`/`\(...\)`/`\[...\]` — escaped-dollar (`\$`) prose is NOT handled and would still match as a delimiter.

### What should be done in the future
- W5: inbox screen with the `useEventStream` long-poll loop.
- Protect fenced code from math extraction (split on fences first).
- Composer preview (render markdown live before posting).

### Code review instructions
- Start: `web/src/components/organisms/PostStream/MarkdownBody.tsx` (the render pipeline), `web/src/lib/markdown.ts` (delimiter rules), then `chrome.css` (the restyle diff).
- Validate: `cd web && pnpm check && pnpm test && pnpm build`; browser flow per diary above.

### Technical details
- Commit: fdcdd9b (includes both the core screens and the restyle). Deps added: `marked`, `@mathjax/src`, `@mathjax/mathjax-newcm-font`, `dompurify` (+ types).
- Live check result: `{mathCount: 2, strongCount: 1, liCount: 2, codeHighlighted: 1, copyBtns: 1, dollarOk: 1}`.

## Step 6: W5 — inbox screen (unified event stream over long-poll)

This step delivered the UI face of the unified inbox: a `useEventStream`
hook that keeps one forward-only cursor per agent and long-polls
`GET /v1/events`, plus the InboxScreen that renders the stream flat with
reason badges and a durable ack. Verified live with two agents: the
inbox cursor advanced 2 → 5 while the page was open as the other agent
posted. Screenshots for the diary and the future project report are
archived in `screens/`.

### Prompt Context

**User prompt (verbatim):** "take screenshots as you go for the diary, and for our future project report."

(Following the phase directive from Step 2; the screenshot instruction applies from here on.)

**Assistant interpretation:** Continue W5 (inbox screen) and capture browser screenshots at each meaningful state, archiving them in the ticket for the diary and the eventual project report.

**Inferred user intent:** The visual record of the UI evolution is part of the deliverable — the future vault report should be able to show the register screen, thread lists, markdown/math rendering, and the live inbox without re-running anything.

### What I did
- `web/src/hooks/useEventStream.ts`: cursor persisted per agent (`agentforum.cursor.<agentId>`, bigint-safe), `wait=25` long-poll loop with a 500 ms pause between polls, at-least-once delivery with dedupe by `sequence`, connected/error state, cancellation-safe teardown; `useAckEvents` for the durable ack.
- `web/src/components/pages/InboxScreen/InboxScreen.tsx`: flat list (reason word with tone — involved/watching/subforum —, actor, `subforum/thread`, timestamp), live indicator, cursor readout, "ack through N" button, click-through to the source thread. Route wired in App.tsx (replacing the W3 stub).
- Live two-agent verification (browser as bob, curl as alice): bob's inbox showed the two pre-existing events (cursor 2, reason "watching"); alice then replied in the watched thread and started a new thread in the watched subforum; both arrived within one poll cycle, cursor advanced to 5, reasons rendered as "watching" and "subforum" respectively.
- Screenshots archived to `ttmp/…/screens/`: `w3-register-screen.png`, `w3-subforum-list.png`, `w4-thread-list-ir.png`, `w4-thread-detail-markdown-math.png`, `w4-markdown-math-verified.png`, `w5-inbox-live.png`, `w5-inbox-live-update.png`.

### Why
Design §7.4: the inbox is the UI face of the cursor contract; the loop is a hook (not an RTK endpoint) because long-poll responses are batches with a cursor, not cacheable query results.

### What worked
- The full pipeline held end to end: proto events (bigint sequences) → long-poll → fromJson → bigint cursor persistence → flat rendering; both reason types displayed with distinct tones.
- Event click-through to the thread (and the thread's MathJax rendering) worked in the same navigation chain as the inbox update.

### What didn't work
- `EventReason.EVENT_REASON_WATCHING` does not exist in the generated TS: protoc-gen-es strips the enum-value prefix (`EventReason.WATCHING`). Also enums are runtime values, so `import type` failed — switched to a value import.
- A `find /` for the playwright output dir was aborted (user pointed out the screenshots were in the repo root — playwright's relative paths resolve to the invocation cwd). Moved them from there; noted for future phases.

### What I learned
- protoc-gen-es enum naming: `EVENT_REASON_WATCHED_SUBFORUM = 3` becomes `EventReason.WATCHED_SUBFORUM` — the common prefix is dropped. Worth remembering for every future enum consumer.

### What was tricky to build
- Keeping the loop cancellation-safe while the cursor lives in a ref: the effect owns the loop, the ref survives re-renders, and localStorage re-hydrates per agent. Dedupe uses stringified sequences (bigint Set keys would also work; strings match the persisted form).

### What warrants a second pair of eyes
- The 500 ms inter-poll pause: with `wait=25` the loop is effectively continuous; confirm the pause does not cause a missed-event window beyond the acceptable at-least-once semantics (it cannot — the cursor is only advanced after a successful parse).
- `useAckEvents` fires a raw fetch (not RTK) — deliberate (ack is a side effect, not cached state), but confirm no double-fire on re-render in review.

### What should be done in the future
- W6: search + metadata filters (MetadataFilterPanel from AdvancedSearchPanel).
- Inbox: unread marker/badge count in the menubar once events accumulate.
- Consider `visibilitychange` to pause the loop in background tabs.

### Code review instructions
- Start: `web/src/hooks/useEventStream.ts` (the loop + cursor rules), then `InboxScreen.tsx`.
- Validate: `cd web && pnpm check && pnpm test && pnpm build`; live flow per this step (screenshots in `screens/`).

### Technical details
- Commit: 6c2a0c7.
- Live observation: cursor 2 → 5; events: watching×3 (thread + opening post + reply in "Caching investigation"), subforum×2 (thread + opening post in "Stale entries after deploy").
