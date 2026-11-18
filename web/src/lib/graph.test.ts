import { describe, expect, it } from "vitest";
import type { CompareResult, JsonObject } from "./api";
import { buildGraph, extractRefs, schemaRefName, selectView, type ViewOptions } from "./graph";

const counts = { added: 0, removed: 0, changed: 0, unchanged: 0 };

const oas3: JsonObject = {
  openapi: "3.0.0",
  info: { title: "T", version: "1" },
  paths: {
    "/pets": {
      parameters: [{ $ref: "#/components/parameters/Limit" }],
      get: {
        responses: {
          "200": { content: { "application/json": { schema: { type: "array", items: { $ref: "#/components/schemas/Pet" } } } } },
          default: { $ref: "#/components/responses/Error" },
        },
      },
      post: { requestBody: { $ref: "#/components/requestBodies/NewPet" }, responses: { "201": { description: "ok" } } },
    },
  },
  components: {
    parameters: { Limit: { name: "limit", in: "query", schema: { $ref: "#/components/schemas/Paging" } } },
    responses: { Error: { content: { "application/json": { schema: { $ref: "#/components/schemas/Error" } } } } },
    requestBodies: { NewPet: { content: { "application/json": { schema: { $ref: "#/components/schemas/Pet" } } } } },
    schemas: {
      Pet: {
        type: "object",
        properties: {
          owner: { $ref: "#/components/schemas/Owner" },
          // A property named "example" is still walked; a self-ref is no edge.
          example: { $ref: "#/components/schemas/Tag" },
          parent: { $ref: "#/components/schemas/Pet" },
        },
      },
      Owner: { type: "object" },
      Tag: { type: "string" },
      Paging: { type: "integer" },
      Error: { type: "object" },
    },
  },
};

describe("schemaRefName", () => {
  it("reads OpenAPI 3 and Swagger 2 schema refs, including sub-paths and escapes", () => {
    expect(schemaRefName("#/components/schemas/Pet")).toBe("Pet");
    expect(schemaRefName("#/definitions/api.Repo")).toBe("api.Repo");
    expect(schemaRefName("#/components/schemas/Pet/properties/name")).toBe("Pet");
    expect(schemaRefName("#/components/schemas/a~1b")).toBe("a/b");
    expect(schemaRefName("#/components/responses/Error")).toBeUndefined();
  });
});

describe("extractRefs", () => {
  it("follows shared parameters, responses and request bodies to the schemas they use", () => {
    const r = extractRefs(oas3);
    expect([...r.ops.get("op:GET /pets")!.refs].sort()).toEqual(["Error", "Paging", "Pet"]);
    expect([...r.ops.get("op:POST /pets")!.refs].sort()).toEqual(["Paging", "Pet"]);
    expect([...r.schemas.get("Pet")!].sort()).toEqual(["Owner", "Tag"]);
    expect(r.schemas.get("Owner")!.size).toBe(0);
  });

  it("reads Swagger 2 definitions", () => {
    const r = extractRefs({
      swagger: "2.0",
      paths: { "/r": { get: { responses: { "200": { schema: { $ref: "#/definitions/Repo" } } } } } },
      definitions: { Repo: { properties: { owner: { $ref: "#/definitions/User" } } }, User: {} },
    });
    expect([...r.ops.get("op:GET /r")!.refs]).toEqual(["Repo"]);
    expect([...r.schemas.get("Repo")!]).toEqual(["User"]);
  });
});

function v2(): JsonObject {
  const doc = structuredClone(oas3);
  const schemas = (doc.components as JsonObject).schemas as JsonObject;
  delete schemas.Tag;
  (schemas.Pet as JsonObject).properties = {
    owner: { $ref: "#/components/schemas/Owner" },
    color: { $ref: "#/components/schemas/Color" },
  };
  schemas.Color = { type: "string" };
  return doc;
}

