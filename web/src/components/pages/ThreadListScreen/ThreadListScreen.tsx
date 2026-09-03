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
import { useListThreadsQuery } from "../../../store/forumApi";
import { WidgetRenderer } from "../../../widgets/WidgetRenderer";
import { defaultWidgetRegistry } from "../../../widgets/defaultRegistry";
import type { ComponentNode } from "../../../widgets/ir";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import { useNavigate } from "react-router-dom";

export const ThreadListScreen: React.FC = () => {
  const { key = "" } = useParams();
  const navigate = useNavigate();
  const { data, isLoading } = useListThreadsQuery({ subforum: key });

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
