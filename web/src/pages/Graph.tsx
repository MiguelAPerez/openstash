import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type PointerEvent } from "react";
import { Method } from "../components/Method";
import { SpecHeader } from "../components/SpecHeader";
import { api } from "../lib/api";
import {
  buildGraph,
  extractRefs,
  isChange,
  selectView,
  type ChangeStatus,
  type GraphEdge,
  type GraphNode,
  type SpecGraph,
  type ViewOptions,
} from "../lib/graph";
import { bounds, layoutGraph, type Point } from "../lib/layout";
import { href, navigate, type Route } from "../lib/route";
import { useSpecs } from "../lib/specs";
import { useAsync } from "../lib/useAsync";

type GraphRoute = Extract<Route, { name: "graph" }>;

const STATUS_SIGN: Record<ChangeStatus, string> = { added: "+", removed: "−", changed: "~" };
const ALL_STATUSES: ChangeStatus[] = ["added", "removed", "changed"];
/** Above this many nodes, labels only show for the selection and when zoomed in. */
const LABEL_ALL_MAX = 60;
/** Label a selected node's neighbors only when it has at most this many. */
const NEAR_LABEL_MAX = 16;
const MAX_LIST = 40;

export function Graph({ route }: { route: GraphRoute }) {
  const { key, query } = route;
  const { groups, loading } = useSpecs();
  const group = groups.find((g) => g.key === key);
  const versions = useMemo(() => group?.versions.map((v) => v.version) ?? [], [group]);
  const last = versions.length - 1;

  // Default pair: the newest step (previous → latest), or the only version.
  const from = query.get("from") ?? versions[Math.max(0, last - 1)] ?? "";
  const to = query.get("to") ?? versions[last] ?? "";
  const diffing = from !== to;
  const selected = query.get("node") ?? "";
  const focus = query.get("focus") ?? "";
  const depth = query.get("depth") === "2" ? 2 : 1;
  const mode: ViewOptions["mode"] = !diffing ? "all" : query.get("view") === "all" ? "all" : "changes";

  const go = (patch: Record<string, string | undefined>, replace = true) => {
    const next = {
      from: query.get("from") ?? undefined,
      to: query.get("to") ?? undefined,
      view: query.get("view") ?? undefined,
      focus: focus || undefined,
      depth: depth === 2 ? "2" : undefined,
      node: selected || undefined,
      ...patch,
    };
    navigate(href.graph(key, next), replace);
  };
  const select = useCallback(
    (id: string | null) => {
      const q = new URLSearchParams(query);
      if (id) q.set("node", id);
      else q.delete("node");
      navigate(href.graph(key, Object.fromEntries(q)), true);
    },
    [key, query],
  );
  // Stable (like select) so panning doesn't re-render every node.
  const focusOn = useCallback(
    (id: string) => {
      const q = new URLSearchParams(query);
      q.set("focus", id);
      q.set("node", id);
      navigate(href.graph(key, Object.fromEntries(q)), true);
    },
    [key, query],
  );

  const [statuses, setStatuses] = useState<Set<ChangeStatus>>(() => new Set(ALL_STATUSES));
  const [includeNoise, setIncludeNoise] = useState(false);
  const [neighbors, setNeighbors] = useState(true);
  const [showOperations, setShowOperations] = useState(true);
  const [showSchemas, setShowSchemas] = useState(true);

  const graph = useAsync(
    async (signal): Promise<SpecGraph | null> => {
      if (!from || !to) return null;
      const [a, b, cmp] = await Promise.all([
        api.dump(key, from, signal),
        diffing ? api.dump(key, to, signal) : null,
        diffing ? api.compare(key, from, to, signal) : null,
      ]);
      const refsA = extractRefs(a);
      return buildGraph(refsA, b ? extractRefs(b) : refsA, cmp);
    },
    [key, from, to],
  );
  const g = graph.status === "ok" ? graph.data : null;

  const opts: ViewOptions = useMemo(
    () => ({
      mode,
      focus: focus || undefined,
      depth,
      statuses,
      includeNoise,
      neighbors,
      showOperations,
      showSchemas,
    }),
    [mode, focus, depth, statuses, includeNoise, neighbors, showOperations, showSchemas],
  );
  const view = useMemo(() => (g ? selectView(g, opts) : null), [g, opts]);

  // Reuse the previous positions so filter changes don't reshuffle the picture.
  const previous = useRef<Map<string, Point> | undefined>(undefined);
  const positions = useMemo(() => {
    if (!view) return null;
    const p = layoutGraph(view.nodes, view.edges, previous.current);
    previous.current = p;
    return p;
  }, [view]);

  const counts = useMemo(() => {
    const c = { added: 0, removed: 0, changed: 0, noise: 0 };
    for (const n of g?.nodes ?? []) {
      if (n.status === "unchanged") continue;
      if (n.noise) c.noise++;
      else c[n.status]++;
    }
    return c;
  }, [g]);

  const byLabel = useMemo(() => new Map((g?.nodes ?? []).map((n) => [n.label, n.id])), [g]);
  const [search, setSearch] = useState("");
  const onSearch = (value: string) => {
    setSearch(value);
    const id = byLabel.get(value.trim());
    if (!id) return;
    const inView = view?.nodes.some((n) => n.id === id);
    go(inView ? { node: id } : { node: id, focus: id });
    setSearch("");
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (e.key !== "Escape" || t.tagName === "INPUT" || t.tagName === "SELECT") return;
      if (selected) select(null);
      else if (focus) go({ focus: undefined });
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  if (!group)
    return (
      <>
        <SpecHeader specKey={key} tab="graph" />
        {loading ? <div className="skeleton">Loading…</div> : <p className="notice">No spec stored under {key}.</p>}
      </>
    );

  const toggleStatus = (s: ChangeStatus) =>
    setStatuses((prev) => {
      const next = new Set(prev);
      if (next.has(s)) next.delete(s);
      else next.add(s);
      return next;
    });
  const selectedNode = g?.byId.get(selected);

  return (
    <>
      <SpecHeader specKey={key} tab="graph" />

      <div className="step-bar">
        {versions.length > 1 ? (
          <div className="range-picker">
            <label className="muted small" htmlFor="graph-from">
              Compare
            </label>
            <select
              id="graph-from"
              className="select"
              value={from}
              onChange={(e) => go({ from: e.target.value, to, node: undefined, focus: undefined })}
            >
              {versions.map((v) => (
                <option key={v}>{v}</option>
              ))}
            </select>
            <span className="arrow">→</span>
            <select
              className="select"
              aria-label="Compare to"
              value={to}
              onChange={(e) => go({ from, to: e.target.value, node: undefined, focus: undefined })}
            >
              {versions.map((v) => (
                <option key={v}>{v}</option>
              ))}
            </select>
          </div>
        ) : (
          <span className="muted small">
            Only one version stored: showing its dependencies. Add another version to see what changes touch.
          </span>
        )}
        <span className="spacer" />
        <input
          className="input graph-search"
          type="search"
          list="graph-nodes"
          placeholder="Find an endpoint or schema"
          aria-label="Find an endpoint or schema"
          value={search}
          onChange={(e) => onSearch(e.target.value)}
        />
        <datalist id="graph-nodes">
          {g?.nodes.map((n) => <option key={n.id} value={n.label} />)}
        </datalist>
      </div>

      <div className="graph-controls">
        {diffing && (
          <div className="segmented" role="group" aria-label="Show">
            <button type="button" aria-pressed={mode === "changes"} onClick={() => go({ view: undefined, focus: undefined })}>
              Changes
            </button>
            <button type="button" aria-pressed={mode === "all"} onClick={() => go({ view: "all", focus: undefined })}>
              Everything
            </button>
          </div>
        )}
        {diffing &&
          ALL_STATUSES.map((s) => (
            <button
              key={s}
              type="button"
              className={`btn btn-small status-toggle st-${s}`}
              aria-pressed={statuses.has(s)}
              onClick={() => toggleStatus(s)}
              title={`${statuses.has(s) ? "Hide" : "Show"} ${s} endpoints and schemas`}
            >
              {STATUS_SIGN[s]}
              {counts[s]} {s}
            </button>
          ))}
        {diffing && (
          <label className="check" title="Changes in x-* vendor extensions only, like generator metadata">
            <input type="checkbox" checked={includeNoise} onChange={(e) => setIncludeNoise(e.target.checked)} />
            x-* only ({counts.noise})
          </label>
        )}
        {diffing && mode === "changes" && (
          <label className="check" title="Also show what changed nodes use and what uses them">
            <input type="checkbox" checked={neighbors} onChange={(e) => setNeighbors(e.target.checked)} />
            neighbors
          </label>
        )}
        <label className="check">
          <input type="checkbox" checked={showOperations} onChange={(e) => setShowOperations(e.target.checked)} />
          endpoints
        </label>
        <label className="check">
          <input type="checkbox" checked={showSchemas} onChange={(e) => setShowSchemas(e.target.checked)} />
          schemas
        </label>
      </div>

      {focus && g?.byId.get(focus) && (
        <div className="graph-focus">
          <span>
            Around <strong className="mono">{g.byId.get(focus)!.label}</strong>
          </span>
          <div className="segmented" role="group" aria-label="Depth">
            {[1, 2].map((d) => (
              <button key={d} type="button" aria-pressed={depth === d} onClick={() => go({ depth: d === 2 ? "2" : undefined })}>
                {d} hop{d > 1 ? "s" : ""}
              </button>
            ))}
          </div>
          <button type="button" className="btn btn-small" onClick={() => go({ focus: undefined })}>
            Clear focus
          </button>
        </div>
      )}

      {graph.status === "error" && <p className="notice">{graph.error.message}</p>}
      {graph.status === "loading" && (
        <div className="skeleton">{diffing ? `Comparing ${from} → ${to}…` : `Loading ${to}…`}</div>
      )}

      {g && view && positions && (
        <div className="graph-layout">
          <section className="graph-stage" aria-label="Dependency graph">
            {view.nodes.length === 0 ? (
              <div className="empty graph-empty">
                {mode === "changes" && counts.added + counts.removed + counts.changed + counts.noise === 0
                  ? `No differences between ${from} and ${to}.`
                  : "Nothing matches these filters."}
              </div>
            ) : (
              <GraphCanvas
                nodes={view.nodes}
                edges={view.edges}
                positions={positions}
                primary={view.primary}
                graph={g}
                selected={selected}
                diffing={diffing}
                onSelect={select}
                onFocus={focusOn}
              />
            )}
            <div className="graph-footer muted small">
              <span>
                {view.nodes.length} nodes · {view.edges.length} links
                {view.nodes.length < g.nodes.length && ` (of ${g.nodes.length})`}
              </span>
              <Legend diffing={diffing} />
            </div>
          </section>
          <aside className="detail graph-detail" aria-label="Node details">
            {selectedNode ? (
              <NodeDetail
                specKey={key}
                node={selectedNode}
                graph={g}
                from={from}
                to={to}
                diffing={diffing}
                inView={view.nodes.some((n) => n.id === selectedNode.id)}
                focused={focus === selectedNode.id}
                opts={opts}
                onSelect={select}
                onFocus={focusOn}
              />
            ) : (
              <div className="detail-placeholder">
                <p>
                  Click a node to see what it uses and what uses it. Double-click to explore around it.
                  <br />
                  <span className="small">Drag to pan, scroll to zoom.</span>
                </p>
              </div>
            )}
          </aside>
        </div>
      )}
    </>
  );
}

function Legend({ diffing }: { diffing: boolean }) {
  return (
    <span className="graph-legend" aria-label="Legend">
      <span>
        <svg viewBox="-6 -6 12 12" aria-hidden="true">
          <rect className="legend-shape" x="-4.5" y="-4.5" width="9" height="9" rx="2" />
        </svg>
        endpoint
      </span>
      <span>
        <svg viewBox="-6 -6 12 12" aria-hidden="true">
          <circle className="legend-shape" r="5" />
        </svg>
        schema
      </span>
      {diffing &&
        (["added", "removed", "changed", "unchanged"] as const).map((s) => (
          <span key={s}>
            <svg viewBox="-6 -6 12 12" aria-hidden="true">
              <circle className={`node-${s}`} r="5" />
            </svg>
            {s}
          </span>
        ))}
    </span>
  );
}

interface Transform {
  k: number;
  x: number;
  y: number;
}

function GraphCanvas({
  nodes,
  edges,
  positions,
  primary,
  graph,
  selected,
  diffing,
  onSelect,
  onFocus,
}: {
  nodes: GraphNode[];
  edges: GraphEdge[];
  positions: Map<string, Point>;
  primary: Set<string>;
  graph: SpecGraph;
  selected: string;
  diffing: boolean;
  onSelect: (id: string | null) => void;
  onFocus: (id: string) => void;
}) {
  const svg = useRef<SVGSVGElement>(null);
  const [t, setT] = useState<Transform>({ k: 1, x: 0, y: 0 });
  const [size, setSize] = useState({ w: 0, h: 0 });
  // Set once the user pans or zooms, so a resize stops refitting over their view.
  const touched = useRef(false);

  useLayoutEffect(() => {
    const el = svg.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) =>
      setSize({ w: entry.contentRect.width, h: entry.contentRect.height }),
    );
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const fit = useCallback(() => {
    if (size.w === 0 || size.h === 0) return;
    const b = bounds(nodes.map((n) => positions.get(n.id)!).filter(Boolean));
    const k = Math.min(size.w / b.w, size.h / b.h, 2.5);
    setT({ k, x: (size.w - b.w * k) / 2 - b.x * k, y: (size.h - b.h * k) / 2 - b.y * k });
  }, [nodes, positions, size]);

  // Fit whenever a new layout arrives.
  useLayoutEffect(() => {
    touched.current = false;
  }, [positions]);
  useLayoutEffect(() => {
    if (!touched.current) fit();
  }, [fit]);

  const zoomBy = useCallback((factor: number, cx?: number, cy?: number) => {
    const el = svg.current;
    if (!el) return;
    touched.current = true;
    const r = el.getBoundingClientRect();
    const px = cx ?? r.width / 2;
    const py = cy ?? r.height / 2;
    setT((cur) => {
      const k = Math.min(8, Math.max(0.05, cur.k * factor));
      const wx = (px - cur.x) / cur.k;
      const wy = (py - cur.y) / cur.k;
      return { k, x: px - wx * k, y: py - wy * k };
    });
  }, []);

  // Wheel zoom needs a non-passive listener to keep the page from scrolling.
  useEffect(() => {
    const el = svg.current;
    if (!el) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const r = el.getBoundingClientRect();
      zoomBy(Math.exp(-e.deltaY * (e.deltaMode === 1 ? 0.05 : 0.0015)), e.clientX - r.left, e.clientY - r.top);
    };
    el.addEventListener("wheel", onWheel, { passive: false });
    return () => el.removeEventListener("wheel", onWheel);
  }, [zoomBy]);

  // Drag on the background pans; a click without movement clears the selection.
  const drag = useRef<{ x: number; y: number; tx: number; ty: number; moved: boolean } | null>(null);
  const onPointerDown = (e: PointerEvent<SVGSVGElement>) => {
    if ((e.target as Element).closest("[data-node]") || e.button !== 0) return;
    drag.current = { x: e.clientX, y: e.clientY, tx: t.x, ty: t.y, moved: false };
    e.currentTarget.setPointerCapture(e.pointerId);
  };
  const onPointerMove = (e: PointerEvent<SVGSVGElement>) => {
    const d = drag.current;
    if (!d) return;
    const dx = e.clientX - d.x;
    const dy = e.clientY - d.y;
    if (!d.moved && Math.hypot(dx, dy) < 3) return;
    d.moved = true;
    touched.current = true;
    setT((cur) => ({ ...cur, x: d.tx + dx, y: d.ty + dy }));
  };
  const onPointerUp = () => {
    const d = drag.current;
    drag.current = null;
    if (d && !d.moved) onSelect(null);
  };

  // Keep the selected node on screen when it's picked from the side panel or
  // search, or when a new layout moves it. Panning never re-runs this, so the
  // user can still move the selection out of view.
  useEffect(() => {
    const el = svg.current;
    const p = positions.get(selected);
    if (!el || !p) return;
    const { width, height } = el.getBoundingClientRect();
    setT((cur) => {
      const sx = p.x * cur.k + cur.x;
      const sy = p.y * cur.k + cur.y;
      if (sx > 40 && sx < width - 40 && sy > 40 && sy < height - 40) return cur;
      return { ...cur, x: width / 2 - p.x * cur.k, y: height / 2 - p.y * cur.k };
    });
  }, [selected, positions]);

  // Quantized so panning never re-renders the layer and zooming rarely does.
  const scale = Math.round(Math.min(2, 1 / Math.sqrt(Math.max(t.k, 0.05))) * 10) / 10;
  const showAllLabels = nodes.length <= LABEL_ALL_MAX || t.k >= 1.6;

  return (
    <div className="graph-canvas">
      <svg
        ref={svg}
        className="graph-svg"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={() => (drag.current = null)}
      >
        <g transform={`translate(${t.x},${t.y}) scale(${t.k})`}>
          <GraphLayer
            nodes={nodes}
            edges={edges}
            positions={positions}
            primary={primary}
            graph={graph}
            selected={selected}
            diffing={diffing}
            scale={scale}
            labelSize={Math.round((11 / t.k) * 10) / 10}
            showAllLabels={showAllLabels}
            onSelect={onSelect}
            onFocus={onFocus}
          />
        </g>
      </svg>
      <div className="graph-zoom">
        <button type="button" className="btn btn-small" aria-label="Zoom in" onClick={() => zoomBy(1.4)}>
          +
        </button>
        <button type="button" className="btn btn-small" aria-label="Zoom out" onClick={() => zoomBy(1 / 1.4)}>
          −
        </button>
        <button
          type="button"
          className="btn btn-small"
          onClick={() => {
            touched.current = false;
            fit();
          }}
        >
          Fit
        </button>
      </div>
    </div>
  );
}

