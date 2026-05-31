package spec

import "testing"

func TestBuildIndex(t *testing.T) {
	doc := map[string]any{
		"paths": map[string]any{
			"/users": map[string]any{
				"get": map[string]any{
					"operationId": "listUsers",
					"summary":     "List users",
					"tags":        []any{"users"},
				},
			},
		},
	}

	index := BuildIndex(doc)
	if len(index) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(index))
	}

	op := index[0]
	if op.Method != "GET" {
		t.Errorf("method: got %q, want GET", op.Method)
	}
	if op.Path != "/users" {
		t.Errorf("path: got %q, want /users", op.Path)
	}
	if op.OperationID != "listUsers" {
		t.Errorf("operationId: got %q, want listUsers", op.OperationID)
	}
	if op.Summary != "List users" {
		t.Errorf("summary: got %q, want List users", op.Summary)
	}
	if len(op.Tags) != 1 || op.Tags[0] != "users" {
		t.Errorf("tags: got %v, want [users]", op.Tags)
	}
}

func TestInfoVersion(t *testing.T) {
	doc := map[string]any{
		"info": map[string]any{
			"version": "2.0.0",
		},
	}

	if got := InfoVersion(doc); got != "2.0.0" {
		t.Errorf("InfoVersion: got %q, want 2.0.0", got)
	}
}

func TestInfoVersionMissing(t *testing.T) {
	if got := InfoVersion(map[string]any{}); got != "" {
		t.Errorf("InfoVersion: got %q, want empty", got)
	}
}
