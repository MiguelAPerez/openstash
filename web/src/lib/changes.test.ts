import { describe, expect, it } from "vitest";
import type { CompareResult } from "./api";
import { changeItems } from "./changes";

const counts = { added: 0, removed: 0, changed: 0, unchanged: 0 };

describe("changeItems", () => {
  it("orders operations before schemas, changed before added before removed", () => {
    const c: CompareResult = {
      baseline: { key: "k", version: "1", operations: 0, schemas: 0 },
      target: { key: "k", version: "2", operations: 0, schemas: 0 },
      summary: { operations: counts, schemas: counts },
      operations: {
        added: [{ method: "POST", path: "/a" }],
        removed: null,
        changed: [{ method: "GET", path: "/b", changes: { responses: {}, summary: { baseline: "x", target: "y" } } }],
      },
      schemas: { added: ["New"], removed: ["Old"], changed: [{ name: "Pet", fieldsAdded: ["tag"], definitionChanged: true }] },
    };
    const items = changeItems(c);
    expect(items.map((i) => `${i.kind}:${i.id}`)).toEqual([
      "changed:op:GET /b",
      "added:op:POST /a",
      "changed:schema:Pet",
      "added:schema:New",
      "removed:schema:Old",
    ]);
    expect(items[0].notes).toEqual(["responses", "summary"]);
    expect(items[2].notes).toEqual(["+tag"]);
  });

  it("lists extension-only changes after real ones of the same kind", () => {
    const c: CompareResult = {
      baseline: { key: "k", version: "1", operations: 0, schemas: 0 },
      target: { key: "k", version: "2", operations: 0, schemas: 0 },
      summary: { operations: counts, schemas: counts },
      operations: { added: [], removed: [], changed: [] },
      schemas: {
        added: ["New"],
        removed: [],
        changed: [
          { name: "A", definitionChanged: true, extensionsOnly: true },
          { name: "B", definitionChanged: true },
          { name: "C", fieldsAdded: ["x"] },
        ],
      },
    };
    const items = changeItems(c);
    expect(items.map((i) => i.id)).toEqual(["schema:B", "schema:C", "schema:A", "schema:New"]);
    expect(items[2].notes).toEqual(["x-*"]);
    expect(items[2].extensionsOnly).toBe(true);
  });
});
