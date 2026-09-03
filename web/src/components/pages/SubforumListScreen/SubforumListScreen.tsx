/**
 * PAGE: SubforumListScreen — the forum home: subforums with thread counts
 * and watching state.
 */
import React from "react";
import { useNavigate } from "react-router-dom";
import { useListSubforumsQuery } from "../../../store/forumApi";
import { Icon } from "../../atoms/Icon/Icon";
import { Caption } from "../../foundation/Caption/Caption";
import { Tag } from "../../atoms/Tag/Tag";
import { Button } from "../../atoms/Button/Button";

export const SubforumListScreen: React.FC = () => {
  const { data, isLoading, isError } = useListSubforumsQuery();
  const navigate = useNavigate();

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
    <div className="p-4 md:p-6 max-w-3xl">
      <Caption as="h3" className="block mb-3">
        Subforums
      </Caption>
      <div className="flex flex-col gap-2">
        {(data?.subforums ?? []).map((sf) => (
          <button
            key={sf.key}
            type="button"
            className="retro-window flex items-center gap-2 px-3 py-2 text-left hover:bg-[var(--color-panel-muted)]"
            onClick={() => navigate(`/s/${sf.key}`)}
          >
            <Icon name="folder" size={14} />
            <span className="font-bold text-xs">{sf.title || sf.key}</span>
            <span className="text-[11px] text-[var(--color-muted-foreground)] truncate flex-1">
              {sf.description || sf.key}
            </span>
            {sf.watching && <Tag label="watching" />}
            <span className="text-[11px] tabular-nums text-[var(--color-muted-foreground)]">
              {sf.threadCount.toString()}
            </span>
          </button>
        ))}
        {(data?.subforums ?? []).length === 0 && (
          <div className="retro-window p-4 text-xs text-[var(--color-muted-foreground)] italic">
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
