// Dependency graph of a spec: endpoints link to the component schemas they use
// (anywhere in parameters, request body or responses, through shared
// parameters/responses/request bodies), and schemas link to the schemas they
// $ref. Two versions are merged into one graph whose nodes and edges carry
// their change status, so the Graph tab can show what a change touches.
import type { CompareResult, Json, JsonObject } from "./api";
import { changeItems } from "./changes";

const METHODS = ["get", "put", "post", "delete", "options", "head", "patch", "trace"];
const SCHEMA_PREFIXES = ["#/components/schemas/", "#/definitions/"];

export type NodeKind = "operation" | "schema";
export type NodeStatus = "added" | "removed" | "changed" | "unchanged";
export type EdgeStatus = "added" | "removed" | "same";

export interface GraphNode {
  /** Same ids as the timeline: "op:GET /path" or "schema:Name". */
  id: string;
  kind: NodeKind;
  label: string;
  method?: string;
  path?: string;
  schema?: string;
  summary?: string;
  status: NodeStatus;
  /** Changed, but only in x-* vendor extensions. */
  noise?: boolean;
  /** What changed, from the compare result. */
  notes: string[];
}

export interface GraphEdge {
  source: string;
  target: string;
  status: EdgeStatus;
}

export interface SpecGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  byId: Map<string, GraphNode>;
  /** Outgoing ("uses") and incoming ("used by") neighbor ids. */
  out: Map<string, Set<string>>;
  in: Map<string, Set<string>>;
}

interface OpRefs {
  method: string;
  path: string;
  summary?: string;
  refs: Set<string>;
}

export interface DocRefs {
  ops: Map<string, OpRefs>;
  schemas: Map<string, Set<string>>;
}

