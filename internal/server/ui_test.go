package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func serve(srv *Server, method, url string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(method, url, nil))
	return rec
}

func TestUIIndex(t *testing.T) {
	srv, _ := testServer(t)
	rec := serve(srv, http.MethodGet, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content type = %q", ct)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("missing CSP: %q", csp)
	}
	// Either the built app or the placeholder, depending on whether `make web` ran.
	if !strings.Contains(rec.Body.String(), "openstash") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestUIUnknownPathsAndHiddenFiles(t *testing.T) {
	srv, _ := testServer(t)
	for _, url := range []string{"/nope", "/.gitkeep", "/assets/missing.js", "/v1/unknown"} {
		if rec := serve(srv, http.MethodGet, url); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", url, rec.Code)
		}
	}
}

func TestUIDoesNotShadowAPI(t *testing.T) {
	srv, _ := testServer(t)
	if rec := serve(srv, http.MethodGet, "/health"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("/health = %d %s", rec.Code, rec.Body.String())
	}
	if rec := serve(srv, http.MethodPost, "/"); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST / status = %d, want 405", rec.Code)
	}
}

func TestUIDisabled(t *testing.T) {
	st, _ := testServer(t)
	srv := NewWithOptions(st.store, ":0", 0, Options{DisableUI: true})
	if rec := serve(srv, http.MethodGet, "/"); rec.Code != http.StatusNotFound {
		t.Fatalf("GET / with UI disabled = %d, want 404", rec.Code)
	}
}

// The built-app branch, with a fixture standing in for the embedded dist.
func TestUIServesBuiltApp(t *testing.T) {
	mux := http.NewServeMux()
	registerUIFS(mux, fstest.MapFS{
		"index.html":     {Data: []byte(`<div id="root"></div>`)},
		"assets/app.js":  {Data: []byte("console.log(1)")},
		"assets/.hidden": {Data: []byte("x")},
		"favicon.svg":    {Data: []byte("<svg/>")},
	})
	do := func(url string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		return rec
	}

	rec := do("/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="root"`) || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("GET / = %d %q cache=%q", rec.Code, rec.Body.String(), rec.Header().Get("Cache-Control"))
	}
	if rec := do("/assets/app.js"); rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset = %d cache=%q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if rec := do("/favicon.svg"); rec.Code != http.StatusOK || strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("favicon = %d cache=%q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if rec := do("/index.html"); rec.Code != http.StatusMovedPermanently {
		t.Fatalf("/index.html = %d, want 301", rec.Code)
	}
	for _, url := range []string{"/assets/.hidden", "/assets", "/missing"} {
		if rec := do(url); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", url, rec.Code)
		}
	}
}

// The placeholder must render under its own CSP (no inline styles or scripts).
func TestUIPlaceholderFitsCSP(t *testing.T) {
	mux := http.NewServeMux()
	registerUIFS(mux, fstest.MapFS{})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "web UI was not included") {
		t.Fatalf("placeholder = %d %q", rec.Code, body)
	}
	if strings.Contains(body, "style=") || strings.Contains(body, "<style") || strings.Contains(body, "<script") {
		t.Fatalf("placeholder has inline code the CSP blocks: %q", body)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("placeholder /index.html = %d, want 301", rec.Code)
	}
}
