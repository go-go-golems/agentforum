/**
 * ORGANISM: ForumSidebar — subforum navigation.
 * New for agentforum (the vault Sidebar is note-specific), composed from
 * copied atoms/molecules and the same retro-window chrome classes.
 */
import React from "react";
import { clsx } from "clsx";
import { useNavigate, useParams } from "react-router-dom";
import { SearchBar } from "../../molecules/SearchBar/SearchBar";
import { ScrollArea } from "../../atoms/ScrollArea/ScrollArea";
import { Icon } from "../../atoms/Icon/Icon";
import { Caption } from "../../foundation/Caption/Caption";
import { Tag } from "../../atoms/Tag/Tag";
import { useListSubforumsQuery } from "../../../store/forumApi";

export interface ForumSidebarProps {
  onSearch: (query: string) => void;
  className?: string;
}

export const ForumSidebar: React.FC<ForumSidebarProps> = ({
  onSearch,
  className,
}) => {
  const { data, isLoading } = useListSubforumsQuery();
  const navigate = useNavigate();
  const params = useParams();

  return (
    <aside
      className={clsx("retro-window flex flex-col h-full", "shrink-0", className)}
    >
      <div className="p-2 border-b border-[var(--color-ink)]">
        <SearchBar onSearch={onSearch} placeholder="Search…" />
      </div>

      <ScrollArea className="flex-1 py-1">
        <div className="px-3 pt-2 pb-1">
          <Caption as="h3">Subforums</Caption>
        </div>
        {isLoading ? (
          <div className="flex items-center gap-2 px-3 py-2 text-[11px] text-[var(--color-muted-foreground)]">
            <Icon name="search" size={11} className="animate-pulse" />
            Loading…
          </div>
        ) : data && data.subforums.length > 0 ? (
          data.subforums.map((sf) => (
            <button
              key={sf.key}
              type="button"
              className={clsx(
                "retro-tree-item w-full text-left",
                params.key === sf.key && "active"
              )}
              onClick={() => navigate(`/s/${sf.key}`)}
            >
              <Icon name="folder" size={11} />
              <span className="flex-1 truncate">{sf.title || sf.key}</span>
              {sf.watching && <Tag label={sf.threadCount.toString()} />}
              {!sf.watching && (
                <span className="text-[10px] tabular-nums text-[var(--color-muted-foreground)]">
                  {sf.threadCount.toString()}
                </span>
              )}
            </button>
          ))
        ) : (
          <div className="px-3 py-2 text-[11px] text-[var(--color-muted-foreground)] italic">
            No subforums yet
          </div>
        )}
      </ScrollArea>
    </aside>
  );
};
