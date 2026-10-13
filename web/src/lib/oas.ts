// Turns a full OpenAPI / Swagger document into the structure the Reference tab
// renders: tag groups of operations, component schemas, and servers.
import type { Json, JsonObject, OperationDetail } from "./api";

const METHODS = ["get", "put", "post", "delete", "options", "head", "patch", "trace"];

export interface TagGroup {
  name: string;
  description?: string;
  operations: OperationDetail[];
}

export interface ReferenceModel {
  title: string;
  version: string;
  description?: string;
  format: string;
  servers: string[];
  groups: TagGroup[];
  schemas: Record<string, JsonObject>;
}

export const UNTAGGED = "default";

function isObject(v: Json | undefined): v is JsonObject {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

function str(v: Json | undefined): string | undefined {
  return typeof v === "string" ? v : undefined;
}

export function buildReference(doc: JsonObject): ReferenceModel {
  const info = isObject(doc.info) ? doc.info : {};
  const groups = new Map<string, TagGroup>();

  // Declared tags first, in document order, so the nav follows the author's intent.
  if (Array.isArray(doc.tags))
    for (const t of doc.tags)
      if (isObject(t) && typeof t.name === "string")
        groups.set(t.name, { name: t.name, description: str(t.description), operations: [] });

  const paths = isObject(doc.paths) ? doc.paths : {};
  const params = (v: Json | undefined) => (Array.isArray(v) ? v.filter(isObject).map((p) => resolveLocal(doc, p)) : []);
  for (const [path, item] of Object.entries(paths)) {
    if (!isObject(item)) continue;
    const shared = params(item.parameters);
    for (const m of METHODS) {
      const op = item[m];
      if (!isObject(op)) continue;
      const own = params(op.parameters);
      // Operation-level parameters override path-level ones with the same name+in.
      const ownKeys = new Set(own.map((p) => `${String(p.in)}:${String(p.name)}`));
      const parameters = [...shared.filter((p) => !ownKeys.has(`${String(p.in)}:${String(p.name)}`)), ...own];
      const tags = Array.isArray(op.tags) ? op.tags.map(String) : [];
      const detail: OperationDetail = {
        method: m.toUpperCase(),
        path,
        operationId: str(op.operationId),
        summary: str(op.summary),
        description: str(op.description),
        tags,
        parameters: parameters.length ? parameters : undefined,
        requestBody: isObject(op.requestBody) ? resolveLocal(doc, op.requestBody) : undefined,
        responses: isObject(op.responses) ? resolveResponses(doc, op.responses) : undefined,
      };
      for (const tag of tags.length ? tags : [UNTAGGED]) {
        if (!groups.has(tag)) groups.set(tag, { name: tag, operations: [] });
        groups.get(tag)!.operations.push(detail);
      }
    }
  }

  const components = isObject(doc.components) ? doc.components : {};
  const schemaSource = isObject(components.schemas) ? components.schemas : isObject(doc.definitions) ? doc.definitions : {};
  // No prototype: names come from the spec, so "constructor" or "__proto__"
  // must behave like any other model name.
  const schemas: Record<string, JsonObject> = Object.create(null);
  for (const name of Object.keys(schemaSource).sort()) {
    const s = schemaSource[name];
    if (isObject(s)) schemas[name] = s;
  }

  return {
    title: str(info.title) ?? "Untitled API",
    version: str(info.version) ?? "",
    description: str(info.description),
    format: str(doc.openapi) ? `OpenAPI ${String(doc.openapi)}` : str(doc.swagger) ? `Swagger ${String(doc.swagger)}` : "",
    servers: servers(doc),
    groups: [...groups.values()].filter((g) => g.operations.length > 0),
    schemas,
  };
}

/**
 * Follows a local $ref to a reusable parameter, response or request body
 * ("#/components/responses/NotFound", Swagger 2 "#/parameters/limit"). Only
 * applied to those objects, never to schemas: the views link schema refs.
 */
function resolveLocal(doc: JsonObject, v: JsonObject): JsonObject {
  let cur = v;
  for (let hops = 0; hops < 8 && typeof cur.$ref === "string" && cur.$ref.startsWith("#/"); hops++) {
    let node: Json | undefined = doc;
    for (const seg of cur.$ref.slice(2).split("/"))
      node = isObject(node) ? node[seg.replace(/~1/g, "/").replace(/~0/g, "~")] : undefined;
    if (!isObject(node)) break;
    cur = node;
  }
  return cur;
}

function resolveResponses(doc: JsonObject, responses: JsonObject): Record<string, JsonObject> {
  const out: Record<string, JsonObject> = {};
  for (const [code, r] of Object.entries(responses)) if (isObject(r)) out[code] = resolveLocal(doc, r);
  return out;
}

function servers(doc: JsonObject): string[] {
  if (Array.isArray(doc.servers)) return doc.servers.flatMap((s) => (isObject(s) && typeof s.url === "string" ? [s.url] : []));
  if (typeof doc.host === "string") {
    const schemes = Array.isArray(doc.schemes) && doc.schemes.length ? doc.schemes.map(String) : ["https"];
    const base = typeof doc.basePath === "string" ? doc.basePath : "";
    return schemes.map((s) => `${s}://${doc.host}${base}`);
  }
  return [];
}

/** Element id for an operation; method + path is unique, so the id is too. */
export function operationAnchor(op: { method: string; path: string }): string {
  return `op-${op.method.toLowerCase()}-${op.path.replace(/\s/g, "%20")}`;
}
