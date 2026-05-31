package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	content := []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if InfoVersion(doc) != "1.0.0" {
		t.Fatalf("version = %q, want 1.0.0", InfoVersion(doc))
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.yaml")
	content := []byte(`openapi: "3.0.0"
info:
  title: Test
  version: 2.0.0
paths: {}
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if InfoVersion(doc) != "2.0.0" {
		t.Fatalf("version = %q, want 2.0.0", InfoVersion(doc))
	}
}

func TestLoadFromEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestLoadFromNotOpenAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"name":"nope"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for non-openapi document")
	}
}

func TestGetOperation(t *testing.T) {
	doc := map[string]any{
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"operationId": "listPets",
					"summary":     "List pets",
					"tags":        []any{"pets"},
				},
			},
		},
	}

	op, err := GetOperation(doc, "/pets", "get")
	if err != nil {
		t.Fatal(err)
	}
	if op.Method != "GET" || op.Path != "/pets" || op.OperationID != "listPets" {
		t.Fatalf("unexpected operation: %+v", op)
	}
}

func TestGetOperationMissingPath(t *testing.T) {
	doc := map[string]any{"paths": map[string]any{}}
	_, err := GetOperation(doc, "/missing", "GET")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}
