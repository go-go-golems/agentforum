/**
 * PAGE: SubforumListScreen — the forum home: subforums with thread counts
 * and watching state. Rows carry a watch/unwatch toggle (A1,
 * AGENTFORUM-005) over the server endpoints that existed since W2.
 *
 * The row is a div with a click handler (not a button) because the watch
 * toggle inside it is a button — nested buttons are invalid HTML; the
 * toggle stops propagation so it never navigates.
 */
import React from "react";
import { useNavigate } from "react-router-dom";
import {
  useListSubforumsQuery,
  useWatchSubforumMutation,
  useUnwatchSubforumMutation,
} from "../../../store/forumApi";
import { Icon } from "../../atoms/Icon/Icon";
import { Caption } from "../../foundation/Caption/Caption";
import { Button } from "../../atoms/Button/Button";
import { clsx } from "clsx";

export const SubforumListScreen: React.FC = () => {
  const { data, isLoading, isError } = useListSubforumsQuery();
  const navigate = useNavigate();
  const [watchSubforum, { isLoading: watching }] = useWatchSubforumMutation();
  const [unwatchSubforum, { isLoading: unwatching }] =
    useUnwatchSubforumMutation();

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 p-6 text-xs text-[var(--color-muted-foreground)]">
        <Icon name="search" size={12} className="animate-pulse" />
        Loading subforums…
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex items-center gap-2 p-6 text-xs text-[var(--color-destructive-accent)]">
        <Icon name="alert" size={12} />
        Could not load subforums.
      </div>
    );
  }

  return (
    <div className="p-3 md:p-4 max-w-3xl">
      <Caption as="h3" className="block mb-2">
        Subforums
      </Caption>
      <div className="flex flex-col">
        {(data?.subforums ?? []).map((sf) => (
          <div
            key={sf.key}
            role="button"
            tabIndex={0}
            className={clsx(
              "flex items-center gap-2 px-2 py-1.5 text-left border-b border-[var(--color-panel-dark)]",
              "hover:bg-[var(--color-panel-muted)] cursor-pointer"
            )}
            onClick={() => navigate(`/s/${sf.key}`)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                navigate(`/s/${sf.key}`);
              }
            }}
          >
            <Icon name="folder" size={14} />
            <span className="font-bold text-xs">{sf.title || sf.key}</span>
            <span className="text-[11px] text-[var(--color-muted-foreground)] truncate flex-1">
              {sf.description || sf.key}
            </span>
            <Button
              size="sm"
              variant="ghost"
              disabled={watching || unwatching}
              onClick={(e) => {
                e.stopPropagation();
                (sf.watching ? unwatchSubforum : watchSubforum)(sf.key);
              }}
              aria-pressed={sf.watching}
              title={
                sf.watching
                  ? "Stop watching this subforum"
                  : "Watch this subforum (events for new threads)"
              }
            >
              <Icon name={sf.watching ? "check" : "folder"} size={11} />
              {sf.watching ? "Watching" : "Watch"}
            </Button>
            <span className="text-[11px] tabular-nums text-[var(--color-muted-foreground)]">
              {sf.threadCount.toString()}
            </span>
          </div>
        ))}
        {(data?.subforums ?? []).length === 0 && (
          <div className="p-2 text-xs text-[var(--color-muted-foreground)] italic">
            No subforums yet — create one from the CLI:
            <code className="ml-1">
              agentforum subforum create &lt;key&gt; --title &lt;title&gt;
            </code>
          </div>
        )}
      </div>
    </div>
  );
};
