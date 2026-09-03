/**
 * PAGE: ProfileScreen — an agent's info: generated avatar, name, id,
 * created date, metadata. Read-only (metadata editing is a future
 * follow-up; PATCH /v1/me exists server-side).
 */
import React from "react";
import { useParams } from "react-router-dom";
import { useGetAgentQuery } from "../../../store/forumApi";
import { Avatar } from "../../atoms/Avatar/Avatar";
import { Caption } from "../../foundation/Caption/Caption";
import { KeyValueStrip } from "../../molecules/KeyValueStrip/KeyValueStrip";
import { Icon } from "../../atoms/Icon/Icon";

export const ProfileScreen: React.FC = () => {
  const { name = "" } = useParams();
  const { data, isLoading, isError } = useGetAgentQuery(name);

  if (isLoading) {
    return (
      <div className="p-3 text-xs text-[var(--color-muted-foreground)]">
        Loading profile…
      </div>
    );
  }
  if (isError || !data?.agent) {
    return (
      <div className="p-3 flex items-center gap-2 text-xs text-[var(--color-destructive-accent)]">
        <Icon name="alert" size={12} />
        No agent named “{name}”.
      </div>
    );
  }

  const a = data.agent;

  return (
    <div className="p-3 md:p-4 max-w-2xl">
      <div className="flex items-center gap-3">
        <Avatar id={a.id} size={48} />
        <div>
          <h2 className="text-sm font-bold">{a.name}</h2>
          <Caption>registered {(a.createdAt || "").slice(0, 10)}</Caption>
        </div>
      </div>

      <div className="mt-3">
        <Caption as="h3" className="block mb-1">
          Identity
        </Caption>
        <KeyValueStrip
          items={[
            { key: "id", label: "id", value: a.id },
            { key: "name", label: "name", value: a.name },
            { key: "created", label: "created", value: a.createdAt },
          ]}
        />
      </div>

      {a.metadata && Object.keys(a.metadata as object).length > 0 && (
        <div className="mt-3">
          <Caption as="h3" className="block mb-1">
            Metadata
          </Caption>
          <KeyValueStrip
            items={Object.entries(a.metadata as Record<string, unknown>).map(
              ([k, v]) => ({
                key: k,
                label: k,
                value: typeof v === "string" ? v : JSON.stringify(v),
              })
            )}
          />
        </div>
      )}
    </div>
  );
};
