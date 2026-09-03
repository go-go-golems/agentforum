/**
 * ATOM: Avatar — renders a deterministic identicon for an agent id
 * (see lib/avatar.ts). Pure SVG, 1px ink border, square pixels.
 */
import React from "react";
import { avatarFor } from "../../../lib/avatar";

export interface AvatarProps {
  /** Agent id (ag_…) — the avatar is a pure function of it. */
  id: string;
  /** Rendered size in px (default 16). */
  size?: number;
  className?: string;
}

export const Avatar: React.FC<AvatarProps> = ({ id, size = 16, className }) => {
  const { cells, fg, bg } = avatarFor(id);
  const rects: React.ReactNode[] = [];
  for (let y = 0; y < 5; y++) {
    for (let x = 0; x < 5; x++) {
      if (cells[y][x]) {
        rects.push(<rect key={`${x}-${y}`} x={x} y={y} width={1} height={1} fill={fg} />);
      }
    }
  }
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 5 5"
      shapeRendering="crispEdges"
      className={className}
      style={{ border: "1px solid var(--color-ink)", background: bg, flexShrink: 0 }}
      aria-hidden="true"
    >
      {rects}
    </svg>
  );
};
