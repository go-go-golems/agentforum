/**
 * MarkdownBody — renders a post body as markdown and typesets its TeX spans.
 *
 * Phase 1 sets innerHTML from the sanitized markdown; phase 2 (effect) swaps
 * each data-af-math placeholder for a MathJax SVG node. A TeX error leaves
 * the source visible in the placeholder's title instead of breaking the post.
 */
import React, { memo, useEffect, useMemo, useRef } from "react";
import { renderMarkdown } from "../../../lib/markdown";
import { ensureMathStyles, typesetTeX } from "../../../lib/mathjax";
import { enhanceCodeBlocks } from "./codeEnhancements";

export interface MarkdownBodyProps {
  body: string;
  className?: string;
}

export const MarkdownBody = memo(function MarkdownBody({
  body,
  className,
}: MarkdownBodyProps) {
  const hostRef = useRef<HTMLDivElement>(null);

  // memo() is load-bearing: the {__html} object is new each render, so an
  // unmemoized re-render would rewrite innerHTML and destroy the typeset
  // math nodes (same reasoning as publish-vault's NoteBody).
  const rendered = useMemo(() => renderMarkdown(body), [body]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    let cancelled = false;
    let stopCode: (() => void) | undefined;

    (async () => {
      // code first: hljs + copy buttons run on the raw markdown HTML;
      // math placeholders are inert spans and survive both passes.
      stopCode = enhanceCodeBlocks(host);
      await ensureMathStyles();
      const placeholders = Array.from(
        host.querySelectorAll<HTMLElement>("[data-af-math]")
      );
      for (const el of placeholders) {
        if (cancelled) return;
        const idx = Number(el.dataset.afMath);
        const span = rendered.maths[idx];
        if (!span) continue;
        const { node, error } = await typesetTeX(span.tex, span.display);
        if (cancelled) return;
        if (node) {
          el.replaceWith(node);
        } else {
          el.textContent = span.display ? `$$${span.tex}$$` : `$${span.tex}$`;
          el.title = error ?? "typeset error";
        }
      }
    })();

    return () => {
      cancelled = true;
      stopCode?.();
    };
  }, [rendered]);

  return (
    <div
      ref={hostRef}
      className={className}
      dangerouslySetInnerHTML={{ __html: rendered.html }}
    />
  );
});
