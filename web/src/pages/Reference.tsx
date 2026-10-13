import { useEffect, useMemo, useState } from "react";
import { Method } from "../components/Method";
import { OperationView } from "../components/OperationView";
import { SchemaTree } from "../components/SchemaTree";
import { SpecHeader } from "../components/SpecHeader";
import { api } from "../lib/api";
import { buildReference, operationAnchor, type ReferenceModel } from "../lib/oas";
import { href, navigate, type Route } from "../lib/route";
import { useAsync } from "../lib/useAsync";

type ReferenceRoute = Extract<Route, { name: "reference" }>;

/**
 * Docs-style view of one stored version, built from the full document:
 * overview, operations grouped by tag, and component schemas ("models").
 */
export function Reference({ route }: { route: ReferenceRoute }) {
  const { key, version, query } = route;
  const tag = query.get("tag") ?? "";
  const op = query.get("op") ?? "";
  const schema = query.get("schema") ?? "";

  const doc = useAsync((signal) => api.dump(key, version, signal), [key, version]);
  const model = useMemo(() => (doc.data ? buildReference(doc.data) : null), [doc.data]);

  const go = (q: Record<string, string | undefined>) => navigate(href.reference(key, version, q));

  // Bring the requested operation (or the top of the section) into view.
  useEffect(() => {
    if (!model) return;
    const el = op ? document.getElementById(op) : null;
    if (el) el.scrollIntoView({ block: "start" });
    else window.scrollTo({ top: 0 });
  }, [model, tag, op, schema]);

  return (
    <>
      <SpecHeader specKey={key} version={version} tab="reference" />
      {doc.status === "loading" && <div className="skeleton">Loading {key}@{version}…</div>}
      {doc.status === "error" && <p className="notice">{doc.error.message}</p>}
      {model && (
        <div className="reference">
          <ReferenceNav model={model} tag={tag} op={op} schema={schema} specKey={key} version={version} />
          <div className="reference-body">
            {schema ? (
              <SchemaSection name={schema} model={model} onRef={(s) => go({ schema: s })} />
            ) : tag ? (
              <TagSection model={model} tag={tag} onRef={(s) => go({ schema: s })} />
            ) : (
              <Overview model={model} onTag={(t) => go({ tag: t })} onModels={() => go({ schema: Object.keys(model.schemas)[0] })} />
            )}
          </div>
        </div>
      )}
    </>
  );
}

function ReferenceNav({
  model,
  tag,
  op,
  schema,
  specKey,
  version,
}: {
  model: ReferenceModel;
  tag: string;
  op: string;
  schema: string;
  specKey: string;
  version: string;
}) {
  const [filter, setFilter] = useState("");
  const f = filter.trim().toLowerCase();
  const link = (q: Record<string, string | undefined>) => href.reference(specKey, version, q);
  const opMatches = (o: { method: string; path: string; summary?: string; operationId?: string }) =>
    !f || [o.method, o.path, o.summary ?? "", o.operationId ?? ""].some((s) => s.toLowerCase().includes(f));
  const schemaNames = Object.keys(model.schemas).filter((n) => !f || n.toLowerCase().includes(f));
  const modelCap = f ? 200 : 500;

  return (
    <nav className="reference-nav" aria-label="Reference contents">
      <input
        className="input"
        type="search"
        placeholder="Filter operations and models"
        aria-label="Filter operations and models"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />
      <a className="nav-link" href={link({})} aria-current={!tag && !schema ? "page" : undefined}>
        Overview
      </a>
      <div className="nav-heading">Operations</div>
      <ul className="nav-list">
        {model.groups.map((g) => {
          const ops = g.operations.filter(opMatches);
          if (f && ops.length === 0) return null;
          const open = g.name === tag || !!f;
          return (
            <li key={g.name}>
              <a className="nav-link" href={link({ tag: g.name })} aria-current={g.name === tag && !op ? "page" : undefined}>
                <span>{g.name}</span>
                <span className="nav-count">{g.operations.length}</span>
              </a>
              {open && (
                <ul className="nav-list nav-ops">
                  {ops.map((o) => {
                    const id = operationAnchor(o);
                    return (
                      <li key={id}>
                        <a
                          className="nav-link nav-op"
                          href={link({ tag: g.name, op: id })}
                          aria-current={g.name === tag && op === id ? "page" : undefined}
                        >
                          <Method method={o.method} />
                          <span className="nav-op-label">{o.summary || o.path}</span>
                        </a>
                      </li>
                    );
                  })}
                </ul>
              )}
            </li>
          );
        })}
      </ul>
      {Object.keys(model.schemas).length > 0 && (
        <>
          <div className="nav-heading">Models</div>
          <ul className="nav-list">
            {schemaNames.slice(0, modelCap).map((n) => (
              <li key={n}>
                <a className="nav-link mono" href={link({ schema: n })} aria-current={n === schema ? "page" : undefined}>
                  {n}
                </a>
              </li>
            ))}
          </ul>
          {schemaNames.length > modelCap && (
            <p className="nav-more muted small">
              {schemaNames.length - modelCap} more. Filter to find them.
            </p>
          )}
        </>
      )}
    </nav>
  );
}

