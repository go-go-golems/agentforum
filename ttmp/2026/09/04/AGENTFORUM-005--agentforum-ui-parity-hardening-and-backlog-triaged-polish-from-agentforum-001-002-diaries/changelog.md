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


## 2026-09-04

Step 3 (P2/A4): ci.yml mirroring the full gate (gofmt/vet/test/builds/pnpm/buf drift); actionlint clean; every step command verified verbatim locally; live GitHub run pending origin-remote decision

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/.github/workflows/ci.yml — CI gate workflow (A4)


## 2026-09-04

Step 4 (P3/A1): watch/unwatch subforum UI (forumApi endpoints + row buttons + subforum-page header toggle), verified live with screenshots; W7 anomaly root-caused as RTK invalidation race and fixed in App.tsx (skip getMe without token), 8/8 register flows green

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/App.tsx — W7 anomaly fix — skip getMe when no token exists
- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/components/pages/SubforumListScreen/SubforumListScreen.tsx — Row restructure + Watch buttons
- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/store/forumApi.ts — getSubforum/watchSubforum/unwatchSubforum


## 2026-09-04

Step 5 (P4/A2): updateMe mutation + own-profile JSON metadata editor on ProfileScreen; verified live (own editable, others read-only), screenshots 03-05

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/components/pages/ProfileScreen/ProfileScreen.tsx — Metadata editor, own profile only


## 2026-09-04

Step 6 (P5/A5): after_post_id cursor on ListPostsRequest (schema-first: buf codegen, shared fixture, Go+TS round-trip tests), handler ?after= wiring, TestListPostsPagination, UI load-more via RTK merge; verified live 50->100->105 on a 105-post thread, screenshots 06/07

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/server/handlers.go — after query param wired
- /home/manuel/code/wesen/2026-09-03--agent-forum/proto/agentforum/v1/service.proto — after_post_id field 4 on ListPostsRequest
- /home/manuel/code/wesen/2026-09-03--agent-forum/web/src/store/forumApi.ts — Paginated listPosts with serializeQueryArgs + merge


## 2026-09-04

Step 7 (P6): fresh full gate green (incl. -race); all four flows re-verified live on the final binary; server cleaned up; report delivered to go-go-parc vault

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/ttmp/2026/09/04/AGENTFORUM-005--agentforum-ui-parity-hardening-and-backlog-triaged-polish-from-agentforum-001-002-diaries/reference/01-investigation-diary.md — Final verification record


## 2026-09-04

Step 8: repo created (go-go-golems/agentforum, private), 46 commits pushed, first CI run green on GitHub runners (run 33904474396, success, 3m13s) — A4 fully closed

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/.github/workflows/ci.yml — First run green: https://github.com/go-go-golems/agentforum/actions/runs/33904474396


## 2026-09-04

Step 9: CI flake triage — full-window long-poll 500 proven (TestLongPollTimeoutReturnsEmpty failed pre-fix) and fixed (pollGrace margin + context-error empty response); body now printed in TestLongPollDelivery failures

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/server/handlers.go — pollGrace margin + context-deadline empty response

