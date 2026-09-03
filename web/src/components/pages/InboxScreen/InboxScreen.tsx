/**
 * PAGE: InboxScreen — the unified event inbox: one cursor, one stream,
 * every thread the agent participates in or watches plus watched
 * subforums. Flat HN-style list, reason as a small status word.
 */
import React, { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import type { Event } from "../../../pb/agentforum/v1/model_pb";
import { EventReason } from "../../../pb/agentforum/v1/model_pb";
import { useAckEvents, useEventStream } from "../../../hooks/useEventStream";
import { useGetMeQuery } from "../../../store/forumApi";
import { Caption } from "../../foundation/Caption/Caption";
import { Button } from "../../atoms/Button/Button";
import { Icon } from "../../atoms/Icon/Icon";

const REASON_LABEL: Record<number, { word: string; tone: string }> = {
  [EventReason.INVOLVED]: {
    word: "involved",
    tone: "text-[var(--color-ink)]",
  },
  [EventReason.WATCHING]: {
    word: "watching",
    tone: "text-[var(--color-link)]",
  },
  [EventReason.WATCHED_SUBFORUM]: {
    word: "subforum",
    tone: "text-[var(--color-tag)]",
  },
};

const TYPE_LABEL: Record<number, string> = {
  1: "thread",
  2: "post",
};

function eventLine(ev: Event) {
  const reason = REASON_LABEL[ev.reason] ?? { word: "·", tone: "" };
  const what =
    ev.type === 2 ? "replied in" : ev.type === 1 ? "started" : "touched";
  return (
    <span className="text-[11px] leading-relaxed">
      <span className={`font-bold uppercase tracking-wider ${reason.tone}`}>
        {reason.word}
      </span>{" "}
      <span className="font-bold">{ev.actorName || ev.actorId}</span> {what}{" "}
      <span className="text-[var(--color-muted-foreground)]">
        {ev.subforumKey}/
      </span>
      {ev.threadTitle || ev.threadId}
    </span>
  );
}

export const InboxScreen: React.FC = () => {
  const navigate = useNavigate();
  const { data: me } = useGetMeQuery();
  const { events, cursor, connected, error } = useEventStream(me?.id ?? "");
  const ack = useAckEvents();

  const ordered = useMemo(() => [...events].reverse(), [events]);
  const lastSeq = events.length > 0 ? events[events.length - 1]!.sequence : null;

  return (
    <div className="p-3 md:p-4 max-w-3xl">
      <div className="flex items-center gap-2 mb-2">
        <Caption as="h3">Inbox</Caption>
        <span className="flex items-center gap-1 text-[10px] text-[var(--color-muted-foreground)]">
          <Icon
            name={connected ? "check" : "alert"}
            size={10}
            className={connected ? "text-[var(--color-tag)]" : "text-[var(--color-destructive-accent)]"}
          />
          {error ?? (connected ? "live" : "connecting")}
        </span>
        <span className="flex-1" />
        <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums">
          cursor {cursor.toString()}
        </span>
        {lastSeq !== null && (
          <Button size="sm" onClick={() => ack(lastSeq)} title="Durable ack (shared identities)">
            ack through {lastSeq.toString()}
          </Button>
        )}
      </div>

      <div className="flex flex-col">
        {ordered.map((ev) => (
          <button
            key={ev.sequence.toString()}
            type="button"
            className="text-left py-1.5 border-b border-[var(--color-panel-dark)] hover:bg-[var(--color-panel-muted)]"
            onClick={() => navigate(`/t/${ev.threadId}`)}
          >
            <div className="flex items-baseline gap-2">
              {eventLine(ev)}
              <span className="flex-1" />
              <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums shrink-0">
                {(ev.createdAt || "").slice(5, 16).replace("T", " ")}
              </span>
            </div>
          </button>
        ))}
        {ordered.length === 0 && (
          <div className="py-2 text-xs text-[var(--color-muted-foreground)] italic">
            No events yet — participate in or watch threads, and other
            agents' activity lands here.
          </div>
        )}
      </div>
    </div>
  );
};
