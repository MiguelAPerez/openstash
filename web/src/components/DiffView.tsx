import { Fragment, useMemo, useState } from "react";
import { collapse, diffStats, lineDiff, type DiffLine, type DiffRow } from "../lib/diff";

export type DiffMode = "unified" | "split";

interface Props {
  oldText: string;
  newText: string;
  mode: DiffMode;
  context?: number;
}

export function DiffView({ oldText, newText, mode, context = 3 }: Props) {
  const lines = useMemo(() => lineDiff(oldText, newText), [oldText, newText]);
  const rows = useMemo(() => collapse(lines, context), [lines, context]);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const visible: DiffRow[] = useMemo(() => {
    const out: DiffRow[] = [];
    rows.forEach((r, i) => {
      if (r.type === "gap" && expanded.has(i)) r.hidden.forEach((line) => out.push({ type: "line", line }));
      else out.push(r);
    });
    return out;
  }, [rows, expanded]);

  // Map rendered gap rows back to their index in `rows` so expanding works.
  const gapIndex = useMemo(() => {
    const m = new Map<DiffRow, number>();
    rows.forEach((r, i) => r.type === "gap" && m.set(r, i));
    return m;
  }, [rows]);

  const expand = (row: DiffRow) => {
    const i = gapIndex.get(row);
    if (i !== undefined) setExpanded((s) => new Set(s).add(i));
  };

  if (lines.length === 0) return <p className="muted">Both sides are empty.</p>;
  if (lines.every((l) => l.kind === "same"))
    return <p className="muted">No textual differences (only key order or formatting changed).</p>;

  return (
    <div className="diff-scroll">
      <table className={`diff diff-${mode}`}>
        <tbody>
          {mode === "unified" ? <UnifiedRows rows={visible} onExpand={expand} /> : <SplitRows rows={visible} onExpand={expand} />}
        </tbody>
      </table>
    </div>
  );
}

function GapRow({ row, cols, onExpand }: { row: Extract<DiffRow, { type: "gap" }>; cols: number; onExpand: (r: DiffRow) => void }) {
  return (
    <tr className="diff-gap">
      <td colSpan={cols}>
        <button type="button" className="link-btn" onClick={() => onExpand(row)}>
          ⋯ {row.hidden.length} unchanged lines
        </button>
      </td>
    </tr>
  );
}

const sign = { add: "+", del: "−", same: " " } as const;

function UnifiedRows({ rows, onExpand }: { rows: DiffRow[]; onExpand: (r: DiffRow) => void }) {
  return (
    <>
      {rows.map((r, i) =>
        r.type === "gap" ? (
          <GapRow key={i} row={r} cols={4} onExpand={onExpand} />
        ) : (
          <tr key={i} className={`diff-${r.line.kind}`}>
            <td className="ln">{r.line.oldNo ?? ""}</td>
            <td className="ln">{r.line.newNo ?? ""}</td>
            <td className="sign">{sign[r.line.kind]}</td>
            <td className="code-cell">{r.line.text}</td>
          </tr>
        ),
      )}
    </>
  );
}

type Pair = { left?: DiffLine; right?: DiffLine };

function SplitRows({ rows, onExpand }: { rows: DiffRow[]; onExpand: (r: DiffRow) => void }) {
  // Pair each run of deletions with the run of additions that follows it.
  const out: Array<{ gap: DiffRow } | { pair: Pair }> = [];
  let dels: DiffLine[] = [];
  let adds: DiffLine[] = [];
  const flush = () => {
    const n = Math.max(dels.length, adds.length);
    for (let i = 0; i < n; i++) out.push({ pair: { left: dels[i], right: adds[i] } });
    dels = [];
    adds = [];
  };
  for (const r of rows) {
    if (r.type === "gap") {
      flush();
      out.push({ gap: r });
    } else if (r.line.kind === "del") {
      if (adds.length) flush();
      dels.push(r.line);
    } else if (r.line.kind === "add") adds.push(r.line);
    else {
      flush();
      out.push({ pair: { left: r.line, right: r.line } });
    }
  }
  flush();

  return (
    <>
      {out.map((item, i) => {
        if ("gap" in item) return <GapRow key={i} row={item.gap as Extract<DiffRow, { type: "gap" }>} cols={4} onExpand={onExpand} />;
        const { left, right } = item.pair;
        return (
          <tr key={i}>
            <Fragment>
              <td className={`ln ${left ? `diff-${left.kind}` : "diff-empty"}`}>{left?.oldNo ?? ""}</td>
              <td className={`code-cell ${left ? `diff-${left.kind}` : "diff-empty"}`}>{left?.text ?? ""}</td>
              <td className={`ln ${right ? `diff-${right.kind}` : "diff-empty"}`}>{right?.newNo ?? ""}</td>
              <td className={`code-cell ${right ? `diff-${right.kind}` : "diff-empty"}`}>{right?.text ?? ""}</td>
            </Fragment>
          </tr>
        );
      })}
    </>
  );
}

export function DiffStats({ oldText, newText }: { oldText: string; newText: string }) {
  const { added, removed } = useMemo(() => diffStats(lineDiff(oldText, newText)), [oldText, newText]);
  return (
    <span className="diff-stats">
      <span className="plus">+{added}</span> <span className="minus">−{removed}</span>
    </span>
  );
}
