import { useCallback, useEffect, useMemo, useState } from "react";
import { AddSpecDialog } from "./components/AddSpecDialog";
import { Sidebar } from "./components/Sidebar";
import { api } from "./lib/api";
import { href, useRoute } from "./lib/route";
import { groupSpecs, SpecsContext } from "./lib/specs";
import { useAsync } from "./lib/useAsync";
import { Explore } from "./pages/Explore";
import { Home } from "./pages/Home";
import Reference from "./pages/Reference";
import { Timeline } from "./pages/Timeline";

export function App() {
  const route = useRoute();
  const [reloadToken, setReloadToken] = useState(0);
  const specs = useAsync((signal) => api.listSpecs().then((e) => (signal.aborted ? [] : e)), [reloadToken], true);
  const [addOpen, setAddOpen] = useState(false);
  const [prefillKey, setPrefillKey] = useState<string | undefined>();
  const [toast, setToast] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const reload = useCallback(() => setReloadToken((n) => n + 1), []);
  const openAdd = useCallback((key?: string) => {
    setPrefillKey(key);
    setAddOpen(true);
  }, []);

  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(null), 4000);
    return () => clearTimeout(t);
  }, [toast]);

  useEffect(() => setSidebarOpen(false), [route]);

  const ctx = useMemo(
    () => ({
      groups: groupSpecs(specs.data ?? []),
      loading: specs.status === "loading",
      error: specs.status === "error" ? specs.error : undefined,
      reload,
      openAdd,
    }),
    [specs, reload, openAdd],
  );

  const currentRef =
    route.name === "explore" || route.name === "reference"
      ? `${route.key}@${route.version}`
      : route.name === "timeline"
        ? `${route.key} · timeline`
        : "";

  return (
    <SpecsContext.Provider value={ctx}>
      <header className="topbar">
        <button
          className="icon-btn only-narrow"
          type="button"
          aria-label="Toggle spec list"
          aria-controls="sidebar"
          aria-expanded={sidebarOpen}
          onClick={() => setSidebarOpen((o) => !o)}
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <a className="brand" href={href.home()}>
          <span className="brand-mark" aria-hidden="true" />
          <span>openstash</span>
        </a>
        <span className="topbar-ref">{currentRef}</span>
        <span className="spacer" />
        <a
          className="topbar-link"
          href="https://github.com/MiguelAPerez/openstash/blob/main/docs/serve-api.md"
          target="_blank"
          rel="noopener noreferrer"
        >
          API docs
        </a>
      </header>

      <div className="layout">
        <Sidebar route={route} open={sidebarOpen} onNavigate={() => setSidebarOpen(false)} />
        <main className="main">
          {route.name === "home" && <Home />}
          {route.name === "explore" && <Explore route={route} />}
          {route.name === "reference" && <Reference specKey={route.key} version={route.version} />}
          {route.name === "timeline" && <Timeline route={route} />}
          {route.name === "notFound" && (
            <div className="empty">
              Page not found. <a href={href.home()}>Back to all specs</a>
            </div>
          )}
        </main>
      </div>

      <AddSpecDialog
        open={addOpen}
        prefillKey={prefillKey}
        onClose={() => setAddOpen(false)}
        onAdded={(msg) => {
          setToast(msg);
          reload();
        }}
      />
      {toast && (
        <div className="toast" role="status">
          {toast}
        </div>
      )}
    </SpecsContext.Provider>
  );
}
