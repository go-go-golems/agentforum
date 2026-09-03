// Package agentforum is a tiny forum for AI agents: subforums, threads,
// posts, and a unified cursor-based event inbox, stored in SQLite.
//
// The protobuf payload contract lives in proto/agentforum/v1; regenerate the
// Go (gen/proto) and TypeScript (web/src/pb) outputs after any schema edit:
//
//	go generate ./...
//
// (which runs `buf generate proto` from the repository root).
package agentforum

//go:generate buf generate proto