const GraphLayer = memo(function GraphLayer({
  nodes,
  edges,
  positions,
  primary,
  graph,
  selected,
  diffing,
  scale,
  labelSize,
  showAllLabels,
  onSelect,
  onFocus,
}: {
  nodes: GraphNode[];
  edges: GraphEdge[];
  positions: Map<string, Point>;
  primary: Set<string>;
  graph: SpecGraph;
  selected: string;
  diffing: boolean;
  scale: number;
  labelSize: number;
  showAllLabels: boolean;
  onSelect: (id: string) => void;
  onFocus: (id: string) => void;
}) {
  const near = useMemo(() => {
    if (!selected) return null;
    return new Set([selected, ...(graph.out.get(selected) ?? []), ...(graph.in.get(selected) ?? [])]);
  }, [graph, selected]);

  const nodeClass = (n: GraphNode) =>
    [
      "node",
      `node-${diffing ? n.status : "unchanged"}`,
      n.noise ? "noise" : "",
      primary.has(n.id) ? "" : "context",
      n.id === selected ? "selected" : "",
      near && !near.has(n.id) ? "dim" : "",
    ]
      .filter(Boolean)
      .join(" ");

  return (
    <>
      <g className="graph-edges">
        {edges.map((e) => {
          const a = positions.get(e.source)!;
          const b = positions.get(e.target)!;
          const hl = selected && (e.source === selected || e.target === selected);
          return (
            <line
              key={`${e.source}>${e.target}`}
              className={`edge edge-${diffing ? e.status : "same"} ${hl ? "hl" : near ? "dim" : ""}`}
              x1={a.x}
              y1={a.y}
              x2={b.x}
              y2={b.y}
              vectorEffect="non-scaling-stroke"
            />
          );
        })}
      </g>
      <g className="graph-nodes">
        {nodes.map((n) => {
          const p = positions.get(n.id)!;
          return (
            <g
              key={n.id}
              data-node={n.id}
              className={nodeClass(n)}
              transform={`translate(${p.x},${p.y}) scale(${scale})`}
              onClick={() => onSelect(n.id)}
              onDoubleClick={() => onFocus(n.id)}
            >
              {n.kind === "operation" ? <rect x="-5" y="-5" width="10" height="10" rx="2" /> : <circle r="5.5" />}
              <title>
                {n.label}
                {diffing && n.status !== "unchanged" ? ` (${n.noise ? "x-* only" : n.status})` : ""}
              </title>
            </g>
          );
        })}
      </g>
      <g className="graph-labels" aria-hidden="true">
        {nodes.map((n) => {
          // A busy node's neighbors would pile their labels on top of each other.
          const labelled =
            showAllLabels || n.id === selected || (near !== null && near.size <= NEAR_LABEL_MAX && near.has(n.id));
          if (!labelled) return null;
          const p = positions.get(n.id)!;
          return (
            <text
              key={n.id}
              className={n.id === selected ? "selected" : near && !near.has(n.id) ? "dim" : ""}
              x={p.x + 8 * scale}
              y={p.y}
              fontSize={labelSize}
              dominantBaseline="middle"
            >
              {n.label}
            </text>
          );
        })}
      </g>
    </>
  );
});

