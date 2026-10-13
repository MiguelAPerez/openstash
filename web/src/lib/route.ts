// Minimal hash router. Hash routing keeps the Go side simple: the server only
// ever serves index.html at "/" plus fingerprinted assets.
import { useEffect, useState } from "react";

export type Route =
  | { name: "home" }
  | { name: "explore"; key: string; version: string; query: URLSearchParams }
  | { name: "reference"; key: string; version: string; query: URLSearchParams }
  | { name: "timeline"; key: string; query: URLSearchParams }
  | { name: "notFound" };

export function parseHash(hash: string): Route {
  const raw = hash.replace(/^#/, "") || "/";
  const qIndex = raw.indexOf("?");
  const pathPart = qIndex >= 0 ? raw.slice(0, qIndex) : raw;
  const query = new URLSearchParams(qIndex >= 0 ? raw.slice(qIndex + 1) : "");
  const parts = pathPart.split("/").filter(Boolean).map(safeDecode);

  if (parts.length === 0) return { name: "home" };
  if (parts[0] !== "specs" || parts.length < 3) return { name: "notFound" };
  const key = parts[1];
  if (parts.length === 3 && parts[2] === "timeline") return { name: "timeline", key, query };
  if (parts[2] === "versions" && parts.length >= 4) {
    const version = parts[3];
    if (parts.length === 4) return { name: "explore", key, version, query };
    if (parts.length === 5 && parts[4] === "reference") return { name: "reference", key, version, query };
  }
  return { name: "notFound" };
}

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

const enc = encodeURIComponent;

function withQuery(path: string, query?: Record<string, string | undefined>): string {
  if (!query) return path;
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) if (v) qs.set(k, v);
  const s = qs.toString();
  return s ? `${path}?${s}` : path;
}

export const href = {
  home: () => "#/",
  explore: (key: string, version: string, query?: Record<string, string | undefined>) =>
    withQuery(`#/specs/${enc(key)}/versions/${enc(version)}`, query),
  reference: (key: string, version: string, query?: Record<string, string | undefined>) =>
    withQuery(`#/specs/${enc(key)}/versions/${enc(version)}/reference`, query),
  timeline: (key: string, query?: Record<string, string | undefined>) =>
    withQuery(`#/specs/${enc(key)}/timeline`, query),
};

/** Navigate to a hash href. replace=true avoids a history entry (for typing). */
export function navigate(to: string, replace = false) {
  if (to === window.location.hash) return;
  if (replace) {
    history.replaceState(null, "", to);
    window.dispatchEvent(new HashChangeEvent("hashchange"));
  } else {
    window.location.hash = to;
  }
}

export function useRoute(): Route {
  const [route, setRoute] = useState(() => parseHash(window.location.hash));
  useEffect(() => {
    const onChange = () => setRoute(parseHash(window.location.hash));
    window.addEventListener("hashchange", onChange);
    return () => window.removeEventListener("hashchange", onChange);
  }, []);
  return route;
}
