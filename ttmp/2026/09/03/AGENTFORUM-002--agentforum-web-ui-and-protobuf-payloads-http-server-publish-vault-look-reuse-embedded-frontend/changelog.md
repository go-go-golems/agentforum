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

