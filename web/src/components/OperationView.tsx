import type { Json, JsonObject, OperationDetail } from "../lib/api";
import { Method } from "./Method";
import { SchemaTree } from "./SchemaTree";

function isObject(v: Json | undefined): v is JsonObject {
  return !!v && typeof v === "object" && !Array.isArray(v);
}

function statusClass(code: string): string {
  return `status status-${code.charAt(0)}`;
}

interface Props {
  op: OperationDetail;
  onRef?: (schema: string) => void;
}

/** Human-readable rendering of one operation from the `show` endpoint. */
export function OperationView({ op, onRef }: Props) {
  const params = (op.parameters ?? []).filter(isObject);
  const bodyParams = params.filter((p) => p.in === "body");
  const otherParams = params.filter((p) => p.in !== "body");
  const responses = Object.entries(op.responses ?? {}).sort(([a], [b]) => a.localeCompare(b));

  return (
    <article className="operation">
      <div className="detail-title">
        <Method method={op.method} />
        <span className="result-path">{op.path}</span>
      </div>
      {op.summary && <p className="detail-summary">{op.summary}</p>}
      <div className="meta-row">
        {op.operationId && (
          <span>
            operationId <strong className="mono">{op.operationId}</strong>
          </span>
        )}
        {op.tags?.map((t) => (
          <span key={t} className="tag">
            {t}
          </span>
        ))}
      </div>
      {op.description && <p className="detail-desc">{op.description}</p>}

      {otherParams.length > 0 && (
        <section className="section">
          <h3>Parameters</h3>
          <div className="table-wrap">
            <table className="params">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>In</th>
                  <th>Type</th>
                  <th>Description</th>
                </tr>
              </thead>
              <tbody>
                {otherParams.map((p, i) => (
                  <tr key={`${String(p.in)}:${String(p.name ?? p.$ref ?? i)}`}>
                    <td className="name">
                      {typeof p.$ref === "string" ? p.$ref.slice(p.$ref.lastIndexOf("/") + 1) : String(p.name ?? "")}
                      {p.required === true && <span className="req">required</span>}
                    </td>
                    <td>{String(p.in ?? "")}</td>
                    <td>
                      {isObject(p.schema) ? (
                        <SchemaTree schema={p.schema} onRef={onRef} maxDepth={2} />
                      ) : (
                        <span className="type mono">{String(p.type ?? "")}</span>
                      )}
                    </td>
                    <td className="muted">{typeof p.description === "string" ? p.description : ""}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {(op.requestBody || bodyParams.length > 0) && (
        <section className="section">
          <h3>
            Request body
            {(op.requestBody?.required === true || bodyParams.some((p) => p.required === true)) && (
              <span className="req">required</span>
            )}
          </h3>
          {typeof op.requestBody?.description === "string" && (
            <p className="detail-desc">{op.requestBody.description}</p>
          )}
          {isObject(op.requestBody?.content) &&
            Object.entries(op.requestBody.content).map(([ct, media]) => (
              <div key={ct}>
                <div className="content-type">{ct}</div>
                <SchemaTree schema={isObject(media) ? media.schema : undefined} onRef={onRef} />
              </div>
            ))}
          {bodyParams.map((p, i) => (
            <SchemaTree key={i} schema={p.schema} onRef={onRef} />
          ))}
        </section>
      )}

      {responses.length > 0 && (
        <section className="section">
          <h3>Responses</h3>
          {responses.map(([code, resp]) => (
            <details className="response" key={code} open={code.startsWith("2")}>
              <summary>
                <span className={statusClass(code)}>{code}</span>
                <span className="muted">{typeof resp.description === "string" ? resp.description : ""}</span>
              </summary>
              <div className="response-body">
                {isObject(resp.schema) && <SchemaTree schema={resp.schema} onRef={onRef} />}
                {isObject(resp.content) &&
                  Object.entries(resp.content).map(([ct, media]) => (
                    <div key={ct}>
                      <div className="content-type">{ct}</div>
                      <SchemaTree schema={isObject(media) ? media.schema : undefined} onRef={onRef} />
                    </div>
                  ))}
                {typeof resp.$ref === "string" && <span className="ref">{resp.$ref}</span>}
                {!isObject(resp.schema) && !isObject(resp.content) && typeof resp.$ref !== "string" && (
                  <span className="muted">No body</span>
                )}
              </div>
            </details>
          ))}
        </section>
      )}

      <details className="raw">
        <summary>Raw JSON</summary>
        <pre className="code">{JSON.stringify(op, null, 2)}</pre>
      </details>
    </article>
  );
}
