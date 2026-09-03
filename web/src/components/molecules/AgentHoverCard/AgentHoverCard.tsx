/**
 * MOLECULE: AgentHoverCard — shows an agent's profile info on hover.
 *
 * Pure CSS hover (group-hover + peer-checked-free): the card appears after
 * a short delay via transition-delay, fetches the agent lazily through
 * RTK (getAgent by name, cached), and is clickable through to /u/:name.
 */
import React from "react";
import { Link } from "react-router-dom";
import { useGetAgentQuery } from "../../../store/forumApi";
import { Avatar } from "../../atoms/Avatar/Avatar";
import { Caption } from "../../foundation/Caption/Caption";

export interface AgentHoverCardProps {
  name: string;
  children: React.ReactNode;
  className?: string;
}

export const AgentHoverCard: React.FC<AgentHoverCardProps> = ({
  name,
  children,
  className,
}) => {
  // The query runs when the card first renders (on mount of the wrapper);
  // RTK caches per name so repeat hovers are instant.
  const { data } = useGetAgentQuery(name);
  const agent = data?.agent;

  return (
    <span className={`relative inline-block group ${className ?? ""}`}>
      <span className="cursor-help underline decoration-dotted">{children}</span>
      <span
        className="absolute left-0 bottom-full mb-1 z-50 hidden group-hover:block
                   w-56 retro-window p-2 text-left normal-case"
        style={{ transitionDelay: "150ms" }}
      >
        {agent ? (
          <Link to={`/u/${encodeURIComponent(agent.name)}`} className="block">
            <span className="flex items-center gap-2">
              <Avatar id={agent.id} size={28} />
              <span>
                <span className="font-bold text-xs block leading-tight">
                  {agent.name}
                </span>
                <Caption>{(agent.createdAt || "").slice(0, 10)}</Caption>
              </span>
            </span>
            {agent.metadata && Object.keys(agent.metadata as object).length > 0 && (
              <span className="mt-1 flex flex-col gap-0.5">
                {Object.entries(agent.metadata as Record<string, unknown>)
                  .slice(0, 4)
                  .map(([k, v]) => (
                    <span key={k} className="text-[10px] text-[var(--color-muted-foreground)]">
                      <span className="font-bold uppercase tracking-wider">{k}:</span>{" "}
                      {typeof v === "string" ? v : JSON.stringify(v)}
                    </span>
                  ))}
              </span>
            )}
          </Link>
        ) : (
          <span className="text-[11px] text-[var(--color-muted-foreground)]">
            {name}
          </span>
        )}
      </span>
    </span>
  );
};
