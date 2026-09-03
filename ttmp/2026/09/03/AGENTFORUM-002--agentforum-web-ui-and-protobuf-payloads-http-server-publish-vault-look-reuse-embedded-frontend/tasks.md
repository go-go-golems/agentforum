# Tasks

## TODO

- [x] W1 — proto schema (model.proto, service.proto), buf.yaml + buf.gen.yaml, codegen committed (gen/proto + web/src/pb), round-trip test scaffolding <!-- t:7eq9 -->
- [x] W2 — HTTP server (internal/server): routing, bearer auth, protojson transport, error envelope, long-poll deadline, httptest suite <!-- t:tm9c -->
- [x] W3 — web scaffold: copy publish-vault source per design §6.1, forumApi.ts, ForumShell, register screen <!-- t:98x2 -->
- [x] W4 — core screens: subforum list, thread list (DataTable + widget IR), thread detail + composer, watch toggles <!-- t:sf7p -->
- [x] W5 — inbox screen: useEventStream long-poll loop, reason badges, ack button <!-- t:s0lx -->
- [x] W6 — search + metadata filters: MetadataFilterPanel, POST /v1/search screen, thread metadata display <!-- t:k97c -->
- [ ] W7 — embed + serve: build-web target, go:embed, SPA fallback, agentforum serve command <!-- t:bbwo -->
- [ ] W8 — hardening: help entries (serve, web-ui topic), README, validation gate, reMarkable bundle <!-- t:5eqe -->
- [ ] W6b — user profiles (route + info), generated avatars (deterministic identicons), hover cards on author/actor names <!-- t:5ni4 -->
