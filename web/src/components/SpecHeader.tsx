import { useState } from "react";
import { api } from "../lib/api";
import { href, navigate } from "../lib/route";
import { formatDateTime, useSpecs } from "../lib/specs";

type Tab = "explore" | "reference" | "timeline";

/** Re-fetches the latest version's source and stores it if its info.version is new. */
function PullButton({ specKey, source }: { specKey: string; source: string }) {
  const { reload, notify } = useSpecs();
  const [busy, setBusy] = useState(false);

  const pull = async () => {
    setBusy(true);
    try {
      const res = await api.pullSpec(specKey);
      if (res.status === "added") {
        notify(
          `Pulled ${res.meta.key}@${res.meta.version} (was ${res.previous}): ${res.indexed} operations, ${res.schemas} schemas`,
        );
        reload();
        navigate(href.timeline(specKey, { to: res.meta.version }));
      } else {
        notify(`${specKey} is up to date: the source still serves ${res.remoteVersion}`);
      }
    } catch (err) {
      notify(`Pull failed: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <button
      className="btn btn-small"
      type="button"
      onClick={pull}
      disabled={busy}
      title={`Fetch ${source} and add it as a new version if its info.version changed`}
    >
      {busy ? "Pulling…" : "Pull latest"}
    </button>
  );
}

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
        {group?.latest.source && <PullButton specKey={specKey} source={group.latest.source} />}
        {v && (
          <a className="btn btn-small" href={`/v1/specs/${encodeURIComponent(specKey)}/versions/${encodeURIComponent(v)}`} download={`${specKey}-${v}.json`}>
            Download JSON
          </a>
        )}
      </nav>
    </header>
  );
}
