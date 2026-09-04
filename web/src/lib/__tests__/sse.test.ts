/**
 * SSE frame parser tests (AGENTFORUM-004 S3): chunks do not align with
 * frames — a frame may be split across any number of chunks, and comment
 * frames (heartbeats) carry no payload.
 */
import { describe, expect, it } from "vitest";
import { parseSSEChunk } from "../sse";

describe("parseSSEChunk", () => {
  it("parses a complete data frame", () => {
    const r = parseSSEChunk('data: {"a":1}\n\n');
    expect(r.frames).toEqual(['{"a":1}']);
    expect(r.rest).toBe("");
  });

  it("reassembles a frame split across chunks", () => {
    const a = parseSSEChunk('data: {"seq');
    expect(a.frames).toEqual([]);
    const b = parseSSEChunk('uence":42}\n', a.rest);
    expect(b.frames).toEqual([]);
    const c = parseSSEChunk("\n", b.rest);
    expect(c.frames).toEqual(['{"sequence":42}']);
    expect(c.rest).toBe("");
  });

  it("splits at a boundary that lands exactly on the separator", () => {
    const a = parseSSEChunk("data: one\n");
    const b = parseSSEChunk("\ndata: two\n\n", a.rest);
    expect(b.frames).toEqual(["one", "two"]);
  });

  it("skips comment frames (heartbeats) and padding", () => {
    const r = parseSSEChunk(": ping\n\n\ndata: {\"x\":1}\n\n");
    expect(r.frames).toEqual(['{"x":1}']);
    expect(r.rest).toBe("");
  });

  it("joins multi-line data frames per the SSE spec", () => {
    const r = parseSSEChunk("data: {\"a\":\ndata: 1}\n\n");
    expect(r.frames).toEqual(['{"a":1}']);
  });

  it("keeps a partial trailing frame as the remainder", () => {
    const r = parseSSEChunk('data: {"a":1}\n\ndata: {"trunc');
    expect(r.frames).toEqual(['{"a":1}']);
    expect(r.rest).toBe('data: {"trunc');
  });
});
