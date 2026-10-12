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
});
