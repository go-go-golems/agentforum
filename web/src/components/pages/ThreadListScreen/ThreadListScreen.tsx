/**
 * PAGE: ThreadListScreen — threads of one subforum, rendered through the
 * copied DataTable molecule (the widget-IR construction arrives in W4).
 * bigint fields (postCount) are stringified here at the widget boundary,
 * per the schema-exchange rule.
 */
import React, { useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useListThreadsQuery } from "../../../store/forumApi";
import { DataTable } from "../../molecules/DataTable/DataTable";
import { BreadcrumbBar } from "../../molecules/BreadcrumbBar/BreadcrumbBar";
import type { DataTableColumn } from "../../molecules/DataTable/DataTable";
import type { Thread } from "../../../pb/agentforum/v1/model_pb";
import { Tag } from "../../atoms/Tag/Tag";

export const ThreadListScreen: React.FC = () => {
  const { key = "" } = useParams();
  const { data, isLoading } = useListThreadsQuery({ subforum: key });
  const navigate = useNavigate();

  const threads = useMemo(() => data?.threads ?? [], [data]);

  const columns: DataTableColumn<Thread>[] = useMemo(
    () => [
      {
        id: "title",
        header: "Thread",
        cell: (t) => (
          <span className="font-bold text-xs">{t.title}</span>
        ),
      },
      {
        id: "posts",
        header: "Posts",
        align: "end",
        cell: (t) => (
          <span className="tabular-nums text-[11px]">
            {t.postCount.toString()}
          </span>
        ),
      },
      {
        id: "flags",
        header: "",
        align: "end",
        cell: (t) => (
          <span className="flex gap-1 justify-end">
            {t.participating && <Tag label="involved" />}
            {t.watching && <Tag label="watching" />}
          </span>
        ),
      },
      {
        id: "updated",
        header: "Updated",
        align: "end",
        cell: (t) => (
          <span className="text-[11px] text-[var(--color-muted-foreground)] tabular-nums">
            {(t.updatedAt || "").slice(0, 16).replace("T", " ")}
          </span>
        ),
      },
    ],
    []
  );

  return (
    <div className="p-4 md:p-6 max-w-4xl">
      <BreadcrumbBar
        segments={[
          { label: "agentforum", slug: "/" },
          { label: key },
        ]}
        onNavigate={(slug) => slug && navigate(slug)}
      />
      <div className="mt-3">
        {isLoading ? (
          <div className="text-xs text-[var(--color-muted-foreground)] p-2">
            Loading threads…
          </div>
        ) : (
          <DataTable
            columns={columns}
            rows={threads}
            getRowKey={(t) => t.id}
            onRowSelect={(t) => navigate(`/t/${t.id}`)}
            emptyMessage={
              <span className="text-xs text-[var(--color-muted-foreground)] italic">
                No threads in this subforum yet.
              </span>
            }
          />
        )}
      </div>
    </div>
  );
};
