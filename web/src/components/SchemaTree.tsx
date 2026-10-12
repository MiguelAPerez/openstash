import type { Json, JsonObject } from "../lib/api";

function isObject(v: Json | undefined): v is JsonObject {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

function refName(ref: string): string {
  return ref.slice(ref.lastIndexOf("/") + 1);
}

function typeLabel(s: JsonObject): string {
  const t = s.type;
  let label = Array.isArray(t) ? t.join(" | ") : typeof t === "string" ? t : "";
  if (label === "array" && isObject(s.items)) {
    const inner = isObject(s.items) && typeof s.items.$ref === "string" ? refName(s.items.$ref) : typeLabel(s.items);
    label = `${inner || "any"}[]`;
  }
  if (typeof s.format === "string") label += label ? ` (${s.format})` : s.format;
  return label;
}

interface Props {
  schema: Json | undefined;
  onRef?: (name: string) => void;
  /** Guards against pathological nesting in very large specs. */
  maxDepth?: number;
}

/** Renders a (possibly $ref-expanded) JSON schema as a nested property tree. */
export function SchemaTree({ schema, onRef, maxDepth = 8 }: Props) {
  if (!isObject(schema)) return <span className="muted">—</span>;
  return (
    <div className="schema">
      <SchemaNode schema={schema} onRef={onRef} depth={0} maxDepth={maxDepth} />
    </div>
  );
}

function RefLink({ name, onRef }: { name: string; onRef?: (n: string) => void }) {
  if (!onRef) return <span className="ref">{name}</span>;
  return (
    <button type="button" className="ref link-btn" onClick={() => onRef(name)} title={`Open schema ${name}`}>
      {name}
    </button>
  );
}

function SchemaNode({
  schema,
  onRef,
  depth,
  maxDepth,
}: {
  schema: JsonObject;
  onRef?: (n: string) => void;
  depth: number;
  maxDepth: number;
}) {
  if (typeof schema.$ref === "string") {
    return <RefLink name={refName(schema.$ref)} onRef={onRef} />;
  }
  const from = typeof schema.$from === "string" ? schema.$from : null;
  const label = typeLabel(schema);
  const required = new Set(Array.isArray(schema.required) ? schema.required.map(String) : []);
  const props = schema.properties;
  const items = isObject(schema.items) ? schema.items : null;
  const composite = (["allOf", "oneOf", "anyOf"] as const).find((k) => Array.isArray(schema[k]));

  const head = (
    <>
      {from && <RefLink name={from} onRef={onRef} />}
      {label && <span className="type">{from ? ` ${label}` : label}</span>}
      {Array.isArray(schema.enum) && (
        <span className="enum"> enum: {schema.enum.map((e) => JSON.stringify(e)).join(", ")}</span>
      )}
      {typeof schema.description === "string" && depth === 0 && <div className="desc">{schema.description}</div>}
    </>
  );

  if (depth >= maxDepth) return head;

  return (
    <>
      {head}
      {/* Shallow schemas list property names only. */}
      {Array.isArray(props) && (
        <div className="props">
          {props.map((p) => (
            <span key={String(p)} className={`tag mono ${required.has(String(p)) ? "required" : ""}`}>
              {String(p)}
            </span>
          ))}
        </div>
      )}
      {isObject(props) && (
        <ul>
          {Object.entries(props).map(([name, child]) => (
            <li key={name}>
              <span className="prop">{name}</span>
              {required.has(name) && <span className="req">required</span>}{" "}
              {isObject(child) ? (
                <SchemaNode schema={child} onRef={onRef} depth={depth + 1} maxDepth={maxDepth} />
              ) : null}
              {isObject(child) && typeof child.description === "string" && (
                <div className="desc">{child.description}</div>
              )}
            </li>
          ))}
        </ul>
      )}
      {/* Array items with their own structure; a bare $ref is already in the type label. */}
      {items && (isObject(items.properties) || typeof items.$from === "string") && (
        <ul>
          <li>
            <span className="muted">items </span>
            <SchemaNode schema={items} onRef={onRef} depth={depth + 1} maxDepth={maxDepth} />
          </li>
        </ul>
      )}
      {composite && (
        <ul>
          {(schema[composite] as Json[]).map((s, i) => (
            <li key={i}>
              <span className="muted">{composite} </span>
              {isObject(s) && <SchemaNode schema={s} onRef={onRef} depth={depth + 1} maxDepth={maxDepth} />}
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
