/**
 * useEventStream — the inbox's SSE stream consumer (AGENTFORUM-004 S3).
 *
 * One persistent connection to GET /v1/events/stream (fetch + ReadableStream,
 * not EventSource: EventSource cannot send Authorization headers, and tokens
 * never travel in query strings — design D1). Each SSE data frame is one
 * protojson PollEventsResponse, byte-identical to the long-poll body, so
 * the decoder is shared (design D2).
 *
 * The cursor is the resume point (persisted in localStorage per agent) and
 * travels as the stream's ?cursor= on (re)connect. Delivery is
 * at-least-once, so the client dedupes by sequence. bigint cursors stay
 * bigint end to end (SQLite autoincrement fits in 2^53, but the comparison
 * is kept bigint-vs-bigint to make that assumption explicit instead of
 * silent).
 *
 * Reconnection is ours (EventSource would have owned it; fetch does not):
 * the outer loop reopens the stream with backoff when it ends or errors.
 */
import { useCallback, useEffect, useRef, useState } from "react";
import type { Event } from "../pb/agentforum/v1/model_pb";
import { PollEventsResponseSchema } from "../pb/agentforum/v1/service_pb";
import type { PollEventsResponse } from "../pb/agentforum/v1/service_pb";
import { fromJson } from "@bufbuild/protobuf";
import type { JsonObject } from "@bufbuild/protobuf";
import { getToken } from "../store/forumApi";
import { parseSSEChunk } from "../lib/sse";

// Reconnect backoff bounds (the stream itself is held open indefinitely;
// these apply when it ends or errors).
const BACKOFF_START_MS = 500;
const BACKOFF_MAX_MS = 10_000;

function cursorKey(agentId: string) {
  return `agentforum.cursor.${agentId}`;
}

export interface EventStreamState {
  events: Event[];
  cursor: bigint;
  connected: boolean;
  error: string | null;
}

export function useEventStream(agentId: string): EventStreamState {
  const [events, setEvents] = useState<Event[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const cursorRef = useRef(0n);
  const eventsRef = useRef<Event[]>([]);
  eventsRef.current = events;

  // hydrate the cursor once per agent
  useEffect(() => {
    const stored = localStorage.getItem(cursorKey(agentId));
    cursorRef.current = stored !== null ? BigInt(stored) : 0n;
    setEvents([]);
  }, [agentId]);

  // Ingest one decoded frame: dedupe by sequence, append, advance the
  // persisted cursor (same semantics as the long-poll loop it replaced).
  const ingest = (body: PollEventsResponse) => {
    if (body.events.length > 0) {
      // at-least-once delivery: dedupe by sequence before appending
      const seen = new Set(eventsRef.current.map((e) => e.sequence.toString()));
      const fresh = body.events.filter((e) => !seen.has(e.sequence.toString()));
      if (fresh.length > 0) {
        setEvents((prev) => [...prev, ...fresh]);
      }
    }
    if (body.nextCursor > cursorRef.current) {
      cursorRef.current = body.nextCursor;
      localStorage.setItem(cursorKey(agentId), body.nextCursor.toString());
    }
  };

  useEffect(() => {
    let cancelled = false;
    let backoff = BACKOFF_START_MS;

    // One stream lifetime: open the connection, consume frames until the
    // stream ends or errors. Chunks do not align with frames — the buffer
    // accumulates until a blank line completes a frame. Comment frames
    // (server heartbeats) carry no data prefix and are skipped.
    async function stream(): Promise<void> {
      const res = await fetch(
        `/v1/events/stream?cursor=${cursorRef.current.toString()}`,
        { headers: { Authorization: `Bearer ${getToken()}` } }
      );
      if (cancelled) return;
      if (res.status === 401) {
        // The token is bad: do not retry (App clears it and unmounts us).
        setError("unauthenticated");
        setConnected(false);
        return;
      }
      if (!res.ok || !res.body) throw new Error(`stream failed: ${res.status}`);

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let rest = "";
      setConnected(true);
      setError(null);
      backoff = BACKOFF_START_MS; // a connection succeeded; reset backoff

      for (;;) {
        const { done, value } = await reader.read();
        if (cancelled) return;
        if (done) throw new Error("stream ended");
        // Chunks do not align with frames; parseSSEChunk keeps the partial
        // frame as the remainder (unit-tested in lib/__tests__/sse.test.ts).
        const { frames, rest: next } = parseSSEChunk(decoder.decode(value, { stream: true }), rest);
        rest = next;
        for (const data of frames) {
          ingest(fromJson(PollEventsResponseSchema, JSON.parse(data) as JsonObject));
        }
      }
    }

    (async function loop() {
      while (!cancelled) {
        try {
          await stream();
          if (cancelled) return;
        } catch {
          // network error or stream ended: reconnect below
        }
        setConnected(false);
        setError("connection lost");
        await new Promise((r) => setTimeout(r, backoff));
        backoff = Math.min(backoff * 2, BACKOFF_MAX_MS);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [agentId]);

  return { events, cursor: cursorRef.current, connected, error };
}

/** Acknowledge everything through a sequence (durable, server-side cursor
 * for shared identities; the UI's own localStorage cursor is separate). */
export function useAckEvents() {
  return useCallback(async (through: bigint) => {
    const res = await fetch("/v1/events/ack", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${getToken()}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ schemaVersion: 1, throughSequence: through.toString() }),
    });
    if (!res.ok) throw new Error(`ack failed: ${res.status}`);
  }, []);
}
