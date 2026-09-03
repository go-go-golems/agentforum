/**
 * useEventStream — the inbox's long-poll loop (design §7.4).
 *
 * The cursor is the resume point (persisted in localStorage per agent);
 * every poll carries it forward. Delivery is at-least-once, so the client
 * dedupes by sequence. bigint cursors stay bigint end to end (SQLite
 * autoincrement fits in 2^53, but the comparison is kept bigint-vs-bigint
 * to make that assumption explicit instead of silent).
 */
import { useCallback, useEffect, useRef, useState } from "react";
import type { Event } from "../pb/agentforum/v1/model_pb";
import { PollEventsResponseSchema } from "../pb/agentforum/v1/service_pb";
import { fromJson } from "@bufbuild/protobuf";
import type { JsonObject } from "@bufbuild/protobuf";
import { getToken } from "../store/forumApi";

const WAIT_SECONDS = 25;

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

  useEffect(() => {
    let cancelled = false;

    async function poll() {
      try {
        const res = await fetch(
          `/v1/events?cursor=${cursorRef.current.toString()}&wait=${WAIT_SECONDS}`,
          { headers: { Authorization: `Bearer ${getToken()}` } }
        );
        if (cancelled) return;
        if (res.status === 401) {
          setError("unauthenticated");
          setConnected(false);
          return;
        }
        if (!res.ok) throw new Error(`poll failed: ${res.status}`);

        const body = fromJson(PollEventsResponseSchema, (await res.json()) as JsonObject);
        setConnected(true);
        setError(null);

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
      } catch {
        if (!cancelled) {
          setConnected(false);
          setError("connection lost");
        }
      }
    }

    (async function loop() {
      while (!cancelled) {
        await poll();
        // brief pause between long-polls; the server holds each request for
        // up to WAIT_SECONDS when caught up, so this is not a busy loop
        await new Promise((r) => setTimeout(r, 500));
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
