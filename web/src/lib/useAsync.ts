import { useEffect, useState, type DependencyList } from "react";

export type AsyncState<T> =
  | { status: "loading"; data?: T }
  | { status: "ok"; data: T }
  | { status: "error"; error: Error; data?: T };

function sameDeps(a: DependencyList, b: DependencyList): boolean {
  return a.length === b.length && a.every((v, i) => Object.is(v, b[i]));
}

/**
 * Runs fn whenever deps change, aborting the previous run. With keepPrevious,
 * the last data stays visible while reloading so views don't flash empty.
 *
 * The result is tied to the deps that produced it: on the render where deps
 * change (before the effect runs) it already reports "loading", so callers
 * never act on the previous deps' "ok" result (e.g. another spec's data).
 */
export function useAsync<T>(
  fn: (signal: AbortSignal) => Promise<T>,
  deps: DependencyList,
  keepPrevious = false,
): AsyncState<T> {
  const [state, setState] = useState<{ deps: DependencyList; result: AsyncState<T> }>(() => ({
    deps,
    result: { status: "loading" },
  }));
  useEffect(() => {
    const ctrl = new AbortController();
    setState((prev) => ({ deps, result: { status: "loading", data: keepPrevious ? prev.result.data : undefined } }));
    fn(ctrl.signal).then(
      (data) => {
        if (!ctrl.signal.aborted) setState({ deps, result: { status: "ok", data } });
      },
      (error: unknown) => {
        if (ctrl.signal.aborted) return;
        setState((prev) => ({
          deps,
          result: {
            status: "error",
            error: error instanceof Error ? error : new Error(String(error)),
            data: prev.result.data,
          },
        }));
      },
    );
    return () => ctrl.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  if (!sameDeps(state.deps, deps)) return { status: "loading", data: keepPrevious ? state.result.data : undefined };
  return state.result;
}

export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}
