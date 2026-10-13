import { href } from "../lib/route";
import { formatDateTime, useSpecs } from "../lib/specs";

type Tab = "explore" | "reference" | "timeline";

export function SpecHeader({ specKey, version, tab }: { specKey: string; version?: string; tab: Tab }) {
  const { groups } = useSpecs();
  const group = groups.find((g) => g.key === specKey);
  const v = version ?? group?.latest.version;
  const entry = group?.versions.find((e) => e.version === v);

  return (
    <header className="spec-header">
      <div className="spec-title-row">
        <h1>{specKey}</h1>
        {group && v && tab !== "timeline" && (
          <select
            className="select"
            aria-label="Version"
            value={v}
            onChange={(e) =>
              (window.location.hash =
                tab === "reference" ? href.reference(specKey, e.target.value) : href.explore(specKey, e.target.value))
            }
          >
            {[...group.versions].reverse().map((e) => (
              <option key={e.version} value={e.version}>
                {e.version}
                {e === group.latest ? " (latest)" : ""}
              </option>
            ))}
          </select>
        )}
      </div>
      {entry && tab !== "timeline" && (
        <div className="meta-row">
          <span className="truncate" title={entry.source}>
            source <strong>{entry.source}</strong>
          </span>
          {entry.endpoint && (
            <span>
              endpoint <strong>{entry.endpoint}</strong>
            </span>
          )}
          <span>fetched {formatDateTime(entry.fetchedAt)}</span>
        </div>
      )}
      <nav className="tabs" aria-label="Spec views">
        {v && (
          <a className="tab" href={href.explore(specKey, v)} aria-current={tab === "explore" ? "page" : undefined}>
            Explore
          </a>
        )}
        {v && (
          <a className="tab" href={href.reference(specKey, v)} aria-current={tab === "reference" ? "page" : undefined}>
            Reference
          </a>
        )}
        <a className="tab" href={href.timeline(specKey)} aria-current={tab === "timeline" ? "page" : undefined}>
          Timeline
          {group && group.versions.length > 1 && <span className="tab-count">{group.versions.length}</span>}
        </a>
        <span className="spacer" />
        {v && (
          <a className="btn btn-small" href={`/v1/specs/${encodeURIComponent(specKey)}/versions/${encodeURIComponent(v)}`} download={`${specKey}-${v}.json`}>
            Download JSON
          </a>
        )}
      </nav>
    </header>
  );
}
