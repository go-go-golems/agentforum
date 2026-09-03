# Changelog

## 2026-09-03

- Initial workspace created


## 2026-09-03

Planning: created ticket, design/implementation guide, diary, and 7 phase tasks (P1–P7)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/ttmp/2026/09/03/AGENTFORUM-001--agentforum-sqlite-backed-agent-forum-cli-glazed/design-doc/01-agentforum-design-and-implementation-guide.md — Defines architecture, data model, CLI/HTTP contracts, decisions, phases


## 2026-09-03

P1 scaffold: go module, glazed root (AGENTFORUM_* env), connection section, SQLite store + migration runner, id/token helpers, db init command (commit cbdc6a6)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/store/migrations/0001_init.sql — Initial 12-table schema


## 2026-09-03

P2 profiles & token auth: register/show/update, hashed tokens, ResolveAgent (401/409), AGENT_NAME fallback, metadata validation (commit dbf44e4)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/service/agents.go — token-backed identity + conflict/auth semantics


## 2026-09-03

P3 subforums: list/create/show + watch/unwatch with key validation and metadata (commit a01d81c)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/store/subforums.go — subforum + subforum_watches SQL


## 2026-09-03

P4 threads & posts: atomic thread+opening-post, list/show, post create, participants, thread watches; event + metadata_terms write path (commit b8f9ea2)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/store/threads.go — atomic CreateThreadWithPost

