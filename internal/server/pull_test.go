package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSpecFile(t *testing.T, path, version string) {
	t.Helper()
	data, err := json.Marshal(testDoc(version))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func pullRequest(key string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/specs/"+key+"/pull", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func doPull(t *testing.T, srv *Server, key string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, pullRequest(key))
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return rec.Code, body
}

func TestPullAddsNewVersion(t *testing.T) {
	srv, st := testServer(t)
	src := filepath.Join(t.TempDir(), "spec.json")
	writeSpecFile(t, src, "1.0.0")
	if _, _, err := st.Add("api", "1.0.0", src, "https://api.example", testDoc("1.0.0")); err != nil {
		t.Fatal(err)
	}

	code, body := doPull(t, srv, "api")
	if code != http.StatusOK || body["status"] != "up-to-date" || body["version"] != "1.0.0" {
		t.Fatalf("unchanged source: %d %v", code, body)
	}

	writeSpecFile(t, src, "1.1.0")
	code, body = doPull(t, srv, "api")
	if code != http.StatusCreated || body["status"] != "added" || body["previous"] != "1.0.0" {
		t.Fatalf("new source version: %d %v", code, body)
	}
	if !st.Exists("api", "1.1.0") {
		t.Fatal("1.1.0 was not stored")
	}
	meta, err := st.LoadMeta("api", "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Source != src || meta.Endpoint != "https://api.example" {
		t.Fatalf("meta did not carry over source/endpoint: %+v", meta)
	}

	// Pulling again is a no-op.
	code, body = doPull(t, srv, "api")
	if code != http.StatusOK || body["version"] != "1.1.0" {
		t.Fatalf("second pull: %d %v", code, body)
	}
}

func TestPullCustomTagIsUpToDate(t *testing.T) {
	srv, st := testServer(t)
	src := filepath.Join(t.TempDir(), "spec.json")
	writeSpecFile(t, src, "1.0.0")
	// Stored under a custom tag; the document's info.version matches the source.
	if _, _, err := st.Add("api", "prod", src, "", testDoc("1.0.0")); err != nil {
		t.Fatal(err)
	}
	code, body := doPull(t, srv, "api")
	if code != http.StatusOK || body["version"] != "prod" {
		t.Fatalf("got %d %v, want up-to-date at prod", code, body)
	}
	if st.Exists("api", "1.0.0") {
		t.Fatal("pull stored a duplicate of the custom-tagged version")
	}
}

func TestPullErrors(t *testing.T) {
	srv, st := testServer(t)

	if code, _ := doPull(t, srv, "missing"); code != http.StatusNotFound {
		t.Fatalf("unknown key: %d, want 404", code)
	}

	seedSpec(t, st, "gone", "1.0.0") // source "file.json" does not exist
	if code, _ := doPull(t, srv, "gone"); code != http.StatusUnprocessableEntity {
		t.Fatalf("unreadable source: %d, want 422", code)
	}

	src := filepath.Join(t.TempDir(), "spec.json")
	writeSpecFile(t, src, "")
	if _, _, err := st.Add("nover", "1.0.0", src, "", testDoc("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if code, _ := doPull(t, srv, "nover"); code != http.StatusUnprocessableEntity {
		t.Fatalf("source without info.version: %d, want 422", code)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/specs/gone/pull", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("no content type: %d, want 415", rec.Code)
	}
}
