import { useEffect, useMemo, useRef, useState } from "react";
import { DiffStats, DiffView, type DiffMode } from "../components/DiffView";
import { Method } from "../components/Method";
import { SpecHeader } from "../components/SpecHeader";
import { api, type CompareCounts, type Timeline as TimelineData, type TimelineStep } from "../lib/api";
import { changeItems, type ChangeItem, type ChangeKind } from "../lib/changes";
import { computeDiff, stableStringify } from "../lib/diff";
import { href, navigate, type Route } from "../lib/route";
import { formatDate, useSpecs } from "../lib/specs";
import { useAsync } from "../lib/useAsync";

type TimelineRoute = Extract<Route, { name: "timeline" }>;

const kindSign: Record<ChangeKind, string> = { added: "+", removed: "−", changed: "~" };

function sum(a: CompareCounts, b: CompareCounts) {
  return { added: a.added + b.added, removed: a.removed + b.removed, changed: a.changed + b.changed };
}

export function Timeline({ route }: { route: TimelineRoute }) {
  const { key, query } = route;
  const { openAdd } = useSpecs();
  const timeline = useAsync((signal) => api.timeline(key, signal), [key]);
  const data = timeline.data;

  const steps = data?.steps ?? [];
  // Selected pair: explicit from/to, else the newest step. A lone `to` (or
  // `from`) picks the step ending (or starting) at that version, which is how
  // the add dialog links to a version it just stored.
  const newest = steps[steps.length - 1];
  const qFrom = query.get("from");
  const qTo = query.get("to");
  const around =
    qFrom === null && qTo !== null
      ? (steps.find((s) => s.to === qTo) ?? steps.find((s) => s.from === qTo))
      : qTo === null && qFrom !== null
        ? (steps.find((s) => s.from === qFrom) ?? steps.find((s) => s.to === qFrom))
        : undefined;
  const from = (around ? around.from : qFrom) ?? newest?.from ?? "";
  const to = (around ? around.to : qTo) ?? newest?.to ?? "";
  const itemId = query.get("item") ?? "";
  const kindFilter = (query.get("show") ?? "all") as "all" | "operations" | "schemas";
  const stepIndex = steps.findIndex((s) => s.from === from && s.to === to);

  const go = (patch: Record<string, string | undefined>, replace = false) => {
    const next = { from, to, item: itemId || undefined, show: kindFilter === "all" ? undefined : kindFilter, ...patch };
    navigate(href.timeline(key, next), replace);
  };
  const selectStep = (s: TimelineStep) => go({ from: s.from, to: s.to, item: undefined });

  const compare = useAsync(
    (signal) => (from && to ? api.compare(key, from, to, signal) : Promise.resolve(null)),
    [key, from, to],
  );
  const allItems = useMemo(() => (compare.data ? changeItems(compare.data) : []), [compare.data]);
  const items = useMemo(
    () =>
      allItems.filter((i) =>
        kindFilter === "operations" ? i.type === "operation" : kindFilter === "schemas" ? i.type === "schema" : true,
      ),
    [allItems, kindFilter],
  );
  const itemIndex = items.findIndex((i) => i.id === itemId);
  const current = itemIndex >= 0 ? items[itemIndex] : undefined;

  // Auto-select the first change when the step loads (or the selection vanished).
  useEffect(() => {
    if (compare.status === "ok" && items.length > 0 && itemIndex < 0) go({ item: items[0].id }, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [compare.status, items, itemIndex]);

  const moveItem = (delta: number) => {
    if (items.length === 0) return;
    const i = Math.min(items.length - 1, Math.max(0, (itemIndex < 0 ? 0 : itemIndex) + delta));
    go({ item: items[i].id }, true);
  };
  // Older/Newer step from the current one. From a custom range they snap to
  // the single steps around the range's newer end.
  const stepTarget = (delta: number): TimelineStep | undefined => {
    if (stepIndex >= 0) return steps[stepIndex + delta];
    const end = steps.findIndex((s) => s.to === to);
    if (end >= 0) return steps[delta < 0 ? end : end + 1];
    return delta > 0 ? steps.find((s) => s.from === to) : undefined;
  };
  const moveStep = (delta: number) => {
    const s = stepTarget(delta);
    if (s) selectStep(s);
  };

  // Keyboard: j/k through changes, [ / ] through steps. Arrow keys are left to
  // scroll the page.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement;
      if (t.tagName === "INPUT" || t.tagName === "SELECT" || t.tagName === "TEXTAREA" || e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "j") moveItem(1);
      else if (e.key === "k") moveItem(-1);
      else if (e.key === "]") moveStep(1);
      else if (e.key === "[") moveStep(-1);
      else return;
      e.preventDefault();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  if (timeline.status === "error")
    return (
      <>
        <SpecHeader specKey={key} tab="timeline" />
        <p className="notice">{timeline.error.message}</p>
      </>
    );
  if (!data)
    return (
      <>
        <SpecHeader specKey={key} tab="timeline" />
        <div className="skeleton">Building timeline…</div>
      </>
    );

  return (
    <>
      <SpecHeader specKey={key} tab="timeline" />

      {data.versions.length < 2 ? (
        <div className="empty">
          <p>
            Only one version of <strong>{key}</strong> is stored, so there is nothing to compare yet.
          </p>
          <p>Add a newer (or older) version and its changes will show up here.</p>
          <button className="btn btn-primary" type="button" onClick={() => openAdd(key)}>
            + Add another version
          </button>
          <pre className="code">{`openstash add ${key} --from <url-or-file> --version <new-version>`}</pre>
        </div>
      ) : (
        <>
          <Rail data={data} from={from} to={to} onStep={selectStep} />

          <div className="step-bar">
            <button type="button" className="btn btn-small" onClick={() => moveStep(-1)} disabled={!stepTarget(-1)}>
              ← Older
            </button>
            <div className="step-title">
              <span className="mono">{from}</span>
              <span className="arrow">→</span>
              <span className="mono">{to}</span>
              {stepIndex >= 0 ? (
                <span className="muted small">
                  step {stepIndex + 1} of {steps.length}
                </span>
              ) : (
                <span className="muted small">custom range</span>
              )}
            </div>
            <button
              type="button"
              className="btn btn-small"
              onClick={() => moveStep(1)}
              disabled={!stepTarget(1)}
            >
              Newer →
            </button>
            <span className="spacer" />
            <RangePicker data={data} from={from} to={to} onChange={(f, t) => go({ from: f, to: t, item: undefined })} />
          </div>

          {compare.status === "error" && <p className="notice">{compare.error.message}</p>}
          {compare.data && (
            <SummaryTiles
              ops={compare.data.summary.operations}
              schemas={compare.data.summary.schemas}
              filter={kindFilter}
              onFilter={(f) => go({ show: f === "all" ? undefined : f, item: undefined }, true)}
            />
          )}

          <div className="split timeline-split">
            <section aria-label="Changes">
              {compare.status === "loading" && <div className="skeleton">Comparing {from} → {to}…</div>}
              {compare.status === "ok" && items.length === 0 && (
                <div className="empty">No differences between these versions.</div>
              )}
              <ChangeList items={items} currentId={current?.id} onSelect={(id) => go({ item: id }, true)} />
            </section>
            <section className="detail diff-panel" aria-label="Diff">
              {current ? (
                <ChangeDiff
                  key={`${from}|${to}|${current.id}`}
                  specKey={key}
                  from={from}
                  to={to}
                  item={current}
                  position={itemIndex + 1}
                  total={items.length}
                  onPrev={() => moveItem(-1)}
                  onNext={() => moveItem(1)}
                />
              ) : (
                compare.status === "ok" &&
                items.length > 0 && <div className="detail-placeholder">Select a change to see its diff.</div>
              )}
            </section>
          </div>
        </>
      )}
    </>
  );
}

function Rail({
  data,
  from,
  to,
  onStep,
}: {
  data: TimelineData;
  from: string;
  to: string;
  onStep: (s: TimelineStep) => void;
}) {
  const ref = useRef<HTMLOListElement>(null);
  const fromIdx = data.versions.findIndex((v) => v.version === from);
  const toIdx = data.versions.findIndex((v) => v.version === to);
  const [lo, hi] = [Math.min(fromIdx, toIdx), Math.max(fromIdx, toIdx)];

  // Keep the selected step in view on narrow screens.
  useEffect(() => {
    ref.current?.querySelector('[aria-current="step"]')?.scrollIntoView({ block: "nearest", inline: "center" });
  }, [from, to]);

  return (
    <ol className="rail" ref={ref} aria-label="Version timeline">
      {data.versions.map((v, i) => {
        const step = i > 0 ? data.steps[i - 1] : undefined;
        const inRange = fromIdx >= 0 && toIdx >= 0 && i > lo && i <= hi;
        const selected = step && step.from === from && step.to === to;
        const c = step && sum(step.summary.operations, step.summary.schemas);
        return (
          <li key={v.version} className="rail-item">
            {step && c && (
              <button
                type="button"
                className={`rail-step ${inRange ? "in-range" : ""}`}
                aria-current={selected ? "step" : undefined}
                onClick={() => onStep(step)}
                title={`Compare ${step.from} → ${step.to}`}
              >
                <span className="rail-line" aria-hidden="true" />
                <span className="rail-counts">
                  {c.added + c.removed + c.changed === 0 ? (
                    <span className="muted">no changes</span>
                  ) : (
                    <>
                      {c.added > 0 && <span className="plus">+{c.added}</span>}
                      {c.removed > 0 && <span className="minus">−{c.removed}</span>}
                      {c.changed > 0 && <span className="tilde">~{c.changed}</span>}
                    </>
                  )}
                </span>
              </button>
            )}
            <div className={`rail-node ${i === fromIdx || i === toIdx ? "endpoint" : ""}`}>
              <span className="rail-dot" aria-hidden="true" />
              <span className="rail-version mono">{v.version}</span>
              <span className="rail-date muted">{formatDate(v.fetchedAt)}</span>
              <span className="rail-stats muted">
                {v.operations} ops · {v.schemas} schemas
              </span>
              <a className="rail-open" href={href.explore(data.key, v.version)}>
                explore
              </a>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function RangePicker({
  data,
  from,
  to,
  onChange,
}: {
  data: TimelineData;
  from: string;
  to: string;
  onChange: (from: string, to: string) => void;
}) {
  const versions = data.versions.map((v) => v.version);
  return (
    <div className="range-picker">
      <label className="muted small" htmlFor="cmp-from">
        Compare
      </label>
      <select id="cmp-from" className="select" value={from} onChange={(e) => onChange(e.target.value, to)}>
        {versions.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </select>
      <span className="muted">→</span>
      <select aria-label="Compare to" className="select" value={to} onChange={(e) => onChange(from, e.target.value)}>
        {versions.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </select>
    </div>
  );
}

function SummaryTiles({
  ops,
  schemas,
  filter,
  onFilter,
}: {
  ops: CompareCounts;
  schemas: CompareCounts;
  filter: "all" | "operations" | "schemas";
  onFilter: (f: "all" | "operations" | "schemas") => void;
}) {
  const tile = (label: string, c: CompareCounts, value: "operations" | "schemas") => (
    <button
      type="button"
      className="tile"
      aria-pressed={filter === value}
      onClick={() => onFilter(filter === value ? "all" : value)}
    >
      <span className="tile-label">{label}</span>
      <span className="tile-counts">
        <span className="plus">+{c.added}</span>
        <span className="minus">−{c.removed}</span>
        <span className="tilde">~{c.changed}</span>
      </span>
      <span className="muted small">{c.unchanged} unchanged</span>
    </button>
  );
  return (
    <div className="tiles">
      {tile("Endpoints", ops, "operations")}
      {tile("Schemas", schemas, "schemas")}
    </div>
  );
}

function ChangeList({
  items,
  currentId,
  onSelect,
}: {
  items: ChangeItem[];
  currentId?: string;
  onSelect: (id: string) => void;
}) {
  const ref = useRef<HTMLUListElement>(null);
  useEffect(() => {
    ref.current?.querySelector('[aria-current="true"]')?.scrollIntoView({ block: "nearest" });
  }, [currentId]);

  const counts = new Map<string, number>();
  for (const it of items) counts.set(changeGroup(it), (counts.get(changeGroup(it)) ?? 0) + 1);
  let lastGroup = "";
  return (
    <ul className="results changes" ref={ref}>
      {items.map((it) => {
        const group = changeGroup(it);
        const header = group !== lastGroup;
        lastGroup = group;
        return (
          <li key={it.id}>
            {header && (
              <div className={`change-group kind-${it.kind}`}>
                {group} <span className="muted">({counts.get(group)})</span>
              </div>
            )}
            <button
              type="button"
              className={`result change kind-${it.kind}`}
              aria-current={it.id === currentId}
              onClick={() => onSelect(it.id)}
            >
              <div className="result-line">
                <span className={`kind-badge kind-${it.kind}`} aria-label={it.kind}>
                  {kindSign[it.kind]}
                </span>
                {it.method && <Method method={it.method} />}
                <span className="result-path">{it.label}</span>
              </div>
              {it.summary && <div className="result-summary">{it.summary}</div>}
              {it.notes.length > 0 && (
                <div className="result-meta">
                  {it.notes.slice(0, 8).map((n) => (
                    <span key={n} className="tag mono">
                      {n}
                    </span>
                  ))}
                  {it.notes.length > 8 && <span className="tag">+{it.notes.length - 8}</span>}
                </div>
              )}
            </button>
          </li>
        );
      })}
    </ul>
  );
}

function changeGroup(it: ChangeItem): string {
  return `${it.type === "operation" ? "Endpoints" : "Schemas"} ${it.kind}${it.extensionsOnly ? " · x-* only" : ""}`;
}

function ChangeDiff({
  specKey,
  from,
  to,
  item,
  position,
  total,
  onPrev,
  onNext,
}: {
  specKey: string;
  from: string;
  to: string;
  item: ChangeItem;
  position: number;
  total: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  const [mode, setMode] = useState<DiffMode>(() => {
    try {
      return localStorage.getItem("openstash.diffMode") === "split" ? "split" : "unified";
    } catch {
      return "unified";
    }
  });
  const setModeSaved = (m: DiffMode) => {
    setMode(m);
    try {
      localStorage.setItem("openstash.diffMode", m);
    } catch {
      // storage unavailable; preference just won't persist
    }
  };

  const def = useAsync(
    (signal) =>
      api.definition(
        specKey,
        from,
        to,
        item.type === "schema" ? { schema: item.schema! } : { method: item.method!, path: item.path! },
        signal,
      ),
    [specKey, from, to, item.id],
  );
  // Stringify and diff once per definition, shared by the stats and the view.
  const diff = useMemo(
    () => (def.data ? computeDiff(stableStringify(def.data.baseline), stableStringify(def.data.target)) : null),
    [def.data],
  );
  const exploreVersion = item.kind === "removed" ? from : to;
  const exploreLink =
    item.type === "operation"
      ? href.explore(specKey, exploreVersion, { op: `${item.method} ${item.path}` })
      : href.explore(specKey, exploreVersion, { in: "schemas", schema: item.schema, q: item.schema });

  return (
    <article>
      <div className="diff-head">
        <div className="detail-title">
          <span className={`kind-badge kind-${item.kind}`}>{kindSign[item.kind]}</span>
          {item.method && <Method method={item.method} />}
          <span className="result-path">{item.label}</span>
        </div>
        <div className="diff-nav">
          <button type="button" className="btn btn-small" onClick={onPrev} disabled={position <= 1} title="Previous change (k)">
            ↑ Prev
          </button>
          <span className="muted small">
            {position} / {total}
          </span>
          <button type="button" className="btn btn-small" onClick={onNext} disabled={position >= total} title="Next change (j)">
            Next ↓
          </button>
        </div>
      </div>
      <div className="detail-toolbar">
        <span className="muted small">
          {item.kind === "added" && <>Added in {to}</>}
          {item.kind === "removed" && <>Removed in {to}</>}
          {item.kind === "changed" && (
            <>
              Changed: {item.notes.join(", ")}
              {item.extensionsOnly && " (vendor extensions only)"}
            </>
          )}
        </span>
        {diff && <DiffStats diff={diff} />}
        <span className="spacer" />
        <div className="segmented" role="group" aria-label="Diff layout">
          <button type="button" aria-pressed={mode === "unified"} onClick={() => setModeSaved("unified")}>
            Unified
          </button>
          <button type="button" aria-pressed={mode === "split"} onClick={() => setModeSaved("split")}>
            Split
          </button>
        </div>
        <a className="btn btn-small" href={exploreLink}>
          Open in {exploreVersion}
        </a>
      </div>
      {mode === "split" && (
        <div className="split-labels mono small">
          <span>{from}</span>
          <span>{to}</span>
        </div>
      )}
      {def.status === "loading" && <div className="skeleton">Loading definitions…</div>}
      {def.status === "error" && <p className="notice">{def.error.message}</p>}
      {def.status === "ok" && diff && <DiffView diff={diff} mode={mode} />}
      <p className="hint muted small">
        <span className="kbd">j</span>/<span className="kbd">k</span> next/previous change ·{" "}
        <span className="kbd">[</span>/<span className="kbd">]</span> older/newer step
      </p>
    </article>
  );
}
