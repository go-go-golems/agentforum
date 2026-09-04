/**
 * Protojson round-trip suite (TS side). Reads the SAME golden fixtures as
 * the Go suite (internal/server/protojson_test.go reads
 * testdata/protojson/*.json) so both languages assert identical wire bytes.
 *
 * Pins the two type traps from the design doc (§4.6):
 *  - int64 fields arrive as JSON strings and decode to bigint;
 *  - google.protobuf.Struct decodes to a plain JsonObject.
 */
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import { fromJson, toJson } from "@bufbuild/protobuf";
import type { JsonObject } from "@bufbuild/protobuf";
import {
  GetThreadResponseSchema,
  PollEventsResponseSchema,
  CreateThreadRequestSchema,
  ListPostsRequestSchema,
} from "../agentforum/v1/service_pb";

const fixturesDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../../../testdata/protojson"
);

function fixture(name: string): Record<string, unknown> {
  return JSON.parse(readFileSync(path.join(fixturesDir, name), "utf-8"));
}

describe("protojson wire shape (shared fixtures)", () => {
  it("decodes events with bigint sequences and string cursors", () => {
    const resp = fromJson(PollEventsResponseSchema, fixture("event.json") as JsonObject);
    expect(resp.events).toHaveLength(1);
    const ev = resp.events[0]!;
    expect(ev.sequence).toBe(42n);
    expect(resp.nextCursor).toBe(43n);
    // bigint survives the cache; stringification happens at render time
    expect(String(ev.sequence)).toBe("42");
  });

  it("decodes metadata as a plain JSON object", () => {
    const resp = fromJson(GetThreadResponseSchema, fixture("thread.json") as JsonObject);
    const t = resp.thread!;
    expect(t.subforumKey).toBe("engineering");
    expect(t.postCount).toBe(3n);
    // Struct -> JsonObject: plain object access, no accessor methods
    const meta = t.metadata as Record<string, unknown>;
    expect(meta["ticket"]).toBe("PLAT-431");
    expect(meta["keywords"]).toEqual(["caching", "invalidation"]);
    expect(t.watching).toBe(true);
  });

  it("decodes nested request bodies (initialPost)", () => {
    const req = fromJson(CreateThreadRequestSchema, fixture("create_thread_request.json") as JsonObject);
    expect(req.subforumKey).toBe("engineering");
    expect(req.initialPost?.body).toBe("The cache key is missing the locale.");
    expect(req.idempotencyKey).toBe("run-7");
    expect(req.watch).toBe(true);
    const threadMeta = req.metadata as Record<string, unknown>;
    expect(threadMeta["ticket"]).toBe("PLAT-432");
  });

  it("re-serializes to the same wire shape (camelCase, string int64)", () => {
    const resp = fromJson(PollEventsResponseSchema, fixture("event.json") as JsonObject);
    const json = toJson(PollEventsResponseSchema, resp) as Record<string, unknown>;
    expect(json["nextCursor"]).toBe("43");
    expect((json["events"] as Record<string, unknown>[])[0]!["sequence"]).toBe("42");
  });
});

describe("A5: ListPosts pagination cursor (shared fixture)", () => {
  it("decodes the afterPostId cursor", () => {
    const req = fromJson(ListPostsRequestSchema, fixture("list_posts_request.json") as JsonObject);
    expect(req.threadId).toBe("th_01M1P8Z2X3");
    expect(req.limit).toBe(50);
    expect(req.afterPostId).toBe("po_01M1P9A4B5");
  });
});