function NodeDetail({
  specKey,
  node,
  graph,
  from,
  to,
  diffing,
  inView,
  focused,
  opts,
  onSelect,
  onFocus,
}: {
  specKey: string;
  node: GraphNode;
  graph: SpecGraph;
  from: string;
  to: string;
  diffing: boolean;
  inView: boolean;
  focused: boolean;
  opts: ViewOptions;
  onSelect: (id: string) => void;
  onFocus: (id: string) => void;
}) {
  const version = node.status === "removed" ? from : to;
  const explore =
    node.kind === "operation"
      ? href.explore(specKey, version, { op: `${node.method} ${node.path}` })
      : href.explore(specKey, version, { schema: node.schema, in: "schemas" });
  const uses = [...(graph.out.get(node.id) ?? [])].map((id) => graph.byId.get(id)!);
  const usedBy = [...(graph.in.get(node.id) ?? [])].map((id) => graph.byId.get(id)!);
  const changedUsers = usedBy.filter((n) => isChange(n, opts)).length;

  return (
    <div className="graph-node-detail">
      <h2 className="detail-title">
        {node.kind === "operation" ? (
          <>
            <Method method={node.method!} />
            <span className="result-path mono">{node.path}</span>
          </>
        ) : (
          <span className="result-path mono">{node.schema}</span>
        )}
      </h2>
      {node.summary && <p className="muted">{node.summary}</p>}
      {diffing && (
        <p>
          <span className={`status-pill st-${node.status}`}>{node.noise ? "x-* only" : node.status}</span>
          {node.status === "changed" && node.notes.length > 0 && (
            <span className="muted small"> {node.notes.join(", ")}</span>
          )}
        </p>
      )}
      <div className="graph-actions">
        {!focused && (
          <button type="button" className="btn btn-small" onClick={() => onFocus(node.id)}>
            Focus here
          </button>
        )}
        {diffing && node.status !== "unchanged" && (
          <a className="btn btn-small" href={href.timeline(specKey, { from, to, item: node.id })}>
            Diff in timeline
          </a>
        )}
        <a className="btn btn-small" href={explore}>
          Open in {version}
        </a>
      </div>
      {!inView && <p className="muted small">Hidden by the current filters. Focus here to see its neighborhood.</p>}
      <NodeList title="Uses" nodes={uses} diffing={diffing} onSelect={onSelect} />
      <NodeList
        title="Used by"
        hint={diffing && changedUsers > 0 ? `${changedUsers} changed` : undefined}
        nodes={usedBy}
        diffing={diffing}
        onSelect={onSelect}
      />
    </div>
  );
}

