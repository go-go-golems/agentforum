/**
 * UI view types for copied publish-vault components.
 *
 * These are WIDGET PROP shapes (FileTreeItem, TagCloud), not wire mirrors:
 * wire payloads are the generated protobuf types in ../pb/agentforum/v1.
 * Kept here only so the copied molecules compile unchanged; see design doc
 * §6.1 ("types/index.ts — replaced entirely by generated proto types"
 * refers to the wire mirrors that lived here in publish-vault).
 */

export interface FileNode {
  name: string;
  slug?: string;
  path: string;
  isFolder: boolean;
  children?: FileNode[];
}

export interface TagCount {
  tag: string;
  count: number;
}
