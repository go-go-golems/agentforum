/**
 * Code-block enhancement for markdown bodies.
 *
 * Extracted from publish-vault's
 * components/organisms/NoteView/noteEnhancements.ts (v of this copy):
 * only enhanceCodeBlocks + addCopyButtons are relevant to forum posts —
 * mermaid, math, embeds, and anchors stay vault-side (math is handled by
 * MarkdownBody's own placeholder pass). Styles (.copy-code-btn, .hljs) live
 * in the copied prose.css.
 */
import { highlightCodeBlocks } from "../../../lib/highlightLanguages";

export function enhanceCodeBlocks(root: HTMLElement): () => void {
  let cancelled = false;

  const run = async () => {
    await highlightCodeBlocks(root);
    if (cancelled) return;
    addCopyButtons(root);
  };

  void run();
  return () => {
    cancelled = true;
  };
}

/** Attach a copy button to every <pre> that does not already have one. */
export function addCopyButtons(root: HTMLElement): void {
  const pres = root.querySelectorAll<HTMLElement>("pre");
  pres.forEach(pre => {
    if (pre.querySelector(".copy-code-btn")) return;
    const btn = document.createElement("button");
    btn.className = "copy-code-btn";
    btn.title = "Copy code";
    btn.textContent = "⎘";
    btn.addEventListener("click", () => {
      const code = pre.querySelector("code");
      if (!code) return;
      navigator.clipboard.writeText(code.textContent ?? "").then(() => {
        btn.textContent = "✓";
        setTimeout(() => {
          btn.textContent = "⎘";
        }, 1500);
      });
    });
    pre.style.position = "relative";
    pre.appendChild(btn);
  });
}
