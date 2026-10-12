import { href } from "../lib/route";
import { formatDate, useSpecs } from "../lib/specs";

export function Home() {
  const { groups, loading, error, openAdd } = useSpecs();

  return (
    <div className="hero">
      <h1>Your OpenAPI stash</h1>
      <p>
        Search endpoints, browse the full reference, and step through how each API changed between the versions you've
        stored.
      </p>
      <button className="btn btn-primary" type="button" onClick={() => openAdd()}>
        + Add a spec
      </button>

      {error && <p className="notice">Could not reach the openstash server: {error.message}</p>}

      {!loading && !error && groups.length === 0 && (
        <div className="empty" style={{ marginTop: 20 }}>
          <p>No specs stored yet. Add one here, or from the CLI:</p>
          <pre className="code">openstash add petstore --from https://petstore3.swagger.io/api/v3/openapi.json</pre>
        </div>
      )}

      <div className="cards">
        {groups.map((g) => (
          <div className="card" key={g.key}>
            <h3>
              <a href={href.explore(g.key, g.latest.version)}>{g.key}</a>
            </h3>
            <div className="meta-row">
              <span>
                latest <strong className="mono">{g.latest.version}</strong>
              </span>
              <span>{formatDate(g.latest.fetchedAt)}</span>
            </div>
            <div className="card-actions">
              <a className="btn btn-small" href={href.explore(g.key, g.latest.version)}>
                Explore
              </a>
              <a className="btn btn-small" href={href.reference(g.key, g.latest.version)}>
                Reference
              </a>
              {g.versions.length > 1 ? (
                <a className="btn btn-small" href={href.timeline(g.key)}>
                  Timeline · {g.versions.length} versions
                </a>
              ) : (
                <span className="muted small">1 version</span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
