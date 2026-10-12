package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
