/**
 * Deterministic generated avatars (identicons).
 *
 * An agent's avatar is a pure function of its id: a 5-column symmetric
 * pixel grid (mirrored left/right, like classic identicons) plus a
 * foreground colour picked from the retro palette by hash. Same id ->
 * same avatar, forever, with no storage or upload path. Fits Retro
 * System 1: hard pixels, no gradients, one accent colour.
 */

/** Retro-compatible accent palette (the functional colours plus ink). */
const PALETTE = [
  "#0000cc", // link blue
  "#005500", // tag green
  "#cc7700", // warning amber
  "#cc0000", // destructive red
  "#551a8b", // visited purple
  "#1a1a1a", // ink
];

/** FNV-1a — small, fast, stable across runs and browsers. */
function hash(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

export interface AvatarSpec {
  /** 5x5 grid, true = foreground pixel. Mirrored on the vertical axis. */
  cells: boolean[][];
  fg: string;
  bg: string;
}

export function avatarFor(id: string): AvatarSpec {
  const fg = PALETTE[hash(id) % PALETTE.length];
  const cells: boolean[][] = [];
  for (let y = 0; y < 5; y++) {
    const row = [false, false, false, false, false];
    for (let x = 0; x < 3; x++) {
      // each cell's bit comes from its own hash — no state, no ordering
      // sensitivity, trivially reproducible anywhere (Go, tests, docs).
      row[x] = hash(`${id}:${y}:${x}`) % 2 === 1;
    }
    row[3] = row[1];
    row[4] = row[0];
    cells.push(row);
  }
  return { cells, fg, bg: "#ffffff" };
}
