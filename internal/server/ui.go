package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// webFS holds the browser UI built from web/ (npm run build writes to
// internal/server/web/dist). The directory is committed empty, so a plain
// `go build` without the UI build still compiles and serves a placeholder.
//
//go:embed all:web/dist
var webFS embed.FS

// uiCSP keeps the UI to same-origin assets. Spec content is untrusted (it is
// fetched from arbitrary URLs), so the UI renders it as text only and the policy
// blocks inline script and styles as a second line of defense.
const uiCSP = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"

const uiPlaceholder = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>openstash</title>
<meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="font-family:system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;line-height:1.5">
<h1>openstash</h1>
<p>The web UI was not included in this build. Build it with <code>make web</code>
(or <code>cd web &amp;&amp; npm ci &amp;&amp; npm run build</code>) and rebuild the binary.</p>
<p>The HTTP API is available under <a href="/v1/specs">/v1/specs</a>.</p>
</body></html>
`

// registerUI serves the single-page app at "/" and its static assets. The app
// uses hash routing, so only real files are served; other paths 404.
func (s *Server) registerUI(mux *http.ServeMux) {
	dist, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		panic(err) // embed path is fixed at compile time
	}
	_, statErr := fs.Stat(dist, "index.html")
	built := statErr == nil
	files := http.FileServer(http.FS(dist))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		setUIHeaders(w)
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" || name == "index.html" {
			if !built {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(uiPlaceholder))
				return
			}
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, dist, "index.html")
			return
		}
		if strings.HasPrefix(path.Base(name), ".") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if st, err := fs.Stat(dist, name); err != nil || st.IsDir() {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if strings.HasPrefix(name, "assets/") {
			// Vite fingerprints asset file names, so they can be cached forever.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

func setUIHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", uiCSP)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
}
