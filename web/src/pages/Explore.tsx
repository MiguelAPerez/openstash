import { useEffect, useRef, useState } from "react";
import { Method } from "../components/Method";
import { OperationView } from "../components/OperationView";
import { SchemaTree } from "../components/SchemaTree";
import { SpecHeader } from "../components/SpecHeader";
import { api } from "../lib/api";
import { href, navigate, type Route } from "../lib/route";
import { useAsync, useDebounced } from "../lib/useAsync";

type ExploreRoute = Extract<Route, { name: "explore" }>;

const METHODS = ["", "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];
const PAGE = 25;

export function Explore({ route }: { route: ExploreRoute }) {
  const { key, version, query } = route;
  const q = query.get("q") ?? "";
  const method = query.get("method") ?? "";
  const prefix = query.get("prefix") ?? "";
  const scope = query.get("in") === "schemas" ? "schemas" : "paths";
  const op = query.get("op") ?? ""; // "GET /path"
  const schema = query.get("schema") ?? "";
  const depth = Number(query.get("depth") ?? "1") || 0;

  // Local input state so typing is instant; the URL follows (debounced).
  const [text, setText] = useState(q);
  const [prefixText, setPrefixText] = useState(prefix);
  const [limit, setLimit] = useState(PAGE);
  const debouncedText = useDebounced(text, 200);
  const debouncedPrefix = useDebounced(prefixText, 300);
  const searchRef = useRef<HTMLInputElement>(null);

  const update = (patch: Record<string, string | undefined>, replace = true) => {
    const next: Record<string, string | undefined> = { q, method, prefix, in: scope === "schemas" ? "schemas" : undefined, op, schema, depth: depth === 1 ? undefined : String(depth), ...patch };
    navigate(href.explore(key, version, next), replace);
  };

  // Reset inputs when switching specs/versions.
  useEffect(() => {
    setText(q);
    setPrefixText(prefix);
    setLimit(PAGE);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, version]);

  useEffect(() => {
    if (debouncedText !== q || debouncedPrefix !== prefix) update({ q: debouncedText, prefix: debouncedPrefix });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedText, debouncedPrefix]);

  useEffect(() => setLimit(PAGE), [q, method, prefix, scope]);

  const results = useAsync(
    (signal) => api.search(key, version, { q, method, pathPrefix: prefix, in: scope, limit: limit + 1 }, signal),
    [key, version, q, method, prefix, scope, limit],
    true,
  );

  const [opMethod, ...opPathParts] = op.split(" ");
  const opPath = opPathParts.join(" ");
  const detail = useAsync(
    (signal) => (op ? api.show(key, version, opMethod, opPath, depth, signal) : Promise.resolve(null)),
    [key, version, op, depth],
  );
  const schemaDetail = useAsync(
    (signal) => (schema ? api.schema(key, version, schema, depth, signal) : Promise.resolve(null)),
    [key, version, schema, depth],
  );

  const hits = results.data?.hits ?? [];
  const schemaHits = results.data?.schemas ?? [];
  const list = scope === "paths" ? hits.slice(0, limit) : schemaHits.slice(0, limit);
  const hasMore = (scope === "paths" ? hits.length : schemaHits.length) > limit;
  const opIds = hits.slice(0, limit).map((h) => `${h.operation.method} ${h.operation.path}`);

  const selectOp = (id: string) => update({ op: id, schema: undefined }, false);
  const selectSchema = (name: string) => update({ schema: name, op: undefined }, false);

  // Keyboard: "/" focuses search, ↑/↓ move through operation results.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const typing = target.tagName === "INPUT" || target.tagName === "SELECT" || target.tagName === "TEXTAREA";
      if (e.key === "/" && !typing) {
        e.preventDefault();
        searchRef.current?.focus();
        return;
      }
      if ((e.key === "ArrowDown" || e.key === "ArrowUp") && (!typing || target === searchRef.current) && scope === "paths") {
        if (opIds.length === 0) return;
        e.preventDefault();
        const i = opIds.indexOf(op);
        const next = e.key === "ArrowDown" ? Math.min(opIds.length - 1, i + 1) : Math.max(0, i - 1);
        update({ op: opIds[next], schema: undefined });
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  return (
    <>
      <SpecHeader specKey={key} version={version} tab="explore" />

      <div className="searchbar" role="search">
        <input
          ref={searchRef}
          className="input q"
          type="search"
          placeholder={scope === "paths" ? "Search endpoints — e.g. user repos" : "Search schemas — e.g. Repository"}
          aria-label="Search"
          value={text}
          onChange={(e) => setText(e.target.value)}
          autoFocus
        />
        <div className="segmented" role="group" aria-label="Search in">
          <button type="button" aria-pressed={scope === "paths"} onClick={() => update({ in: undefined })}>
            Endpoints
          </button>
          <button type="button" aria-pressed={scope === "schemas"} onClick={() => update({ in: "schemas" })}>
            Schemas
          </button>
        </div>
        {scope === "paths" && (
          <>
            <select className="select" aria-label="Method" value={method} onChange={(e) => update({ method: e.target.value })}>
              {METHODS.map((m) => (
                <option key={m} value={m}>
                  {m || "Any method"}
                </option>
              ))}
            </select>
            <input
              className="input prefix mono"
              placeholder="/path/prefix"
              aria-label="Path prefix"
              value={prefixText}
              onChange={(e) => setPrefixText(e.target.value)}
            />
          </>
        )}
        <span className="hint muted">
          <span className="kbd">/</span> search <span className="kbd">↑</span>
          <span className="kbd">↓</span> move
        </span>
      </div>

      <div className="split">
        <section aria-label="Results">
          <div className="results-head">
            <span>
              {results.status === "loading" && !results.data
                ? "Searching…"
                : `${list.length}${hasMore ? "+" : ""} ${scope === "paths" ? "endpoints" : "schemas"}${q ? ` matching “${q}”` : ""}`}
            </span>
          </div>
          {results.status === "error" && <p className="notice">{results.error.message}</p>}
          <ul className="results">
            {scope === "paths" &&
              hits.slice(0, limit).map((h) => {
                const id = `${h.operation.method} ${h.operation.path}`;
                return (
                  <li key={id}>
                    <button type="button" className="result" aria-current={id === op} onClick={() => selectOp(id)}>
                      <div className="result-line">
                        <Method method={h.operation.method} />
                        <span className="result-path">{h.operation.path}</span>
                      </div>
                      {h.operation.summary && <div className="result-summary">{h.operation.summary}</div>}
                      {(h.operation.tags?.length || h.operation.operationId) && (
                        <div className="result-meta">
                          {h.operation.tags?.map((t) => (
                            <span className="tag" key={t}>
                              {t}
                            </span>
                          ))}
                          {h.operation.operationId && <span className="tag mono">{h.operation.operationId}</span>}
                        </div>
                      )}
                    </button>
                  </li>
                );
              })}
            {scope === "schemas" &&
              schemaHits.slice(0, limit).map((h) => (
                <li key={h.schema.name}>
                  <button
                    type="button"
                    className="result"
                    aria-current={h.schema.name === schema}
                    onClick={() => selectSchema(h.schema.name)}
                  >
                    <div className="result-line">
                      <span className="result-path">{h.schema.name}</span>
                      {h.schema.type && <span className="type mono small">{h.schema.type}</span>}
                    </div>
                    {(h.schema.title || h.schema.description) && (
                      <div className="result-summary clamp">{h.schema.title || h.schema.description}</div>
                    )}
                    {h.schema.properties && (
                      <div className="result-meta">
                        {h.schema.properties.slice(0, 8).map((p) => (
                          <span key={p} className="tag mono">
                            {p}
                          </span>
                        ))}
                        {h.schema.properties.length > 8 && <span className="tag">+{h.schema.properties.length - 8}</span>}
                      </div>
                    )}
                  </button>
                </li>
              ))}
          </ul>
          {hasMore && (
            <button type="button" className="btn load-more" onClick={() => setLimit((l) => l + PAGE)}>
              Show more
            </button>
          )}
        </section>

        <section className="detail" aria-label="Details">
          {(op || schema) && (
            <div className="detail-toolbar">
              <label htmlFor="depth">Inline $refs</label>
              <select
                id="depth"
                className="select"
                value={depth}
                onChange={(e) => update({ depth: e.target.value === "1" ? undefined : e.target.value })}
              >
                <option value={0}>Off (shallow)</option>
                <option value={1}>1 level</option>
                <option value={2}>2 levels</option>
                <option value={3}>3 levels</option>
              </select>
              <span className="spacer" />
              {op && (
                <a
                  className="btn btn-small"
                  href={`/v1/specs/${encodeURIComponent(key)}/versions/${encodeURIComponent(version)}/operations?detail=show&method=${encodeURIComponent(opMethod)}&path=${encodeURIComponent(opPath)}&depth=${depth}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  API JSON ↗
                </a>
              )}
            </div>
          )}
          {!op && !schema && (
            <div className="detail-placeholder">
              <div>
                <p>Select {scope === "paths" ? "an endpoint" : "a schema"} to see its details.</p>
              </div>
            </div>
          )}
          {op && detail.status === "loading" && <div className="skeleton">Loading…</div>}
          {op && detail.status === "error" && <p className="notice">{detail.error.message}</p>}
          {op && detail.status === "ok" && detail.data && <OperationView op={detail.data} onRef={selectSchema} />}
          {schema && schemaDetail.status === "loading" && <div className="skeleton">Loading…</div>}
          {schema && schemaDetail.status === "error" && <p className="notice">{schemaDetail.error.message}</p>}
          {schema && schemaDetail.status === "ok" && schemaDetail.data && (
            <article>
              <div className="detail-title">
                <span className="result-path">{schema}</span>
                <span className="muted small">schema</span>
              </div>
              <SchemaTree schema={schemaDetail.data} onRef={selectSchema} />
              <details className="raw">
                <summary>Raw JSON</summary>
                <pre className="code">{JSON.stringify(schemaDetail.data, null, 2)}</pre>
              </details>
            </article>
          )}
        </section>
      </div>
    </>
  );
}
