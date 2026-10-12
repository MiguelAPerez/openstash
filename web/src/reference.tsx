// Standalone page hosting the Scalar API reference. It runs inside an iframe
// on the Reference tab because Scalar manages location.hash for its own deep
// links, which would otherwise collide with the app's hash router.
import { ApiReferenceReact } from "@scalar/api-reference-react";
import "@scalar/api-reference-react/style.css";
import "./reference.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { api, type JsonObject } from "./lib/api";
import { useAsync } from "./lib/useAsync";

const params = new URLSearchParams(window.location.search);
const key = params.get("key") ?? "";
const version = params.get("version") ?? "";
const dark = window.matchMedia("(prefers-color-scheme: dark)").matches;

function ReferencePage() {
  const doc = useAsync<JsonObject>((signal) => api.dump(key, version, signal), []);
  if (doc.status === "loading") return <p style={{ font: "14px system-ui", padding: 24 }}>Loading {key}@{version}…</p>;
  if (doc.status === "error")
    return <p style={{ font: "14px system-ui", padding: 24, color: "#c62828" }}>{doc.error.message}</p>;
  return (
    <ApiReferenceReact
      configuration={{
        content: doc.data,
        darkMode: dark,
        hideDarkModeToggle: true,
        hideClientButton: true,
        hideTestRequestButton: true,
        withDefaultFonts: false,
        telemetry: false,
        mcp: { disabled: true },
        agent: { disabled: true },
        showDeveloperTools: "never",
        documentDownloadType: "none",
      }}
    />
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ReferencePage />
  </StrictMode>,
);
