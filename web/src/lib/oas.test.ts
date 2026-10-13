import { describe, expect, it } from "vitest";
import { buildReference, UNTAGGED } from "./oas";

describe("buildReference", () => {
  it("groups by declared tag order, then encountered tags, then untagged", () => {
    const m = buildReference({
      openapi: "3.1.0",
      info: { title: "T", version: "1" },
      tags: [{ name: "users", description: "People" }, { name: "unused" }],
      paths: {
        "/pets": { get: { tags: ["pets"] } },
        "/users": { get: { tags: ["users"] }, post: { tags: ["users", "admin"] } },
        "/health": { get: {} },
      },
    });
    expect(m.groups.map((g) => g.name)).toEqual(["users", "pets", "admin", UNTAGGED]);
    expect(m.groups[0].description).toBe("People");
    expect(m.groups[0].operations.map((o) => o.method)).toEqual(["GET", "POST"]);
    expect(m.format).toBe("OpenAPI 3.1.0");
  });

  it("merges path-level parameters with operation overrides", () => {
    const m = buildReference({
      paths: {
        "/x/{id}": {
          parameters: [
            { name: "id", in: "path", description: "shared" },
            { name: "trace", in: "header" },
          ],
          get: { parameters: [{ name: "id", in: "path", description: "own" }] },
        },
      },
    });
    const params = m.groups[0].operations[0].parameters!;
    expect(params.map((p) => `${p.name}:${p.description ?? ""}`)).toEqual(["trace:", "id:own"]);
  });

  it("resolves local refs to reusable parameters, responses and request bodies", () => {
    const m = buildReference({
      openapi: "3.0.3",
      components: {
        parameters: { Id: { name: "id", in: "path", required: true, schema: { type: "string" } } },
        responses: {
          NotFound: { description: "Missing" },
          Alias: { $ref: "#/components/responses/NotFound" },
          Pet: { description: "A pet", content: { "application/json": { schema: { $ref: "#/components/schemas/Pet" } } } },
        },
        requestBodies: { PetBody: { required: true, content: {} } },
      },
      paths: {
        "/pets/{id}": {
          parameters: [{ $ref: "#/components/parameters/Id" }],
          put: {
            parameters: [{ name: "dryRun", in: "query" }],
            requestBody: { $ref: "#/components/requestBodies/PetBody" },
            responses: {
              "200": { $ref: "#/components/responses/Pet" },
              "404": { $ref: "#/components/responses/Alias" },
              "500": { $ref: "other.yaml#/Error" },
            },
          },
        },
      },
    });
    const op = m.groups[0].operations[0];
    expect(op.parameters!.map((p) => `${p.in}:${p.name}`)).toEqual(["path:id", "query:dryRun"]);
    expect(op.requestBody!.required).toBe(true);
    expect(op.responses!["404"].description).toBe("Missing");
    // Schema refs inside a resolved response stay as refs for the views to link.
    expect(JSON.stringify(op.responses!["200"])).toContain('"$ref":"#/components/schemas/Pet"');
    // External refs are left alone.
    expect(op.responses!["500"].$ref).toBe("other.yaml#/Error");
  });

  it("resolves Swagger 2 #/responses refs", () => {
    const m = buildReference({
      swagger: "2.0",
      responses: { forbidden: { description: "Forbidden" } },
      paths: { "/x": { delete: { responses: { "403": { $ref: "#/responses/forbidden" } } } } },
    });
    expect(m.groups[0].operations[0].responses!["403"].description).toBe("Forbidden");
  });

  it("reads Swagger 2 servers and definitions", () => {
    const m = buildReference({
      swagger: "2.0",
      host: "api.example",
      basePath: "/v1",
      schemes: ["https", "http"],
      definitions: { B: { type: "object" }, A: { type: "string" } },
      paths: {},
    });
    expect(m.servers).toEqual(["https://api.example/v1", "http://api.example/v1"]);
    expect(Object.keys(m.schemas)).toEqual(["A", "B"]);
    expect(m.format).toBe("Swagger 2.0");
  });

  it("treats Object.prototype names as ordinary model names", () => {
    const m = buildReference(JSON.parse('{"definitions":{"__proto__":{"type":"object"},"Pet":{"type":"object"}},"paths":{}}'));
    expect(Object.keys(m.schemas)).toEqual(["Pet", "__proto__"]);
    expect(m.schemas["constructor"]).toBeUndefined();
    expect(m.schemas["toString"]).toBeUndefined();
  });
});
