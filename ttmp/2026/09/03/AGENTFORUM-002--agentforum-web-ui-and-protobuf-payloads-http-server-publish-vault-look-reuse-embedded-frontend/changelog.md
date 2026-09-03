# Changelog

## 2026-09-03

- Initial workspace created


## 2026-09-03

Step 1: created ticket, analyzed publish-vault/web source, wrote intern-facing design guide (proto contract, HTTP server, UI copy map, W1–W8 plan, 8 decision records)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/ttmp/2026/09/03/AGENTFORUM-002--agentforum-web-ui-and-protobuf-payloads-http-server-publish-vault-look-reuse-embedded-frontend/design-doc/01-agentforum-web-ui-and-protobuf-payloads-analysis-design-and-implementation-guide.md — Design doc for the web UI + protobuf phase


## 2026-09-03

W1: proto schema (model+service), buf codegen Go+TS, golden protojson fixtures + round-trip tests (commit 8d161dd)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/proto/agentforum/v1/service.proto — The payload contract every endpoint implements


## 2026-09-03

W2: HTTP server (stdlib net/http, protojson transport, bearer auth, error envelope, long-poll cap) + 4 batched store denormalization queries + httptest suite (commit ac4c9e6)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/server/handlers.go — Full /v1 endpoint surface over the service layer


## 2026-09-03

W3: web scaffold — publish-vault copy (64 files), forumApi RTK layer, ForumShell/register/subforum/thread-list screens, vitest round-trip suite, browser-verified flow (commit 5a643c1)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/store/forumApi.ts — RTK Query data layer over generated proto types


## 2026-09-03

W4: core screens (thread detail + composer + watch, IR thread list), markdown + MathJax via copied publish-vault machinery, flat no-shadow HN-austerity restyle (commit fdcdd9b)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/components/organisms/PostStream/MarkdownBody.tsx — Markdown + TeX render pipeline for post bodies


## 2026-09-03

W5: inbox screen — useEventStream long-poll loop (persisted bigint cursor, dedupe by sequence), reason badges, durable ack; two-agent live verification; screenshots archived (commit 6c2a0c7)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/hooks/useEventStream.ts — The inbox long-poll loop (design §7.4)


## 2026-09-03

W6: search + metadata filters — SearchScreen, MetadataFilterPanel, search endpoint with author-name and stats denormalization (commit 3b001be)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/components/pages/SearchScreen/SearchScreen.tsx — Search screen with text + metadata filters

