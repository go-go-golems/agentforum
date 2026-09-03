package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentforumv1 "github.com/go-go-golems/agentforum/gen/proto/agentforum/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// The protojson wire shape is a contract shared with the TypeScript UI
// (web/src/pb, decoded with fromJson). These tests pin it:
//
//  1. camelCase field names,
//  2. int64 fields (sequence, postCount, nextCursor) as JSON *strings*,
//  3. google.protobuf.Struct as a plain JSON object,
//  4. golden round-trip: fixture JSON -> message -> JSON -> message is
//     stable and lossless (proto.Equal).
//
// The fixtures in testdata/protojson/ are also read by the vitest suite
// (web/src/pb/__tests__/roundtrip.test.ts) so both languages assert the
// exact same wire bytes.

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protojson", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func roundTrip(t *testing.T, msg proto.Message, fixture string) {
	t.Helper()

	// 1. Marshal and check the invariants the UI depends on.
	data, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	json := string(data)
	if strings.Contains(json, `"_`) || strings.Contains(json, `"post_count"`) {
		t.Errorf("expected camelCase JSON, got: %s", json)
	}

	// 2. Decode the golden fixture and compare with the constructed message.
	golden := loadFixture(t, fixture)
	var fromFixture proto.Message = proto.Clone(msg)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(golden, fromFixture); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", fixture, err)
	}
	if !proto.Equal(msg, fromFixture) {
		t.Errorf("fixture round-trip mismatch for %s:\nwant: %v\n got: %v",
			fixture, msg, fromFixture)
	}

	// 3. Re-marshal the decoded message — must be byte-stable against a
	// fresh marshal of the constructed message.
	remarshaled, err := protojson.Marshal(fromFixture)
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	if string(remarshaled) != json {
		t.Errorf("remarshal not stable:\nfirst: %s\nsecond: %s", json, remarshaled)
	}
}

func TestPollEventsResponseJSONShape(t *testing.T) {
	ev := &agentforumv1.Event{
		Sequence:    42,
		Type:        agentforumv1.EventType_EVENT_TYPE_POST_CREATED,
		ActorId:     "ag_01M1MJS1ZV605G7JG1CMED8ZTT",
		ActorName:   "alice",
		ThreadId:    "th_01M1MJS2CH3G0QKXYM3BVHHSTC",
		ThreadTitle: "Caching investigation",
		PostId:      "po_01M1MJS2CH3G0QKXYM3ETPG8ZF",
		SubforumKey: "engineering",
		CreatedAt:   "2026-09-03T21:28:41Z",
		Reason:      agentforumv1.EventReason_EVENT_REASON_WATCHING,
	}
	resp := &agentforumv1.PollEventsResponse{
		SchemaVersion: 1,
		Events:        []*agentforumv1.Event{ev},
		NextCursor:    43,
	}

	data, err := protojson.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	// int64 must serialize as JSON strings (TS: bigint after fromJson).
	if !strings.Contains(s, `"sequence":"42"`) {
		t.Errorf("sequence must be a JSON string, got: %s", s)
	}
	if !strings.Contains(s, `"nextCursor":"43"`) {
		t.Errorf("nextCursor must be a JSON string, got: %s", s)
	}
	// enum names are part of the wire contract.
	if !strings.Contains(s, `"EVENT_TYPE_POST_CREATED"`) {
		t.Errorf("event type must serialize as its name, got: %s", s)
	}

	roundTrip(t, resp, "event.json")
}

func TestGetThreadResponseJSONShape(t *testing.T) {
	meta, err := structpb.NewStruct(map[string]any{
		"ticket":   "PLAT-431",
		"keywords": []any{"caching", "invalidation"},
	})
	if err != nil {
		t.Fatalf("new struct: %v", err)
	}
	resp := &agentforumv1.GetThreadResponse{
		SchemaVersion: 1,
		Thread: &agentforumv1.Thread{
			Id:            "th_01M1MJS2CH3G0QKXYM3BVHHSTC",
			SubforumKey:   "engineering",
			Title:         "Caching investigation",
			Metadata:      meta,
			CreatedAt:     "2026-09-03T21:28:41Z",
			UpdatedAt:     "2026-09-03T21:29:10Z",
			LastPostAt:    "2026-09-03T21:29:10Z",
			PostCount:     3,
			Watching:      true,
			Participating: false,
		},
	}

	data, err := protojson.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	// Struct must be a plain JSON object with camelCase outer fields.
	if !strings.Contains(s, `"subforumKey"`) {
		t.Errorf("expected camelCase subforumKey, got: %s", s)
	}
	if !strings.Contains(s, `"postCount":"3"`) {
		t.Errorf("postCount must be a JSON string, got: %s", s)
	}
	if strings.Contains(s, `"metadata":{"fields"`) || strings.Contains(s, `"listValue"`) {
		t.Errorf("Struct must serialize as a plain JSON object, got: %s", s)
	}

	roundTrip(t, resp, "thread.json")
}

func TestCreateThreadRequestJSONShape(t *testing.T) {
	threadMeta, err := structpb.NewStruct(map[string]any{"ticket": "PLAT-432"})
	if err != nil {
		t.Fatalf("new struct: %v", err)
	}
	postMeta, err := structpb.NewStruct(map[string]any{"turn": "14"})
	if err != nil {
		t.Fatalf("new struct: %v", err)
	}
	req := &agentforumv1.CreateThreadRequest{
		SchemaVersion: 1,
		SubforumKey:   "engineering",
		Title:         "Stale entries",
		Metadata:      threadMeta,
		InitialPost: &agentforumv1.CreateThreadRequest_PostBody{
			Body:     "The cache key is missing the locale.",
			Metadata: postMeta,
		},
		Watch:          true,
		IdempotencyKey: "run-7",
	}
	roundTrip(t, req, "create_thread_request.json")
}

func TestUnmarshalAcceptsInt64AsStringAndNumber(t *testing.T) {
	// TS clients send bigints via JSON.stringify -> strings; some proxies
	// or hand-rolled curl calls send numbers. protojson accepts both.
	var resp agentforumv1.PollEventsResponse
	if err := protojson.Unmarshal([]byte(`{"nextCursor":"44"}`), &resp); err != nil {
		t.Fatalf("string cursor: %v", err)
	}
	if resp.NextCursor != 44 {
		t.Errorf("string cursor decoded to %d, want 44", resp.NextCursor)
	}
	if err := protojson.Unmarshal([]byte(`{"nextCursor":45}`), &resp); err != nil {
		t.Fatalf("number cursor: %v", err)
	}
	if resp.NextCursor != 45 {
		t.Errorf("number cursor decoded to %d, want 45", resp.NextCursor)
	}
}
