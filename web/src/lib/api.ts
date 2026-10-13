// Typed client for the openstash serve API (api/serve.openapi.yaml).

export interface SpecEntry {
  key: string;
  version: string;
  source: string;
  endpoint?: string;
  fetchedAt: string;
  specVersion?: string;
}

export interface OperationIndex {
  method: string;
  path: string;
  operationId?: string;
  summary?: string;
  description?: string;
  tags?: string[];
}

export interface Hit {
  score: number;
  operation: OperationIndex;
}

export interface SchemaIndex {
  name: string;
  type?: string;
  title?: string;
  description?: string;
  properties?: string[];
  required?: string[];
}

export interface SchemaHit {
  score: number;
  schema: SchemaIndex;
}

export interface SearchResult {
  hits?: Hit[];
  schemas?: SchemaHit[];
}

export type Json = null | boolean | number | string | Json[] | { [key: string]: Json };
export type JsonObject = { [key: string]: Json };

export interface OperationDetail extends OperationIndex {
  parameters?: JsonObject[];
  requestBody?: JsonObject;
  responses?: Record<string, JsonObject>;
}

export interface CompareCounts {
  added: number;
  removed: number;
  changed: number;
  unchanged: number;
}

export interface CompareSummary {
  operations: CompareCounts;
  schemas: CompareCounts;
}

export interface TimelineVersion {
  version: string;
  title?: string;
  infoVersion?: string;
  source: string;
  fetchedAt: string;
  operations: number;
  schemas: number;
}

export interface TimelineStep {
  from: string;
  to: string;
  summary: CompareSummary;
}

export interface Timeline {
  key: string;
  versions: TimelineVersion[];
  steps: TimelineStep[];
}

export interface FieldChange {
  baseline?: Json;
  target?: Json;
}

export interface OperationChange {
  method: string;
  path: string;
  changes: Record<string, FieldChange>;
  extensionsOnly?: boolean;
}

export interface SchemaDiff {
  name: string;
  fieldsAdded?: string[];
  fieldsRemoved?: string[];
  definitionChanged?: boolean;
  extensionsOnly?: boolean;
}

export interface CompareResult {
  baseline: { key: string; version: string; operations: number; schemas: number };
  target: { key: string; version: string; operations: number; schemas: number };
  summary: CompareSummary;
  operations: { added: OperationIndex[] | null; removed: OperationIndex[] | null; changed: OperationChange[] | null };
  schemas: { added: string[] | null; removed: string[] | null; changed: SchemaDiff[] | null };
}

export interface DefinitionPair {
  baseline: Json;
  target: Json;
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

type Params = Record<string, string | number | boolean | undefined | string[]>;

function seg(s: string): string {
  return encodeURIComponent(s);
}

export function apiUrl(path: string, params: Params = {}): string {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === "") continue;
    if (Array.isArray(v)) v.forEach((item) => qs.append(k, item));
    else qs.set(k, String(v));
  }
  const q = qs.toString();
  return q ? `${path}?${q}` : path;
}

async function request<T>(path: string, params?: Params, init?: RequestInit): Promise<T> {
  const res = await fetch(apiUrl(path, params), init);
  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    // non-JSON body; handled below
  }
  if (!res.ok) {
    const msg =
      body && typeof body === "object" && "error" in body ? String((body as { error: unknown }).error) : res.statusText;
    throw new ApiError(msg || `HTTP ${res.status}`, res.status);
  }
  return body as T;
}

export const api = {
  listSpecs: () => request<{ entries: SpecEntry[] | null }>("/v1/specs").then((r) => r.entries ?? []),

  addSpec: (body: { key: string; from: string; version?: string; endpoint?: string }) =>
    request<{ status: string; indexed: number; schemas: number; meta: SpecEntry }>("/v1/specs", undefined, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),

  specUrl: (key: string, version: string) => `/v1/specs/${seg(key)}/versions/${seg(version)}`,

  dump: (key: string, version: string, signal?: AbortSignal) =>
    request<JsonObject>(api.specUrl(key, version), undefined, { signal }),

  search: (
    key: string,
    version: string,
    p: { q?: string; limit?: number; pathPrefix?: string; method?: string; in?: "paths" | "schemas" },
    signal?: AbortSignal,
  ) => request<SearchResult>(`${api.specUrl(key, version)}/operations`, p, { signal }),

  show: (key: string, version: string, method: string, path: string, depth: number, signal?: AbortSignal) =>
    request<{ operation: OperationDetail }>(
      `${api.specUrl(key, version)}/operations`,
      { detail: "show", method, path, depth },
      { signal },
    ).then((r) => r.operation),

  schema: (key: string, version: string, name: string, depth: number, signal?: AbortSignal) =>
    request<{ schema: JsonObject }>(`${api.specUrl(key, version)}/schemas/${seg(name)}`, { depth }, { signal }).then(
      (r) => r.schema,
    ),

  timeline: (key: string, signal?: AbortSignal) =>
    request<Timeline>(`/v1/specs/${seg(key)}/timeline`, undefined, { signal }),

  compare: (key: string, from: string, to: string, signal?: AbortSignal) =>
    request<CompareResult>(`/v1/specs/${seg(key)}/compare`, { from, to }, { signal }),

  definition: (
    key: string,
    from: string,
    to: string,
    item: { method: string; path: string } | { schema: string },
    signal?: AbortSignal,
  ) => request<DefinitionPair>(`/v1/specs/${seg(key)}/compare/definition`, { from, to, ...item }, { signal }),
};