function isObject(v: Json | undefined): v is JsonObject {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

function unescapePointer(seg: string): string {
  let s = seg.replace(/~1/g, "/").replace(/~0/g, "~");
  try {
    s = decodeURIComponent(s);
  } catch {
    // keep as is
  }
  return s;
}

/** Schema name a $ref points at, or undefined when it isn't a component schema. */
export function schemaRefName(ref: string): string | undefined {
  for (const p of SCHEMA_PREFIXES)
    if (ref.startsWith(p)) {
      const rest = ref.slice(p.length);
      // "#/components/schemas/Pet/properties/name" still depends on Pet.
      return unescapePointer(rest.split("/")[0]);
    }
  return undefined;
}

function resolvePointer(doc: JsonObject, ref: string): Json | undefined {
  let node: Json | undefined = doc;
  for (const seg of ref.slice(2).split("/")) node = isObject(node) ? node[unescapePointer(seg)] : undefined;
  return node;
}

/**
 * Collects the component schemas v refers to. Local refs to anything other
 * than a schema (shared parameters, responses, request bodies, headers) are
 * followed; schema refs are recorded, not followed, so only direct
 * dependencies become edges.
 */
function collectSchemaRefs(doc: JsonObject, v: Json | undefined, into: Set<string>, seen = new Set<string>()) {
  if (Array.isArray(v)) {
    for (const item of v) collectSchemaRefs(doc, item, into, seen);
    return;
  }
  if (!isObject(v)) return;
  for (const [k, child] of Object.entries(v)) {
    if (k === "$ref" && typeof child === "string") {
      const name = schemaRefName(child);
      if (name !== undefined) into.add(name);
      else if (child.startsWith("#/") && !seen.has(child)) {
        seen.add(child);
        collectSchemaRefs(doc, resolvePointer(doc, child), into, seen);
      }
    } else {
      collectSchemaRefs(doc, child, into, seen);
    }
  }
}

/** Schema references of every operation and component schema in doc. */
export function extractRefs(doc: JsonObject): DocRefs {
  const ops = new Map<string, OpRefs>();
  const paths = isObject(doc.paths) ? doc.paths : {};
  for (const [path, item] of Object.entries(paths)) {
    if (!isObject(item)) continue;
    for (const m of METHODS) {
      const op = item[m];
      if (!isObject(op)) continue;
      const refs = new Set<string>();
      collectSchemaRefs(doc, item.parameters, refs);
      collectSchemaRefs(doc, op, refs);
      const method = m.toUpperCase();
      ops.set(`op:${method} ${path}`, {
        method,
        path,
        summary: typeof op.summary === "string" ? op.summary : undefined,
        refs,
      });
    }
  }

  const components = isObject(doc.components) ? doc.components : {};
  const source = isObject(components.schemas) ? components.schemas : isObject(doc.definitions) ? doc.definitions : {};
  const schemas = new Map<string, Set<string>>();
  for (const [name, def] of Object.entries(source)) {
    const refs = new Set<string>();
    collectSchemaRefs(doc, def, refs);
    refs.delete(name);
    schemas.set(name, refs);
  }
  return { ops, schemas };
}

/**
 * Merges the dependency graphs of two versions. Node status comes from the
 * compare result (the same definition-aware changes the timeline lists);
 * nodes it doesn't mention fall back to presence in either version. With
 * from === to (or no compare), every node is unchanged.
 */
export function buildGraph(from: DocRefs, to: DocRefs, compare: CompareResult | null): SpecGraph {
  const changes = new Map(compare ? changeItems(compare).map((c) => [c.id, c]) : []);
  const byId = new Map<string, GraphNode>();

  const status = (id: string, inFrom: boolean, inTo: boolean): Pick<GraphNode, "status" | "noise" | "notes"> => {
    const c = changes.get(id);
    if (c) return { status: c.kind, noise: c.extensionsOnly || undefined, notes: c.notes };
    if (compare && inTo && !inFrom) return { status: "added", notes: [] };
    if (compare && inFrom && !inTo) return { status: "removed", notes: [] };
    return { status: "unchanged", notes: [] };
  };

  for (const id of new Set([...from.ops.keys(), ...to.ops.keys()])) {
    const op = (to.ops.get(id) ?? from.ops.get(id))!;
    byId.set(id, {
      id,
      kind: "operation",
      label: `${op.method} ${op.path}`,
      method: op.method,
      path: op.path,
      summary: op.summary,
      ...status(id, from.ops.has(id), to.ops.has(id)),
    });
  }
  for (const name of new Set([...from.schemas.keys(), ...to.schemas.keys()])) {
    const id = `schema:${name}`;
    byId.set(id, {
      id,
      kind: "schema",
      label: name,
      schema: name,
      ...status(id, from.schemas.has(name), to.schemas.has(name)),
    });
  }

  const edgeKeys = (refs: DocRefs) => {
    const keys = new Set<string>();
    for (const [id, op] of refs.ops) for (const s of op.refs) keys.add(`${id}\n schema:${s}`);
    for (const [name, deps] of refs.schemas) for (const s of deps) keys.add(`schema:${name}\n schema:${s}`);
    return keys;
  };
  const fromEdges = edgeKeys(from);
  const toEdges = edgeKeys(to);
  const edges: GraphEdge[] = [];
  const out = new Map<string, Set<string>>();
  const inc = new Map<string, Set<string>>();
  for (const key of new Set([...fromEdges, ...toEdges])) {
    const [source, target] = key.split("\n ");
    // A $ref to a schema the document doesn't define has nothing to point at.
    if (!byId.has(source) || !byId.has(target)) continue;
    const inFrom = fromEdges.has(key);
    const inTo = toEdges.has(key);
    edges.push({ source, target, status: inFrom && inTo ? "same" : inTo ? "added" : "removed" });
    if (!out.has(source)) out.set(source, new Set());
    out.get(source)!.add(target);
    if (!inc.has(target)) inc.set(target, new Set());
    inc.get(target)!.add(source);
  }

  return { nodes: [...byId.values()], edges, byId, out, in: inc };
}

export type ChangeStatus = Exclude<NodeStatus, "unchanged">;

export interface ViewOptions {
  /** "changes": changed nodes (and optionally their neighbors); "all": everything. */
  mode: "changes" | "all";
  /** Show only this node's neighborhood, depth hops out in either direction. */
  focus?: string;
  depth: number;
  statuses: Set<ChangeStatus>;
  /** Count x-* only changes as changes. */
  includeNoise: boolean;
  /** In "changes" mode, also show the direct neighbors of changed nodes. */
  neighbors: boolean;
  showOperations: boolean;
  showSchemas: boolean;
}

export interface GraphView {
  nodes: GraphNode[];
  edges: GraphEdge[];
  /** Nodes that matched the view itself, as opposed to context neighbors. */
  primary: Set<string>;
}

function neighborsOf(g: SpecGraph, id: string): string[] {
  return [...(g.out.get(id) ?? []), ...(g.in.get(id) ?? [])];
}

export function isChange(n: GraphNode, opts: Pick<ViewOptions, "statuses" | "includeNoise">): boolean {
  return n.status !== "unchanged" && opts.statuses.has(n.status) && (opts.includeNoise || !n.noise);
}

/** Picks the nodes and edges to draw for the given options. */
export function selectView(g: SpecGraph, opts: ViewOptions): GraphView {
  const visible = new Set<string>();
  const primary = new Set<string>();

  if (opts.focus && g.byId.has(opts.focus)) {
    primary.add(opts.focus);
    let frontier = [opts.focus];
    visible.add(opts.focus);
    for (let d = 0; d < opts.depth; d++) {
      const next: string[] = [];
      for (const id of frontier)
        for (const n of neighborsOf(g, id))
          if (!visible.has(n)) {
            visible.add(n);
            next.push(n);
          }
      frontier = next;
    }
  } else if (opts.mode === "all") {
    for (const n of g.nodes) {
      visible.add(n.id);
      primary.add(n.id);
    }
  } else {
    for (const n of g.nodes)
      if (isChange(n, opts)) {
        visible.add(n.id);
        primary.add(n.id);
      }
    if (opts.neighbors) for (const id of primary) for (const n of neighborsOf(g, id)) visible.add(n);
  }

  const keep = (n: GraphNode) =>
    n.id === opts.focus || (n.kind === "operation" ? opts.showOperations : opts.showSchemas);
  const nodes = g.nodes.filter((n) => visible.has(n.id) && keep(n));
  const ids = new Set(nodes.map((n) => n.id));
  const edges = g.edges.filter((e) => ids.has(e.source) && ids.has(e.target));
  for (const id of primary) if (!ids.has(id)) primary.delete(id);
  return { nodes, edges, primary };
}
