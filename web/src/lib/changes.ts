import type { CompareResult } from "./api";

export type ChangeKind = "added" | "removed" | "changed";

export interface ChangeItem {
  /** Stable id used in the URL: "op:GET /path" or "schema:Name". */
  id: string;
  kind: ChangeKind;
  type: "operation" | "schema";
  label: string;
  method?: string;
  path?: string;
  schema?: string;
  summary?: string;
  /** Field names that changed (operations) or property-level notes (schemas). */
  notes: string[];
}

const kindOrder: ChangeKind[] = ["changed", "added", "removed"];

/** Flattens a compare result into the ordered list the timeline steps through. */
export function changeItems(c: CompareResult): ChangeItem[] {
  const ops: ChangeItem[] = [];
  const schemas: ChangeItem[] = [];
  for (const kind of kindOrder) {
    if (kind === "changed") {
      for (const o of c.operations.changed ?? [])
        ops.push({
          id: `op:${o.method} ${o.path}`,
          kind,
          type: "operation",
          label: o.path,
          method: o.method,
          path: o.path,
          notes: Object.keys(o.changes).sort(),
        });
      for (const s of c.schemas.changed ?? []) {
        const notes = [
          ...(s.fieldsAdded ?? []).map((f) => `+${f}`),
          ...(s.fieldsRemoved ?? []).map((f) => `−${f}`),
        ];
        if (s.definitionChanged && notes.length === 0) notes.push("definition");
        schemas.push({ id: `schema:${s.name}`, kind, type: "schema", label: s.name, schema: s.name, notes });
      }
    } else {
      for (const o of c.operations[kind] ?? [])
        ops.push({
          id: `op:${o.method} ${o.path}`,
          kind,
          type: "operation",
          label: o.path,
          method: o.method,
          path: o.path,
          summary: o.summary,
          notes: [],
        });
      for (const name of c.schemas[kind] ?? [])
        schemas.push({ id: `schema:${name}`, kind, type: "schema", label: name, schema: name, notes: [] });
    }
  }
  return [...ops, ...schemas];
}

export function totalChanges(s: CompareResult["summary"]): number {
  const n = (c: { added: number; removed: number; changed: number }) => c.added + c.removed + c.changed;
  return n(s.operations) + n(s.schemas);
}
