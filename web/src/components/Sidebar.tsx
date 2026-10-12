import { useState } from "react";
import { href, type Route } from "../lib/route";
import { useSpecs } from "../lib/specs";

export function Sidebar({ route, open, onNavigate }: { route: Route; open: boolean; onNavigate: () => void }) {
  const { groups, loading, error, openAdd } = useSpecs();
  const [filter, setFilter] = useState("");
  const f = filter.trim().toLowerCase();
  const shown = f ? groups.filter((g) => g.key.toLowerCase().includes(f)) : groups;
  const currentKey = route.name === "home" || route.name === "notFound" ? null : route.key;
  const currentVersion = route.name === "explore" || route.name === "reference" ? route.version : null;

  return (
    <aside className={`sidebar ${open ? "open" : ""}`} id="sidebar">
      <div className="sidebar-head">
        <h2>Specs</h2>
        <button className="btn btn-small" type="button" onClick={() => openAdd()}>
          + Add
        </button>
      </div>
      {groups.length > 5 && (
        <input
          className="input sidebar-filter"
          type="search"
          placeholder="Filter specs"
          aria-label="Filter specs"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      )}
      {loading && groups.length === 0 && <div className="sidebar-empty">Loading…</div>}
      {error && <div className="sidebar-empty">Could not load specs: {error.message}</div>}
      {!loading && !error && groups.length === 0 && <div className="sidebar-empty">No specs stored yet.</div>}
      <ul className="spec-list" onClick={(e) => (e.target as HTMLElement).closest("a") && onNavigate()}>
        {shown.map((g) => (
          <li className={`spec-item ${currentKey === g.key ? "current" : ""}`} key={g.key}>
            <a
              className="spec-link"
              href={href.explore(g.key, g.latest.version)}
              aria-current={currentKey === g.key ? "page" : undefined}
            >
              <span className="spec-key">{g.key}</span>
            </a>
            <div className="spec-versions">
              {[...g.versions].reverse().map((v) => (
                <a
                  key={v.version}
                  className="chip"
                  href={href.explore(g.key, v.version)}
                  title={v.version}
                  aria-current={currentKey === g.key && currentVersion === v.version ? "page" : undefined}
                >
                  {v.version}
                </a>
              ))}
              {g.versions.length > 1 && (
                <a
                  className="chip chip-timeline"
                  href={href.timeline(g.key)}
                  aria-current={currentKey === g.key && route.name === "timeline" ? "page" : undefined}
                >
                  timeline
                </a>
              )}
            </div>
          </li>
        ))}
      </ul>
    </aside>
  );
}
