import { useEffect, useState, type DependencyList } from "react";

export type AsyncState<T> =
  | { status: "loading"; data?: T }
  | { status: "ok"; data: T }
  | { status: "error"; error: Error; data?: T };

/**
 * Runs fn whenever deps change, aborting the previous run. With keepPrevious,
 * the last data stays visible while reloading so views don't flash empty.
 */
export function useAsync<T>(
  fn: (signal: AbortSignal) => Promise<T>,
  deps: DependencyList,
  keepPrevious = false,
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({ status: "loading" });
  useEffect(() => {
    const ctrl = new AbortController();
    setState((prev) => ({ status: "loading", data: keepPrevious ? prev.data : undefined }));
    fn(ctrl.signal).then(
      (data) => {
        if (!ctrl.signal.aborted) setState({ status: "ok", data });
      },
      (error: unknown) => {
        if (ctrl.signal.aborted) return;
        setState((prev) => ({
          status: "error",
          error: error instanceof Error ? error : new Error(String(error)),
          data: prev.data,
        }));
      },
    );
    return () => ctrl.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return state;
}

export function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}