const statusRank = { removed: 0, added: 1, changed: 2, unchanged: 3 };

function NodeList({
  title,
  hint,
  nodes,
  diffing,
  onSelect,
}: {
  title: string;
  hint?: string;
  nodes: GraphNode[];
  diffing: boolean;
  onSelect: (id: string) => void;
}) {
  const [all, setAll] = useState(false);
  const sorted = useMemo(
    () =>
      [...nodes].sort(
        (a, b) =>
          (diffing ? statusRank[a.status] - statusRank[b.status] : 0) ||
          Number(a.kind === "schema") - Number(b.kind === "schema") ||
          a.label.localeCompare(b.label),
      ),
    [nodes, diffing],
  );
  if (nodes.length === 0) return null;
  const shown = all ? sorted : sorted.slice(0, MAX_LIST);
  return (
    <section className="graph-list">
      <h3>
        {title} <span className="muted">({nodes.length})</span>
        {hint && <span className="muted small"> · {hint}</span>}
      </h3>
      <ul>
        {shown.map((n) => (
          <li key={n.id}>
            <button type="button" onClick={() => onSelect(n.id)} title={n.label}>
              <span className={`dot node-${diffing ? n.status : "unchanged"} ${n.kind}`} aria-hidden="true" />
              {n.kind === "operation" ? (
                <>
                  <Method method={n.method!} />
                  <span className="mono truncate">{n.path}</span>
                </>
              ) : (
                <span className="mono truncate">{n.label}</span>
              )}
            </button>
          </li>
        ))}
      </ul>
      {sorted.length > shown.length && (
        <button type="button" className="btn btn-small" onClick={() => setAll(true)}>
          Show all {sorted.length}
        </button>
      )}
    </section>
  );
}
