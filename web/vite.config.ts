import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";

const outDir = resolve(import.meta.dirname, "../internal/server/web/dist");

// emptyOutDir wipes the tracked .gitkeep that lets `go build` embed the
// directory before the UI has been built; put it back after every build.
const keepPlaceholder = (): Plugin => ({
  name: "openstash-keep-placeholder",
  apply: "build",
  closeBundle() {
    writeFileSync(resolve(outDir, ".gitkeep"), "");
  },
});

// The build is embedded into the Go binary (internal/server/ui.go).
// During `npm run dev`, API calls are proxied to a local `openstash serve`.
export default defineConfig({
  plugins: [react(), keepPlaceholder()],
  build: {
    outDir,
    emptyOutDir: true,
    chunkSizeWarningLimit: 4096,
    rollupOptions: {
      input: {
        main: resolve(import.meta.dirname, "index.html"),
        reference: resolve(import.meta.dirname, "reference.html"),
      },
    },
  },
  server: {
    proxy: {
      "/v1": process.env.OPENSTASH_API ?? "http://127.0.0.1:8080",
      "/health": process.env.OPENSTASH_API ?? "http://127.0.0.1:8080",
    },
  },
});
