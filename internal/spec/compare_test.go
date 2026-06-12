package spec

import "testing"

func TestCompareOperations(t *testing.T) {
	left := map[string]any{
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"operationId": "listPets",
					"summary":     "List pets",
					"tags":        []any{"pets"},
				},
			},
			"/pets/{id}": map[string]any{
				"delete": map[string]any{
					"operationId": "deletePet",
					"summary":     "Delete a pet",
				},
			},
		},
		"definitions": map[string]any{
			"Pet": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":   map[string]any{"type": "integer"},
					"name": map[string]any{"type": "string"},
				},
			},
			"Error": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"code": map[string]any{"type": "integer"},
				},
			},
		},
	}

	right := map[string]any{
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"operationId": "listPets",
					"summary":     "List all pets",
					"tags":        []any{"pets"},
				},
			},
			"/pets/{id}": map[string]any{
				"get": map[string]any{
					"operationId": "getPet",
					"summary":     "Get a pet",
				},
			},
		},
		"definitions": map[string]any{
			"Pet": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "integer"},
					"name":  map[string]any{"type": "string"},
					"color": map[string]any{"type": "string"},
				},
			},
			"Owner": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
		},
	}

	result := Compare(left, right)
	result.Left.Key = "left"
	result.Left.Version = "1"
	result.Right.Key = "right"
	result.Right.Version = "2"

	if result.Summary.Operations.Added != 1 {
		t.Fatalf("operations.added = %d, want 1", result.Summary.Operations.Added)
	}
	if result.Summary.Operations.Removed != 1 {
		t.Fatalf("operations.removed = %d, want 1", result.Summary.Operations.Removed)
	}
	if result.Summary.Operations.Changed != 1 {
		t.Fatalf("operations.changed = %d, want 1", result.Summary.Operations.Changed)
	}
	if result.Summary.Operations.Unchanged != 0 {
		t.Fatalf("operations.unchanged = %d, want 0", result.Summary.Operations.Unchanged)
	}

	if len(result.Operations.Added) != 1 || result.Operations.Added[0].Method != "GET" || result.Operations.Added[0].Path != "/pets/{id}" {
		t.Fatalf("unexpected added op: %+v", result.Operations.Added)
	}
	if len(result.Operations.Removed) != 1 || result.Operations.Removed[0].Method != "DELETE" {
		t.Fatalf("unexpected removed op: %+v", result.Operations.Removed)
	}
	if len(result.Operations.Changed) != 1 || result.Operations.Changed[0].Path != "/pets" {
		t.Fatalf("unexpected changed op: %+v", result.Operations.Changed)
	}

	if result.Summary.Schemas.Added != 1 || result.Schemas.Added[0] != "Owner" {
		t.Fatalf("schemas.added = %v, want [Owner]", result.Schemas.Added)
	}
	if result.Summary.Schemas.Removed != 1 || result.Schemas.Removed[0] != "Error" {
		t.Fatalf("schemas.removed = %v, want [Error]", result.Schemas.Removed)
	}
	if result.Summary.Schemas.Changed != 1 || result.Schemas.Changed[0].Name != "Pet" {
		t.Fatalf("schemas.changed = %+v, want Pet", result.Schemas.Changed)
	}
	if len(result.Schemas.Changed[0].FieldsAdded) != 1 || result.Schemas.Changed[0].FieldsAdded[0] != "color" {
		t.Fatalf("Pet fieldsAdded = %v, want [color]", result.Schemas.Changed[0].FieldsAdded)
	}
}

func TestCompareIdentical(t *testing.T) {
	doc := map[string]any{
		"paths": map[string]any{
			"/items": map[string]any{
				"get": map[string]any{"summary": "List items"},
			},
		},
	}
	result := Compare(doc, doc)
	if result.Summary.Operations.Added != 0 || result.Summary.Operations.Removed != 0 || result.Summary.Operations.Changed != 0 {
		t.Fatalf("expected no operation diffs, got %+v", result.Summary.Operations)
	}
	if result.Summary.Operations.Unchanged != 1 {
		t.Fatalf("operations.unchanged = %d, want 1", result.Summary.Operations.Unchanged)
	}
}
