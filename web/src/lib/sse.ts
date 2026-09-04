/**
 * SSE frame parsing (AGENTFORUM-004 S3).
 *
 * Server-Sent Events is a line protocol: frames are separated by a blank
 * line; within a frame, "data: " lines carry the payload and lines starting
 * with ":" are comments (server heartbeats). A ReadableStream delivers the
 * body in chunks that do not align with frames — the parser therefore keeps
 * the unparsed remainder and is fed each chunk as it arrives.
 */

/** The result of feeding one chunk to the SSE frame parser. */
export interface SSEParseResult {
  /** Data payloads of every complete frame in order (comment frames and
   *  empty padding contribute nothing). */
  frames: string[];
  /** Unparsed remainder (a partial frame) to prepend to the next chunk. */
  rest: string;
}

/**
 * parseSSEChunk consumes one decoded chunk plus any leftover from the
 * previous call, returning the complete data frames and the new remainder.
 * Chunks that split a frame in half are the normal case, not the exception.
 */
export function parseSSEChunk(chunk: string, previousRest = ""): SSEParseResult {
  let buf = previousRest + chunk;
  const frames: string[] = [];
  for (;;) {
    const idx = buf.indexOf("\n\n");
    if (idx === -1) break;
    const frame = buf.slice(0, idx);
    buf = buf.slice(idx + 2);
    const data = frame
      .split("\n")
      .filter((l) => l.startsWith("data: "))
      .map((l) => l.slice(6))
      .join("");
    if (data !== "") {
      frames.push(data);
    }
  }
  return { frames, rest: buf };
}
