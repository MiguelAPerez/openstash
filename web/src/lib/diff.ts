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
    // No prototype, so a spec key like "__proto__" stays an ordinary key.
    const out: { [k: string]: Json } = Object.create(null);
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

export interface LineDiffResult {
  lines: DiffLine[];
  /** The diff hit its time budget, so every old line is shown removed and every new line added. */
  approximate: boolean;
}

/** Diff budget: jsdiff is quadratic in the worst case, and it runs on the main thread. */
const DIFF_TIMEOUT_MS = 500;

export function lineDiff(oldText: string, newText: string): DiffLine[] {
  return computeDiff(oldText, newText).lines;
}

export function computeDiff(oldText: string, newText: string, timeout = DIFF_TIMEOUT_MS): LineDiffResult {
  const out: DiffLine[] = [];
  let oldNo = 1;
  let newNo = 1;
  // Terminate both sides so the last line compares equal regardless of a trailing newline.
  const term = (t: string) => (t === "" || t.endsWith("\n") ? t : t + "\n");
  const a = term(oldText);
  const b = term(newText);
  const result = diffLines(a, b, { timeout });
  const approximate = result === undefined;
  const parts = result ?? [
    { value: a, removed: true, added: false },
    { value: b, removed: false, added: true },
  ];
  for (const part of parts) {
    const lines = part.value.replace(/\n$/, "").split("\n");
    if (part.value === "") continue;
    for (const text of lines) {
      if (part.added) out.push({ kind: "add", text, newNo: newNo++ });
      else if (part.removed) out.push({ kind: "del", text, oldNo: oldNo++ });
      else out.push({ kind: "same", text, oldNo: oldNo++, newNo: newNo++ });
    }
  }
  return { lines: out, approximate };
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
