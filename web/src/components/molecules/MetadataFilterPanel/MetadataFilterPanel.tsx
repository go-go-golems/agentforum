/**
 * MOLECULE: MetadataFilterPanel
 * Adapted from publish-vault's AdvancedSearchPanel (design §6.1): same
 * dialog + draft-state + onApply structure, forum filters instead of note
 * filters (metadata terms, subforum, created-after, entity types).
 */
import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../ui/dialog";
import { Button } from "../../atoms/Button/Button";
import { Input } from "../../atoms/Input/Input";
import { Caption } from "../../foundation/Caption/Caption";
import { Icon } from "../../atoms/Icon/Icon";
import type { ForumSearchInput } from "../../../store/forumApi";

export interface MetadataFilterPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  current: ForumSearchInput;
  subforums: { key: string; title: string }[];
  onApply: (req: ForumSearchInput) => void;
}

export const MetadataFilterPanel: React.FC<MetadataFilterPanelProps> = ({
  open,
  onOpenChange,
  current,
  subforums,
  onApply,
}) => {
  const [terms, setTerms] = useState<[string, string][]>(current.terms ?? []);
  const [subforum, setSubforum] = useState(current.subforum ?? "");
  const [createdAfter, setCreatedAfter] = useState(current.createdAfter ?? "");
  const [entityTypes, setEntityTypes] = useState<string[]>(
    current.entityTypes ?? []
  );

  useEffect(() => {
    if (open) {
      setTerms(current.terms ?? []);
      setSubforum(current.subforum ?? "");
      setCreatedAfter(current.createdAfter ?? "");
      setEntityTypes(current.entityTypes ?? []);
    }
  }, [open, current]);

  const toggleEntity = (t: string) =>
    setEntityTypes((prev) =>
      prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]
    );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="retro-window max-w-md">
        <DialogHeader className="retro-window-title">
          <DialogTitle>Filters</DialogTitle>
          <DialogDescription className="sr-only">
            Metadata and scope filters for search
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3 p-3 text-xs">
          <label className="flex flex-col gap-1">
            <Caption>Subforum</Caption>
            <select
              value={subforum}
              onChange={(e) => setSubforum(e.target.value)}
              className="retro-search"
            >
              <option value="">all subforums</option>
              {subforums.map((sf) => (
                <option key={sf.key} value={sf.key}>
                  {sf.title || sf.key}
                </option>
              ))}
            </select>
          </label>

          <div className="flex flex-col gap-1">
            <Caption>Entity types</Caption>
            <div className="flex gap-2">
              {["thread", "post"].map((t) => (
                <button
                  key={t}
                  type="button"
                  className={
                    entityTypes.includes(t)
                      ? "retro-btn retro-btn-primary"
                      : "retro-btn"
                  }
                  onClick={() => toggleEntity(t)}
                >
                  {t}
                </button>
              ))}
              {entityTypes.length === 0 && (
                <span className="text-[10px] text-[var(--color-muted-foreground)] self-center">
                  (both)
                </span>
              )}
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <Caption>Metadata terms (key = value, AND-combined)</Caption>
            {terms.map(([k, v], i) => (
              <div key={i} className="flex gap-1 items-center">
                <Input
                  value={k}
                  placeholder="key"
                  onChange={(e) =>
                    setTerms((prev) =>
                      prev.map((t, j) => (j === i ? [e.target.value, t[1]] : t))
                    )
                  }
                />
                <span className="text-[var(--color-muted-foreground)]">=</span>
                <Input
                  value={v}
                  placeholder="value"
                  onChange={(e) =>
                    setTerms((prev) =>
                      prev.map((t, j) => (j === i ? [t[0], e.target.value] : t))
                    )
                  }
                />
                <Button
                  size="sm"
                  variant="danger"
                  onClick={() =>
                    setTerms((prev) => prev.filter((_, j) => j !== i))
                  }
                >
                  <Icon name="close" size={10} />
                </Button>
              </div>
            ))}
            <Button size="sm" onClick={() => setTerms((prev) => [...prev, ["", ""]])}>
              + term
            </Button>
          </div>

          <label className="flex flex-col gap-1">
            <Caption>Created after (RFC 3339)</Caption>
            <Input
              value={createdAfter}
              placeholder="2026-01-01T00:00:00Z"
              onChange={(e) => setCreatedAfter(e.target.value)}
            />
          </label>
        </div>

        <DialogFooter className="border-t border-[var(--color-ink)] p-2 flex gap-2 justify-end">
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => {
              onApply({
                ...current,
                subforum,
                terms: terms.filter(([k, v]) => k.trim() && v.trim()),
                createdAfter: createdAfter.trim() || undefined,
                entityTypes,
              });
              onOpenChange(false);
            }}
          >
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
