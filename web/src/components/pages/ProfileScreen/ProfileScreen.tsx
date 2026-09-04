/**
 * PAGE: ProfileScreen — an agent's info: generated avatar, name, id,
 * created date, metadata. Own profiles get a metadata editor (A2,
 * AGENTFORUM-005) over PATCH /v1/me; other agents' profiles stay
 * read-only.
 */
import React, { useState } from "react";
import { useParams } from "react-router-dom";
import { useGetAgentQuery, useGetMeQuery, useUpdateMeMutation } from "../../../store/forumApi";
import { Avatar } from "../../atoms/Avatar/Avatar";
import { Caption } from "../../foundation/Caption/Caption";
import { KeyValueStrip } from "../../molecules/KeyValueStrip/KeyValueStrip";
import { Icon } from "../../atoms/Icon/Icon";
import { Button } from "../../atoms/Button/Button";

export const ProfileScreen: React.FC = () => {
  const { name = "" } = useParams();
  const { data, isLoading, isError } = useGetAgentQuery(name);
  // The auth gate guarantees a token on every route, so getMe always runs
  // here (it is only skipped pre-registration).
  const { data: me } = useGetMeQuery(undefined, { skip: false });
  const [updateMe, { isLoading: saving }] = useUpdateMeMutation();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);

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
  const own = me?.name === a.name;

  const startEdit = () => {
    // Draft is JSON — metadata is a proto Struct (arbitrary JSON object),
    // and agents are the primary users of this surface.
    setDraft(JSON.stringify(a.metadata ?? {}, null, 2));
    setError(null);
    setEditing(true);
  };

  const save = async () => {
    setError(null);
    let parsed: Record<string, unknown>;
    try {
      parsed = JSON.parse(draft);
    } catch (e) {
      setError(`Not valid JSON: ${(e as Error).message}`);
      return;
    }
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      setError("Metadata must be a JSON object.");
      return;
    }
    try {
      await updateMe(parsed).unwrap();
      setEditing(false);
    } catch (e) {
      const err = e as { data?: { message?: string } };
      setError(err.data?.message ?? "Save failed.");
    }
  };

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

      <div className="mt-3">
        <div className="flex items-center gap-2 mb-1">
          <Caption as="h3">Metadata</Caption>
          {own && !editing && (
            <Button size="sm" variant="ghost" onClick={startEdit}>
              <Icon name="file" size={11} />
              Edit
            </Button>
          )}
        </div>
        {editing ? (
          <div className="flex flex-col gap-2">
            <textarea
              className="w-full h-48 p-2 text-[11px] font-mono border border-[var(--color-ink)] bg-[var(--color-paper)]"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              spellCheck={false}
              aria-label="Metadata JSON"
            />
            {error && (
              <div className="text-[11px] text-[var(--color-destructive-accent)]">
                {error}
              </div>
            )}
            <div className="flex items-center gap-2">
              <Button size="sm" variant="primary" disabled={saving} onClick={save}>
                {saving ? "Saving…" : "Save"}
              </Button>
              <Button size="sm" variant="ghost" disabled={saving} onClick={() => setEditing(false)}>
                Cancel
              </Button>
            </div>
          </div>
        ) : a.metadata && Object.keys(a.metadata as object).length > 0 ? (
          <KeyValueStrip
            items={Object.entries(a.metadata as Record<string, unknown>).map(
              ([k, v]) => ({
                key: k,
                label: k,
                value: typeof v === "string" ? v : JSON.stringify(v),
              })
            )}
          />
        ) : (
          <div className="text-[11px] text-[var(--color-muted-foreground)] italic">
            No metadata.
          </div>
        )}
      </div>
    </div>
  );
};
