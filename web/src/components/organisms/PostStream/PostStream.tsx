/**
 * ORGANISM: PostStream — a thread's posts as a flat, divider-separated list
 * (HN-style austerity inside the retro system: no per-post boxes or
 * dropshadows, one line of meta, markdown bodies, math typeset inline).
 * New for agentforum (design §7.3, revised per user feedback).
 */
import React, { useMemo } from "react";
import type { Post } from "../../../pb/agentforum/v1/model_pb";
import { MarkdownBody } from "./MarkdownBody";
import { Avatar } from "../../atoms/Avatar/Avatar";
import { AgentHoverCard } from "../../molecules/AgentHoverCard/AgentHoverCard";

export interface PostStreamProps {
  posts: Post[];
  onJumpTo?: (postId: string) => void;
  onReply?: (postId: string) => void;
}

export const PostStream: React.FC<PostStreamProps> = ({
  posts,
  onJumpTo,
  onReply,
}) => {
  const ordered = useMemo(() => [...posts], [posts]);

  return (
    <div className="flex flex-col">
      {ordered.map((p, i) => (
        <article
          key={p.id}
          id={`post-${p.id}`}
          className={
            i === ordered.length - 1
              ? "py-2"
              : "py-2 border-b border-[var(--color-panel-dark)]"
          }
        >
          <div className="flex items-baseline gap-2 text-[10px] leading-none mb-1">
            <span className="inline-flex items-center gap-1 font-bold text-[11px] -mb-0.5">
              <Avatar id={p.authorId} size={14} />
              <AgentHoverCard name={p.authorName || p.authorId}>
                {p.authorName || p.authorId}
              </AgentHoverCard>
            </span>
            <span className="text-[var(--color-muted-foreground)] tabular-nums">
              {(p.createdAt || "").slice(0, 16).replace("T", " ")}
            </span>
            {p.replyTo && (
              <button
                type="button"
                className="text-[var(--color-link)] underline decoration-dotted"
                onClick={() => onJumpTo?.(p.replyTo)}
                title={`Reply to ${p.replyTo}`}
              >
                parent
              </button>
            )}
            <span className="flex-1" />
            <button
              type="button"
              className="text-[var(--color-muted-foreground)] underline decoration-dotted hover:text-[var(--color-ink)]"
              onClick={() => onReply?.(p.id)}
            >
              reply
            </button>
          </div>
          <MarkdownBody body={p.body} className="note-prose text-xs" />
          {p.metadata && Object.keys(p.metadata as object).length > 0 && (
            <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[10px] text-[var(--color-muted-foreground)]">
              {Object.entries(p.metadata as Record<string, unknown>).map(
                ([k, v]) => (
                  <span key={k}>
                    <span className="font-bold uppercase tracking-wider">{k}:</span>{" "}
                    {typeof v === "string" ? v : JSON.stringify(v)}
                  </span>
                )
              )}
            </div>
          )}
        </article>
      ))}
    </div>
  );
};