const compare: CompareResult = {
  baseline: { key: "k", version: "1", operations: 2, schemas: 5 },
  target: { key: "k", version: "2", operations: 2, schemas: 5 },
  summary: { operations: counts, schemas: counts },
  operations: { added: [], removed: [], changed: [{ method: "GET", path: "/pets", changes: { responses: {} } }] },
  schemas: {
    added: ["Color"],
    removed: ["Tag"],
    changed: [
      { name: "Pet", fieldsAdded: ["color"], fieldsRemoved: ["example", "parent"], definitionChanged: true },
      { name: "Owner", definitionChanged: true, extensionsOnly: true },
    ],
  },
};

const allOpts: ViewOptions = {
  mode: "changes",
  depth: 1,
  statuses: new Set(["added", "removed", "changed"]),
  includeNoise: false,
  neighbors: false,
  showOperations: true,
  showSchemas: true,
};

describe("buildGraph", () => {
  const g = buildGraph(extractRefs(oas3), extractRefs(v2()), compare);

  it("marks nodes with the compare result's status", () => {
    const st = (id: string) => g.byId.get(id)!.status;
    expect(st("op:GET /pets")).toBe("changed");
    expect(st("op:POST /pets")).toBe("unchanged");
    expect(st("schema:Color")).toBe("added");
    expect(st("schema:Tag")).toBe("removed");
    expect(g.byId.get("schema:Pet")!.notes).toEqual(["+color", "−example", "−parent"]);
    expect(g.byId.get("schema:Owner")!.noise).toBe(true);
  });

  it("marks edges that only exist in one version", () => {
    const edge = (s: string, t: string) => g.edges.find((e) => e.source === s && e.target === t)?.status;
    expect(edge("schema:Pet", "schema:Color")).toBe("added");
    expect(edge("schema:Pet", "schema:Tag")).toBe("removed");
    expect(edge("schema:Pet", "schema:Owner")).toBe("same");
    expect([...g.in.get("schema:Pet")!].sort()).toEqual(["op:GET /pets", "op:POST /pets"]);
  });

  it("without a compare, every node is unchanged", () => {
    const one = extractRefs(oas3);
    const same = buildGraph(one, one, null);
    expect(same.nodes.every((n) => n.status === "unchanged")).toBe(true);
    expect(same.edges.every((e) => e.status === "same")).toBe(true);
  });
});

describe("selectView", () => {
  const g = buildGraph(extractRefs(oas3), extractRefs(v2()), compare);
  const ids = (opts: Partial<ViewOptions>) =>
    selectView(g, { ...allOpts, ...opts })
      .nodes.map((n) => n.id)
      .sort();

  it("shows changes, leaving x-* only changes out unless asked", () => {
    expect(ids({})).toEqual(["op:GET /pets", "schema:Color", "schema:Pet", "schema:Tag"]);
    expect(ids({ includeNoise: true })).toContain("schema:Owner");
    expect(ids({ statuses: new Set(["added"]) })).toEqual(["schema:Color"]);
  });

  it("adds neighbors of changes as context", () => {
    const v = selectView(g, { ...allOpts, neighbors: true });
    expect(v.nodes.map((n) => n.id)).toContain("op:POST /pets");
    expect(v.primary.has("op:POST /pets")).toBe(false);
    expect(v.primary.has("schema:Pet")).toBe(true);
  });

  it("focuses on a node's neighborhood in both directions", () => {
    expect(ids({ focus: "schema:Owner" })).toEqual(["schema:Owner", "schema:Pet"]);
    expect(ids({ focus: "schema:Owner", depth: 2 })).toEqual([
      "op:GET /pets",
      "op:POST /pets",
      "schema:Color",
      "schema:Owner",
      "schema:Pet",
      "schema:Tag",
    ]);
  });

  it("filters by node kind but keeps the focused node and only edges between shown nodes", () => {
    const v = selectView(g, { ...allOpts, mode: "all", showOperations: false });
    expect(v.nodes.some((n) => n.kind === "operation")).toBe(false);
    expect(v.edges.every((e) => e.source.startsWith("schema:"))).toBe(true);
    expect(ids({ focus: "op:POST /pets", showOperations: false })).toContain("op:POST /pets");
  });
});
