# Changelog

## 2026-09-04

- Initial workspace created


## 2026-09-04

Step 1: ticket created, R4 pool premise verified in code, intern-facing design guide written (auth analysis, endpoint, client reader, S1-S4)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/ttmp/2026/09/04/AGENTFORUM-004--agentforum-sse-event-stream-connection-pool-hardening-v1-events-stream/design-doc/01-agentforum-sse-event-stream-analysis-design-and-implementation-guide.md — Design contract for the SSE stream


## 2026-09-04

Step 2 (S1): pool raised to 8 open/8 idle; TestPollEventsConcurrentLongPollers (16 pollers + 8 writes, green under -race, spread 2.7ms -> 0.4ms); TestOpenPoolSettings pins the regression; full gate green

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/service/events_test.go — N-poller concurrency test with latency distribution
- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/store/store.go — Pool raised, rationale documented, PRAGMA-per-connection verified


## 2026-09-04

Step 3 (S2): SSE endpoint (one PollEventsResponse per frame, heartbeat goroutine, disconnect-safe join) + streaming test suite green under -race; found and fixed the DSN pragma bug (url.Values.Set replaced — WAL/busy_timeout never active since AGENTFORUM-001, root cause of the AGENTFORUM-005 CI flake)

### Related Files

- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/server/events_stream.go — SSE endpoint (S2)
- /home/manuel/code/wesen/2026-09-03--agent-forum/internal/store/store.go — buildDSN pragma fix — q.Add, function-call syntax

