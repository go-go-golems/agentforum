/**
 * Markdown + TeX rendering for post bodies.
 *
 * Pipeline: extract math spans ($$…$$ display, $…$ inline, \(…\) and \[…\])
 * from the raw markdown, replace each with an inert placeholder element that
 * marked passes through verbatim, render the remaining markdown with marked,
 * and sanitize the result with DOMPurify. The MarkdownBody component then
 * swaps each placeholder for a MathJax-typeset node.
 *
 * Delimiters require non-space content edges so "$5 and $6" stays prose.
 * Math inside fenced code blocks is not extracted (v1 limitation: the
 * extraction runs before markdown parsing).
 */
import { marked } from "marked";
import DOMPurify from "dompurify";

export interface MathSpan {
  tex: string;
  display: boolean;
}

export interface RenderedMarkdown {
  html: string;
  maths: MathSpan[];
}

const DISPLAY_RE = /\$\$([\s\S]+?)\$\$|\\\[([\s\S]+?)\\\]/g;
const INLINE_RE = /\$([^\s$](?:[^$]*[^\s$])?)\$|\\\(([\s\S]+?)\\\)/g;

marked.setOptions({ gfm: true, breaks: true });

export function renderMarkdown(raw: string): RenderedMarkdown {
  const maths: MathSpan[] = [];

  const push = (tex: string, display: boolean) => {
    maths.push({ tex: tex.trim(), display });
    return `<span data-af-math="${maths.length - 1}"></span>`;
  };

  let text = raw.replace(DISPLAY_RE, (_m, d1, d2) => push(d1 ?? d2, true));
  text = text.replace(INLINE_RE, (_m, i1, i2) => push(i1 ?? i2, false));

  const html = DOMPurify.sanitize(marked.parse(text, { async: false }) as string, {
    ADD_ATTR: ["data-af-math"],
  });
  return { html, maths };
}
