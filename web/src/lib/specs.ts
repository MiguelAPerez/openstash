import { createContext, useContext } from "react";
import type { SpecEntry } from "./api";

export interface SpecGroup {
  key: string;
  /** Oldest to newest. */
  versions: SpecEntry[];
  latest: SpecEntry;
}

const semverRe = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$/;

/**
 * Orders versions like the server (internal/store/version.go): semver when
 * both parse, semver sorts after non-semver, lexical otherwise.
 */
export function compareVersions(a: string, b: string): number {
  const ma = semverRe.exec(a.trim());
  const mb = semverRe.exec(b.trim());
  if (ma && mb) {
    for (let i = 1; i <= 3; i++) {
      const d = Number(ma[i]) - Number(mb[i]);
      if (d !== 0) return Math.sign(d);
    }
    const pa = ma[4];
    const pb = mb[4];
    if (pa === pb) return 0;
    if (pa === undefined) return 1; // release > prerelease
    if (pb === undefined) return -1;
    return pa < pb ? -1 : 1;
  }
  if (ma) return 1;
  if (mb) return -1;
  return a < b ? -1 : a > b ? 1 : 0;
}

export function groupSpecs(entries: SpecEntry[]): SpecGroup[] {
  const byKey = new Map<string, SpecEntry[]>();
  for (const e of entries) {
    const list = byKey.get(e.key) ?? [];
    list.push(e);
    byKey.set(e.key, list);
  }
  return [...byKey.entries()]
    .map(([key, versions]) => {
      versions.sort((a, b) => compareVersions(a.version, b.version));
      return { key, versions, latest: versions[versions.length - 1] };
    })
    .sort((a, b) => a.key.localeCompare(b.key));
}

export interface SpecsContextValue {
  groups: SpecGroup[];
  loading: boolean;
  error?: Error;
  reload: () => void;
  openAdd: (prefillKey?: string) => void;
}

export const SpecsContext = createContext<SpecsContextValue>({
  groups: [],
  loading: true,
  reload: () => {},
  openAdd: () => {},
});

export const useSpecs = () => useContext(SpecsContext);

export function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

export function formatDateTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}