function Overview({ model, onTag, onModels }: { model: ReferenceModel; onTag: (t: string) => void; onModels: () => void }) {
  const opCount = new Set(model.groups.flatMap((g) => g.operations.map((o) => `${o.method} ${o.path}`))).size;
  const schemaCount = Object.keys(model.schemas).length;
  return (
    <article>
      <h2 className="reference-title">{model.title}</h2>
      <div className="meta-row">
        {model.version && <span className="tag mono">v{model.version}</span>}
        {model.format && <span className="tag">{model.format}</span>}
        <span>
          {opCount} operations · {schemaCount} models
        </span>
      </div>
      {model.description && <p className="detail-desc">{model.description}</p>}
      {model.servers.length > 0 && (
        <section className="section">
          <h3>Servers</h3>
          <ul className="server-list">
            {model.servers.map((s) => (
              <li key={s} className="mono">
                {s}
              </li>
            ))}
          </ul>
        </section>
      )}
      <section className="section">
        <h3>Operations by tag</h3>
        <div className="cards">
          {model.groups.map((g) => (
            <button type="button" className="card card-button" key={g.name} onClick={() => onTag(g.name)}>
              <span className="card-title">{g.name}</span>
              <span className="muted small">{g.operations.length} operations</span>
              {g.description && <span className="muted small clamp">{g.description}</span>}
            </button>
          ))}
          {schemaCount > 0 && (
            <button type="button" className="card card-button" onClick={onModels}>
              <span className="card-title">Models</span>
              <span className="muted small">{schemaCount} schemas</span>
            </button>
          )}
        </div>
      </section>
    </article>
  );
}

function TagSection({ model, tag, onRef }: { model: ReferenceModel; tag: string; onRef: (s: string) => void }) {
  const group = model.groups.find((g) => g.name === tag);
  if (!group) return <p className="notice">No operations tagged “{tag}”.</p>;
  return (
    <>
      <h2 className="reference-title">{group.name}</h2>
      {group.description && <p className="detail-desc">{group.description}</p>}
      {group.operations.map((o) => (
        <div className="reference-op" key={operationAnchor(o)}>
          <OperationView op={o} onRef={onRef} showRaw={false} id={operationAnchor(o)} />
        </div>
      ))}
    </>
  );
}

function SchemaSection({ name, model, onRef }: { name: string; model: ReferenceModel; onRef: (s: string) => void }) {
  const schema = model.schemas[name];
  if (!schema) return <p className="notice">Model “{name}” not found.</p>;
  return (
    <article>
      <div className="detail-title">
        <h2 className="reference-title mono">{name}</h2>
        <span className="muted small">model</span>
      </div>
      <SchemaTree schema={schema} onRef={onRef} />
      <details className="raw">
        <summary>Raw JSON</summary>
        <pre className="code">{JSON.stringify(schema, null, 2)}</pre>
      </details>
    </article>
  );
}
