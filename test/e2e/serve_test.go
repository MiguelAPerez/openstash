package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func buildOpenstash(t *testing.T, root, outPath string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", outPath, "./cmd/openstash")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
}

func startServe(t *testing.T, bin, storeDir, addr string) (context.CancelFunc, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "serve", "--store", storeDir, "--addr", addr)
	cmd.Env = append(os.Environ(), "OPENSTASH_STORE="+storeDir)
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	return cancel, func() {
		cancel()
		_ = cmd.Wait()
	}
}

func waitHealthy(t *testing.T, base string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(base + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("server did not become healthy")
}

func mustJSONRequest(t *testing.T, client *http.Client, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s %s: %v\nbody: %s", method, url, err, data)
		}
	}
	return resp.StatusCode, out
}

func writeSpecFile(t *testing.T, dir string) string {
	t.Helper()
	doc := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "E2E", "version": "1.0.0"},
		"paths": map[string]any{
			"/items": map[string]any{
				"get": map[string]any{"summary": "List items", "tags": []any{"items"}},
			},
		},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "spec.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestServeE2E builds openstash, runs `openstash serve`, and exercises the HTTP API over TCP.
func TestServeE2E(t *testing.T) {
	root := repoRoot(t)
	storeDir := t.TempDir()
	specPath := writeSpecFile(t, t.TempDir())

	bin := filepath.Join(t.TempDir(), "openstash")
	buildOpenstash(t, root, bin)

	addr := freeAddr(t)
	base := "http://" + addr
	cancel, stop := startServe(t, bin, storeDir, addr)
	defer func() {
		cancel()
		stop()
	}()

	waitHealthy(t, base)
	client := &http.Client{Timeout: 5 * time.Second}

	code, body := mustJSONRequest(t, client, http.MethodGet, base+"/health", nil)
	if code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("GET /health = %d %#v", code, body)
	}

	code, body = mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /v1/specs (empty) = %d", code)
	}
	entries, _ := body["entries"].([]any)
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(entries))
	}

	code, body = mustJSONRequest(t, client, http.MethodPost, base+"/v1/specs", map[string]string{
		"key":  "api",
		"from": specPath,
	})
	if code != http.StatusCreated || body["status"] != "added" {
		t.Fatalf("POST /v1/specs = %d %#v", code, body)
	}

	code, body = mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /v1/specs = %d", code)
	}
	entries, _ = body["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	code, dump := mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs/api", nil)
	if code != http.StatusOK || dump["openapi"] != "3.1.0" {
		t.Fatalf("GET /v1/specs/api = %d", code)
	}

	code, body = mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs/api/versions", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /v1/specs/api/versions = %d", code)
	}
	versions, _ := body["versions"].([]any)
	if len(versions) != 1 {
		t.Fatalf("versions = %v", versions)
	}

	code, dump = mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs/api/versions/1.0.0", nil)
	if code != http.StatusOK || dump["openapi"] != "3.1.0" {
		t.Fatalf("GET pinned dump = %d", code)
	}

	opsURL := base + "/v1/specs/api/versions/1.0.0/operations"
	code, body = mustJSONRequest(t, client, http.MethodGet, opsURL+"?q=items", nil)
	if code != http.StatusOK {
		t.Fatalf("search = %d %#v", code, body)
	}
	hits, _ := body["hits"].([]any)
	if len(hits) == 0 {
		t.Fatalf("search hits empty: %#v", body)
	}

	code, body = mustJSONRequest(t, client, http.MethodGet, opsURL+"?detail=show&path=/items&method=GET", nil)
	if code != http.StatusOK || body["operation"] == nil {
		t.Fatalf("show = %d %#v", code, body)
	}

	code, body = mustJSONRequest(t, client, http.MethodGet, opsURL+"?detail=gather&q=items", nil)
	if code != http.StatusOK {
		t.Fatalf("gather = %d %#v", code, body)
	}
	gatherOps, _ := body["operations"].([]any)
	if len(gatherOps) == 0 {
		t.Fatalf("gather operations empty: %#v", body)
	}

	code, _ = mustJSONRequest(t, client, http.MethodPost, base+"/v1/specs", map[string]string{
		"key":  "api",
		"from": specPath,
	})
	if code != http.StatusConflict {
		t.Fatalf("POST duplicate = %d, want 409", code)
	}

	code, _ = mustJSONRequest(t, client, http.MethodGet, base+"/v1/specs/missing", nil)
	if code != http.StatusNotFound {
		t.Fatalf("GET missing = %d, want 404", code)
	}
}
