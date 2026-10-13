import { describe, expect, it } from "vitest";
import { collapse, computeDiff, diffStats, lineDiff, stableStringify } from "./diff";

describe("stableStringify", () => {
  it("sorts keys recursively", () => {
    expect(stableStringify({ b: 1, a: { d: [{ z: 1, y: 2 }], c: true } })).toBe(
      stableStringify({ a: { c: true, d: [{ y: 2, z: 1 }] }, b: 1 }),
    );
  });
  it("renders null and undefined as empty", () => {
    expect(stableStringify(null)).toBe("");
    expect(stableStringify(undefined)).toBe("");
  });
});

describe("lineDiff", () => {
  it("numbers lines on both sides", () => {
    const lines = lineDiff("a\nb\nc", "a\nB\nc\nd");
    expect(lines.map((l) => `${l.kind}:${l.text}`)).toEqual(["same:a", "del:b", "add:B", "same:c", "add:d"]);
    expect(lines[3]).toMatchObject({ oldNo: 3, newNo: 3 });
    expect(diffStats(lines)).toEqual({ added: 2, removed: 1 });
  });
  it("treats an empty side as all added", () => {
    expect(lineDiff("", "x\ny").every((l) => l.kind === "add")).toBe(true);
  });
});

describe("collapse", () => {
  it("hides long unchanged runs but keeps context", () => {
    const old = Array.from({ length: 20 }, (_, i) => `l${i}`).join("\n");
    const next = old.replace("l10", "L10");
    const rows = collapse(lineDiff(old, next), 2);
    const gaps = rows.filter((r) => r.type === "gap");
    expect(gaps).toHaveLength(2);
    const shown = rows.flatMap((r) => (r.type === "line" ? [r.line.text] : []));
    expect(shown).toEqual(["l8", "l9", "l10", "L10", "l11", "l12"]);
  });
  it("does not create tiny gaps", () => {
    const rows = collapse(lineDiff("a\nb\nc", "a\nb\nC"), 1);
    expect(rows.every((r) => r.type === "line")).toBe(true);
  });
});

describe("computeDiff", () => {
  it("falls back to a whole replacement when the diff runs out of time", () => {
    const old = Array.from({ length: 3000 }, (_, i) => `a${i}`).join("\n");
    const next = Array.from({ length: 3000 }, (_, i) => `b${i}`).join("\n");
    const d = computeDiff(old, next, 0);
    expect(d.approximate).toBe(true);
    expect(diffStats(d.lines)).toEqual({ added: 3000, removed: 3000 });
  });

  it("is exact within the budget", () => {
    expect(computeDiff("a\nb", "a\nc").approximate).toBe(false);
  });
});

describe("stableStringify with hostile keys", () => {
  it("keeps a __proto__ key from the spec", () => {
    const value = JSON.parse('{"b":1,"__proto__":{"x":1},"a":2}');
    expect(stableStringify(value)).toContain('"__proto__"');
  });
});
