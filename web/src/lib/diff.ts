import { diffLines } from "diff";
import type { Json } from "./api";

/** JSON.stringify with object keys sorted, so diffs ignore key order. */
export function stableStringify(value: Json | undefined): string {
  if (value === undefined || value === null) return "";
  return JSON.stringify(sortKeys(value), null, 2);
}

function sortKeys(value: Json): Json {
  if (Array.isArray(value)) return value.map(sortKeys);
  if (value && typeof value === "object") {
    const out: { [k: string]: Json } = {};
    for (const k of Object.keys(value).sort()) out[k] = sortKeys(value[k]);
    return out;
  }
  return value;
}

export type LineKind = "add" | "del" | "same";

export interface DiffLine {
  kind: LineKind;
  text: string;
  /** 1-based line numbers in the old / new text (absent for add / del). */
  oldNo?: number;
  newNo?: number;
}

export type DiffRow = { type: "line"; line: DiffLine } | { type: "gap"; hidden: DiffLine[] };

export function lineDiff(oldText: string, newText: string): DiffLine[] {
  const out: DiffLine[] = [];
  let oldNo = 1;
  let newNo = 1;
  // Terminate both sides so the last line compares equal regardless of a trailing newline.
  const term = (t: string) => (t === "" || t.endsWith("\n") ? t : t + "\n");
  for (const part of diffLines(term(oldText), term(newText))) {
    const lines = part.value.replace(/\n$/, "").split("\n");
    if (part.value === "") continue;
    for (const text of lines) {
      if (part.added) out.push({ kind: "add", text, newNo: newNo++ });
      else if (part.removed) out.push({ kind: "del", text, oldNo: oldNo++ });
      else out.push({ kind: "same", text, oldNo: oldNo++, newNo: newNo++ });
    }
  }
  return out;
}

/**
 * Collapses runs of unchanged lines longer than 2*context+1 into a gap row,
 * keeping `context` lines around every change.
 */
export function collapse(lines: DiffLine[], context = 3): DiffRow[] {
  const keep = new Array<boolean>(lines.length).fill(false);
  lines.forEach((l, i) => {
    if (l.kind === "same") return;
    for (let j = Math.max(0, i - context); j <= Math.min(lines.length - 1, i + context); j++) keep[j] = true;
  });
  const rows: DiffRow[] = [];
  let gap: DiffLine[] = [];
  const flush = () => {
    if (gap.length === 0) return;
    if (gap.length <= 2) gap.forEach((line) => rows.push({ type: "line", line }));
    else rows.push({ type: "gap", hidden: gap });
    gap = [];
  };
  lines.forEach((line, i) => {
    if (keep[i]) {
      flush();
      rows.push({ type: "line", line });
    } else gap.push(line);
  });
  flush();
  return rows;
}

export function diffStats(lines: DiffLine[]): { added: number; removed: number } {
  let added = 0;
  let removed = 0;
  for (const l of lines) {
    if (l.kind === "add") added++;
    else if (l.kind === "del") removed++;
  }
  return { added, removed };
}
