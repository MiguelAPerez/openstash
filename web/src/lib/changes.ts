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
  /** Only x-* vendor extensions differ (e.g. generator metadata). */
  extensionsOnly?: boolean;
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
          extensionsOnly: o.extensionsOnly,
        });
      for (const s of c.schemas.changed ?? []) {
        const notes = [
          ...(s.fieldsAdded ?? []).map((f) => `+${f}`),
          ...(s.fieldsRemoved ?? []).map((f) => `−${f}`),
        ];
        if (s.definitionChanged && notes.length === 0) notes.push(s.extensionsOnly ? "x-*" : "definition");
        schemas.push({
          id: `schema:${s.name}`,
          kind,
          type: "schema",
          label: s.name,
          schema: s.name,
          notes,
          extensionsOnly: s.extensionsOnly,
        });
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
  // Extension-only changes are mostly generator noise: list them after the
  // real changes of their kind (sort is stable, so server order holds within).
  const noiseLast = (list: ChangeItem[]) =>
    list.sort((a, b) =>
      a.kind === b.kind ? Number(!!a.extensionsOnly) - Number(!!b.extensionsOnly) : kindOrder.indexOf(a.kind) - kindOrder.indexOf(b.kind),
    );
  return [...noiseLast(ops), ...noiseLast(schemas)];
}

export function totalChanges(s: CompareResult["summary"]): number {
  const n = (c: { added: number; removed: number; changed: number }) => c.added + c.removed + c.changed;
  return n(s.operations) + n(s.schemas);
}
