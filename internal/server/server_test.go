package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MiguelAPerez/openstash/internal/store"
)

func testServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(st, ":0", 0), st
}

func testDoc(version string) map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "Test", "version": version},
		"paths": map[string]any{
			"/items": map[string]any{
				"get": map[string]any{"summary": "List items", "tags": []any{"items"}},
			},
			"/items/{id}": map[string]any{
				"get": map[string]any{"summary": "Get item"},
			},
		},
	}
}

func seedSpec(t *testing.T, st *store.Store, key, version string) {
	t.Helper()
	if _, _, err := st.Add(key, version, "file.json", "", testDoc(version)); err != nil {
		t.Fatal(err)
	}
}

func TestHealth(t *testing.T) {
	srv, _ := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v", body["status"])
	}
}

func TestListAndDump(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/api", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dump latest status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/api/versions/1.0.0", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("dump pinned status = %d", rec.Code)
	}
}

func TestListVersions(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")
	seedSpec(t, st, "api", "2.0.0")

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/api/versions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	vers, _ := body["versions"].([]any)
	if len(vers) != 2 {
		t.Fatalf("versions = %v, want 2", vers)
	}
}

func TestAddSpec(t *testing.T) {
	srv, st := testServer(t)
	specPath := filepath.Join(t.TempDir(), "spec.json")
	data, err := json.Marshal(testDoc("3.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]string{
		"key":  "petstore",
		"from": specPath,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body))
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if !st.Exists("petstore", "3.0.0") {
		t.Fatal("spec was not stored")
	}
}

func TestAddConflict(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")

	specPath := filepath.Join(t.TempDir(), "spec.json")
	data, _ := json.Marshal(testDoc("1.0.0"))
	_ = os.WriteFile(specPath, data, 0o644)

	body, _ := json.Marshal(map[string]string{"key": "api", "from": specPath})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestOperationsSearchAndShow(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/api/versions/1.0.0/operations?q=items", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/api/versions/1.0.0/operations?detail=show&path=/items&method=GET", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("show status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	srv, _ := testServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/specs/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAddRejectsTraversalKey(t *testing.T) {
	srv, st := testServer(t)
	specPath := filepath.Join(t.TempDir(), "spec.json")
	data, _ := json.Marshal(testDoc("1.0.0"))
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"..", "../escape", "a/b", "a@b"} {
		body, _ := json.Marshal(map[string]string{"key": key, "from": specPath})
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("key %q: status = %d, want 400: %s", key, rec.Code, rec.Body.String())
		}
	}

	// Nothing should have been written outside the store's specs/ directory.
	escaped := filepath.Join(st.Root, "spec.json")
	if _, err := os.Stat(escaped); err == nil {
		t.Fatalf("traversal wrote spec outside specs/: %s", escaped)
	}
}

func TestAddRejectsUnknownFields(t *testing.T) {
	srv, _ := testServer(t)
	body := []byte(`{"key":"api","from":"x","bogus":"oops"}`)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestGetRoutesRejectTraversal(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")

	// Values with no '/' so ServeMux path-cleaning can't mask the guard:
	// ".." embedded in a segment and a smuggled key@version both must 400.
	cases := []string{
		"/v1/specs/a..b",
		"/v1/specs/api@1.0.0",
		"/v1/specs/a..b/versions",
		"/v1/specs/api/versions/v..1",
		"/v1/specs/a..b/versions/1.0.0",
		"/v1/specs/api/versions/v..1/operations",
	}
	for _, path := range cases {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("GET %s: status = %d, want 400: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAddRejectsOversizedBody(t *testing.T) {
	srv, _ := testServer(t)
	// Pad an otherwise-valid body past the 64 KiB cap.
	big := make([]byte, 128<<10)
	for i := range big {
		big[i] = 'a'
	}
	body, _ := json.Marshal(map[string]string{"key": "api", "from": "x", "endpoint": string(big)})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestConfigurableBodyCap(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A tiny 32-byte cap should reject an otherwise-small valid-looking body.
	srv := New(st, ":0", 32)
	body, _ := json.Marshal(map[string]string{"key": "api", "from": "https://example.test/openapi.json"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/specs", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for body over 32-byte cap: %s", rec.Code, rec.Body.String())
	}
}

func TestNewBodyCapDefaults(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := New(st, ":0", 0).maxBodyBytes; got != DefaultMaxBodyBytes {
		t.Fatalf("maxBodyBytes with 0 = %d, want default %d", got, DefaultMaxBodyBytes)
	}
	if got := New(st, ":0", -5).maxBodyBytes; got != DefaultMaxBodyBytes {
		t.Fatalf("maxBodyBytes with negative = %d, want default %d", got, DefaultMaxBodyBytes)
	}
	if got := New(st, ":0", 1024).maxBodyBytes; got != 1024 {
		t.Fatalf("maxBodyBytes with 1024 = %d, want 1024", got)
	}
}

func TestValidatePathSegment(t *testing.T) {
	valid := []string{"api", "petstore", "1.0.0", "v2-beta", "a.b.c"}
	for _, v := range valid {
		if err := validatePathSegment("key", v); err != nil {
			t.Errorf("validatePathSegment(%q) = %v, want nil", v, err)
		}
	}
	invalid := []string{"", ".", "..", "../x", "x/..", "a/b", `a\b`, "key@1.0"}
	for _, v := range invalid {
		if err := validatePathSegment("key", v); err == nil {
			t.Errorf("validatePathSegment(%q) = nil, want error", v)
		}
	}
}
