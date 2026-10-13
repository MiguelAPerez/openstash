import { describe, expect, it } from "vitest";
import { groupSpecs } from "./specs";

describe("groupSpecs", () => {
  const e = (key: string, version: string) => ({ key, version, source: "", fetchedAt: "" });

  it("groups by key with the server's last entry as latest", () => {
    const groups = groupSpecs([e("b", "1.0.0"), e("a", "2.0.0"), e("a", "10.0.0")]);
    expect(groups.map((g) => g.key)).toEqual(["a", "b"]);
    expect(groups[0].latest.version).toBe("10.0.0");
  });

  it("keeps the server's version order instead of re-sorting", () => {
    // The server orders shorthand and prerelease versions; the client must not disagree.
    const groups = groupSpecs([e("a", "1.2"), e("a", "1.9"), e("a", "1.10"), e("a", "1.0.0-rc.10")]);
    expect(groups[0].versions.map((v) => v.version)).toEqual(["1.2", "1.9", "1.10", "1.0.0-rc.10"]);
    expect(groups[0].latest.version).toBe("1.0.0-rc.10");
  });
});
