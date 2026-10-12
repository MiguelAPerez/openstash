import { describe, expect, it } from "vitest";
import { compareVersions, groupSpecs } from "./specs";

describe("compareVersions", () => {
  it("orders semver numerically", () => {
    const vs = ["1.10.0", "1.2.0", "1.2.0-rc.1", "v0.9.1"];
    expect([...vs].sort(compareVersions)).toEqual(["v0.9.1", "1.2.0-rc.1", "1.2.0", "1.10.0"]);
  });
  it("puts non-semver before semver", () => {
    expect(["1.0.0", "latest", "2024-01"].sort(compareVersions)).toEqual(["2024-01", "latest", "1.0.0"]);
  });
});

describe("groupSpecs", () => {
  it("groups by key with latest last", () => {
    const e = (key: string, version: string) => ({ key, version, source: "", fetchedAt: "" });
    const groups = groupSpecs([e("b", "1.0.0"), e("a", "2.0.0"), e("a", "10.0.0")]);
    expect(groups.map((g) => g.key)).toEqual(["a", "b"]);
    expect(groups[0].latest.version).toBe("10.0.0");
  });
});
