/**
 * PAGE: SearchScreen — text + metadata search over POST /v1/search.
 * The sidebar's SearchBar lands here with ?q=; the Filters dialog adds
 * metadata terms, subforum scope, entity types, and created-after.
 * Results are a flat list (threads and posts) in the HN-austere style,
 * with dates on the right edge.
 */
import React, { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useSearchQuery, useListSubforumsQuery } from "../../../store/forumApi";
import type { ForumSearchInput } from "../../../store/forumApi";
import { MetadataFilterPanel } from "../../molecules/MetadataFilterPanel/MetadataFilterPanel";
import { Button } from "../../atoms/Button/Button";
import { Caption } from "../../foundation/Caption/Caption";
import { Icon } from "../../atoms/Icon/Icon";

/** 2026-09-03T22:57:14Z -> "2026-09-03 22:57" */
function fmt(s: string): string {
  return s ? s.slice(0, 16).replace("T", " ") : "";
}

export const SearchScreen: React.FC = () => {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [text, setText] = useState(params.get("q") ?? "");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filters, setFilters] = useState<ForumSearchInput>({ text: "" });

  const { data: subforums } = useListSubforumsQuery();
  const { data, isLoading, isFetching } = useSearchQuery({
    ...filters,
    text,
  });

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setParams(text ? { q: text } : {});
  };

  const hits = data?.hits ?? [];

  return (
    <div className="p-3 md:p-4 max-w-3xl">
      <form onSubmit={submit} className="flex gap-2 mb-2">
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Search threads and posts…"
          className="retro-search flex-1"
          autoFocus
        />
        <Button type="submit" variant="primary">
          <Icon name="search" size={11} />
          Search
        </Button>
        <Button type="button" onClick={() => setFiltersOpen(true)}>
          Filters
          {filters.terms && filters.terms.length > 0
            ? ` (${filters.terms.length})`
            : ""}
        </Button>
      </form>

      <div className="flex items-center gap-2 mb-1">
        <Caption>Results</Caption>
        {isFetching && (
          <span className="text-[10px] text-[var(--color-muted-foreground)]">
            searching…
          </span>
        )}
        {!isFetching && data && (
          <span className="text-[10px] text-[var(--color-muted-foreground)] tabular-nums">
            {hits.length} hit{hits.length === 1 ? "" : "s"}
          </span>
        )}
      </div>

      <div className="flex flex-col">
        {hits.map((hit, i) => (
          <button
            key={`h${i}`}
            type="button"
            className="text-left py-1.5 border-b border-[var(--color-panel-dark)] hover:bg-[var(--color-panel-muted)]"
            onClick={() =>
              navigate(hit.thread ? `/t/${hit.thread.id}` : `/t/${hit.post!.threadId}`)
            }
          >
            {hit.thread ? (
              <span className="text-[11px]">
                <span className="text-[var(--color-muted-foreground)]">
                  {hit.thread.subforumKey}/
                </span>
                <span className="font-bold">{hit.thread.title}</span>{" "}
                <span className="text-[var(--color-muted-foreground)]">
                  thread · {hit.thread.postCount.toString()} posts
                </span>
              </span>
            ) : (
              <span className="text-[11px]">
                <span className="text-[var(--color-muted-foreground)]">post by </span>
                <span className="font-bold">
                  {hit.post!.authorName || hit.post!.authorId}
                </span>
                <span className="text-[var(--color-muted-foreground)]">
                  {" "}
                  · {(hit.post!.body ?? "").slice(0, 90)}
                  {(hit.post!.body ?? "").length > 90 ? "…" : ""}
                </span>
              </span>
            )}
            <span className="float-right text-[10px] text-[var(--color-muted-foreground)] tabular-nums">
              {fmt(hit.thread ? hit.thread.createdAt : hit.post!.createdAt)}
            </span>
          </button>
        ))}
        {!isLoading && hits.length === 0 && (
          <div className="py-2 text-xs text-[var(--color-muted-foreground)] italic">
            No hits — try text, or add metadata terms in Filters.
          </div>
        )}
      </div>

      <MetadataFilterPanel
        open={filtersOpen}
        onOpenChange={setFiltersOpen}
        current={{ ...filters, text }}
        subforums={(subforums?.subforums ?? []).map((sf) => ({
          key: sf.key,
          title: sf.title,
        }))}
        onApply={setFilters}
      />
    </div>
  );
};
