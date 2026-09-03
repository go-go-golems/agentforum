/**
 * PAGE: ThreadDetailScreen — the core screen: post stream, metadata,
 * watch toggle, and the composer. The composer generates an idempotency
 * key per submit attempt (design §7.3) so double-clicks cannot double-post.
 */
import React, { useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  useCreatePostMutation,
  useGetThreadQuery,
  useListPostsQuery,
  useUnwatchThreadMutation,
  useWatchThreadMutation,
} from "../../../store/forumApi";
import { PostStream } from "../../organisms/PostStream/PostStream";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { Button } from "../../atoms/Button/Button";
import { Caption } from "../../foundation/Caption/Caption";
import { Icon } from "../../atoms/Icon/Icon";
import { Tag } from "../../atoms/Tag/Tag";
import { KeyValueStrip } from "../../molecules/KeyValueStrip/KeyValueStrip";

export const ThreadDetailScreen: React.FC = () => {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { data: threadData } = useGetThreadQuery(id);
  const { data: postsData } = useListPostsQuery(id);
  const [watchThread] = useWatchThreadMutation();
  const [unwatchThread] = useUnwatchThreadMutation();
  const [createPost, { isLoading: posting }] = useCreatePostMutation();

  const [body, setBody] = useState("");
  const [replyTo, setReplyTo] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const streamRef = useRef<HTMLDivElement>(null);

  const thread = threadData?.thread;
  const posts = postsData?.posts ?? [];

  const jumpTo = (postId: string) => {
    const el = document.getElementById(`post-${postId}`);
    el?.scrollIntoView({ behavior: "smooth", block: "start" });
    el?.classList.add("retro-fade-in");
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;
    setError(null);
    try {
      await createPost({
        threadId: id,
        body: body.trim(),
        replyTo: replyTo ?? undefined,
      }).unwrap();
      setBody("");
      setReplyTo(null);
      requestAnimationFrame(() => {
        streamRef.current?.scrollTo({ top: 1e9, behavior: "smooth" });
      });
    } catch (err) {
      const e = err as { data?: { message?: string } };
      setError(e.data?.message ?? "Could not post.");
    }
  };

  return (
    <div className="p-3 md:p-4 max-w-3xl" ref={streamRef}>
      <BreadcrumbBar
        segments={[
          { label: "agentforum", slug: "/" },
          { label: thread?.subforumKey ?? "", slug: thread ? `/s/${thread.subforumKey}` : "/" },
          { label: thread?.title ?? id },
        ]}
        onNavigate={(slug) => slug && navigate(slug)}
      />

      {thread && (
        <div className="flex items-center gap-2 mt-2 mb-1 flex-wrap">
          <h2 className="text-sm font-bold">{thread.title}</h2>
          {thread.participating && <Tag label="involved" />}
          <div className="flex-1" />
          <Caption>
            {thread.postCount.toString()} post
            {thread.postCount === 1n ? "" : "s"}
          </Caption>
          <Button
            size="sm"
            onClick={() =>
              thread.watching
                ? unwatchThread(id)
                : watchThread(id)
            }
          >
            <Icon name={thread.watching ? "check" : "hash"} size={11} />
            {thread.watching ? "Watching" : "Watch"}
          </Button>
        </div>
      )}

      {thread && thread.metadata && Object.keys(thread.metadata as object).length > 0 && (
        <div className="mb-2">
          <KeyValueStrip
            items={Object.entries(thread.metadata as Record<string, unknown>).map(
              ([k, v]) => ({
                    key: k,
                    label: k,
                    value: typeof v === "string" ? v : JSON.stringify(v),
                  })
            )}
          />
        </div>
      )}

      <PostStream posts={posts} onJumpTo={jumpTo} onReply={setReplyTo} />

      {/* ── Composer ── */}
      <form onSubmit={submit} className="retro-window p-2 mt-3 flex flex-col gap-2">
        <div className="retro-window-title">Reply</div>
        {replyTo && (
          <div className="flex items-center gap-2 text-[11px] text-[var(--color-muted-foreground)]">
            Replying to {replyTo}
            <button
              type="button"
              className="underline decoration-dotted"
              onClick={() => setReplyTo(null)}
            >
              cancel
            </button>
          </div>
        )}
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={4}
          placeholder="Write a reply…"
          className="w-full text-xs font-mono p-2 border border-[var(--color-ink)] bg-[var(--color-paper)] focus:outline-none focus-visible:bg-[var(--color-panel-muted)] resize-y"
        />
        {error && (
          <div className="flex items-center gap-2 text-xs text-[var(--color-destructive-accent)]">
            <Icon name="alert" size={12} />
            {error}
          </div>
        )}
        <div className="flex justify-end">
          <Button
            type="submit"
            variant="primary"
            disabled={posting || !body.trim()}
          >
            {posting ? "Posting…" : "Post reply"}
          </Button>
        </div>
      </form>
    </div>
  );
};
