# Changelog

## 2026-09-04

- Initial workspace created


## 2026-09-04

Step 1: backlog triaged — 5 address (watch UI, profile editing, token anomaly probe, CI, pagination cursor), 4 deferred, 2 accepted

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/ttmp/2026/09/04/AGENTFORUM-005--agentforum-ui-parity-hardening-and-backlog-triaged-polish-from-agentforum-001-002-diaries/design-doc/01-backlog-triage-what-to-address-what-to-defer-what-to-drop.md — Triage record


## 2026-09-04

Step 2 (P1/A3): anomaly probe pinned by 5 tests; guard tightened startsWith(af_) -> ^af_[A-Za-z0-9_-]+$ (bare prefix was accepted — real bug found by the probe); vitest 9/9, tsc clean

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/store/forumApi.ts — Guard tightened; case analysis pinned in forumApi.test.ts

