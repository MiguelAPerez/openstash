import { createContext, useContext } from "react";
import type { SpecEntry } from "./api";

export interface SpecGroup {
  key: string;
  /** Oldest to newest. */
  versions: SpecEntry[];
  latest: SpecEntry;
}

/**
 * Groups entries by key. GET /v1/specs lists each key's versions oldest to
 * newest in the server's version order (the one the timeline and "latest" in
 * the API use), so that order is kept rather than re-derived here.
 */
export function groupSpecs(entries: SpecEntry[]): SpecGroup[] {
  const byKey = new Map<string, SpecEntry[]>();
  for (const e of entries) {
    const list = byKey.get(e.key) ?? [];
    list.push(e);
    byKey.set(e.key, list);
  }
  return [...byKey.entries()]
    .map(([key, versions]) => ({ key, versions, latest: versions[versions.length - 1] }))
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
