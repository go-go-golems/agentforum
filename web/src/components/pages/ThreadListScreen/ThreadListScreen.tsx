/**
 * PAGE: ThreadListScreen — threads of one subforum, built as widget IR in
 * useMemo and rendered through the WidgetRenderer + defaultRegistry
 * (design §6.3: the IR is the presentation contract; client-side IR for
 * now, server-emitted later).
 *
 * bigint fields are stringified at the row-construction boundary (the IR is
 * plain JSON data); the RTK cache keeps proto messages untouched.
 */
import React, { useMemo } from "react";
import { useParams } from "react-router-dom";
import {
  useListThreadsQuery,
  useGetSubforumQuery,
  useWatchSubforumMutation,
  useUnwatchSubforumMutation,
} from "../../../store/forumApi";
import { WidgetRenderer } from "../../../widgets/WidgetRenderer";
import { defaultWidgetRegistry } from "../../../widgets/defaultRegistry";
import type { ComponentNode } from "../../../widgets/ir";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { Button } from "../../atoms/Button/Button";
import { Icon } from "../../atoms/Icon/Icon";
import { useNavigate } from "react-router-dom";

export const ThreadListScreen: React.FC = () => {
  const { key = "" } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useListThreadsQuery({ subforum: key });
  // A1: subforum watch state + toggle on the subforum's own page (the
  // server surface existed since W2; this is its UI).
  const { data: subforumData } = useGetSubforumQuery(key);
  const [watchSubforum, { isLoading: watching }] = useWatchSubforumMutation();
  const [unwatchSubforum, { isLoading: unwatching }] =
    useUnwatchSubforumMutation();
  const subforum = subforumData?.subforum;

  const threads = useMemo(() => data?.threads ?? [], [data]);

  // IR construction: rows are plain JSON (stringified counts); columns are
  // defunctionalized cell specs; row selection is a navigate action with
  // ${row.id} interpolation.
  const tableNode = useMemo<ComponentNode | null>(() => {
    if (isLoading) return null;
    return {
      kind: "component",
      type: "DataTable",
      props: {
        columns: [
          {
            id: "title",
            header: "Thread",
            cell: { kind: "field", field: "title" },
          },
          {
            id: "posts",
            header: "Posts",
            align: "end",
            cell: { kind: "field", field: "postCount" },
          },
          {
            id: "flags",
            header: "",
            align: "end",
            cell: { kind: "status", field: "perspective" },
          },
          {
            id: "updated",
            header: "Updated",
            align: "end",
            cell: { kind: "caption", field: "updatedAt", tone: "muted" },
          },
        ],
        rows: threads.map((t) => ({
          id: t.id,
          title: t.title,
          postCount: t.postCount.toString(),
          perspective:
            t.participating && t.watching
              ? "done"
              : t.participating
                ? "involved"
                : t.watching
                  ? "published"
                  : "",
          updatedAt: (t.updatedAt || "").slice(0, 16).replace("T", " "),
        })),
        getRowKey: "id",
        emptyMessage: "No threads in this subforum yet.",
        onRowSelect: { kind: "navigate", to: "/t/${row.id}" },
      },
    };
  }, [threads, isLoading]);

  return (
    <div className="p-3 md:p-4 max-w-4xl">
      <BreadcrumbBar
        segments={[
          { label: "agentforum", slug: "/" },
          { label: key },
        ]}
        onNavigate={(slug) => slug && navigate(slug)}
      />
      {subforum && (
        <div className="flex items-center gap-2 mt-2 mb-1">
          {subforum.description && (
            <span className="text-[11px] text-[var(--color-muted-foreground)] truncate flex-1">
              {subforum.description}
            </span>
          )}
          <Button
            size="sm"
            variant="ghost"
            disabled={watching || unwatching}
            onClick={() =>
              (subforum.watching ? unwatchSubforum : watchSubforum)(key)
            }
            aria-pressed={subforum.watching}
          >
            <Icon name={subforum.watching ? "check" : "folder"} size={11} />
            {subforum.watching ? "Watching subforum" : "Watch subforum"}
          </Button>
        </div>
      )}
      <div className="mt-2">
        {isLoading ? (
          <div className="text-xs text-[var(--color-muted-foreground)] p-2">
            Loading threads…
          </div>
        ) : tableNode ? (
          <WidgetRenderer node={tableNode} registry={defaultWidgetRegistry} />
        ) : null}
      </div>
    </div>
  );
};
