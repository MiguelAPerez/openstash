package spec

import "testing"

func TestCompareOperations(t *testing.T) {
	baseline := map[string]any{
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

	target := map[string]any{
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

	result := Compare(baseline, target)

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
	summaryChange, ok := result.Operations.Changed[0].Changes["summary"]
	if !ok {
		t.Fatalf("expected summary change, got %+v", result.Operations.Changed[0].Changes)
	}
	if summaryChange.Baseline != "List pets" || summaryChange.Target != "List all pets" {
		t.Fatalf("summary change = %+v", summaryChange)
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

func TestOperationChanges(t *testing.T) {
	baseline := OperationIndex{
		Method:      "GET",
		Path:        "/pets",
		OperationID: "listPets",
		Summary:     "List pets",
		Tags:        []string{"pets"},
	}
	target := OperationIndex{
		Method:      "GET",
		Path:        "/pets",
		OperationID: "listPets",
		Summary:     "List all pets",
		Tags:        []string{"pets", "animals"},
	}

	changes := operationChanges(baseline, target)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}
	if changes["summary"].Baseline != "List pets" || changes["summary"].Target != "List all pets" {
		t.Fatalf("summary change = %+v", changes["summary"])
	}
}

func TestTagComparisonElementWise(t *testing.T) {
	// Join-based comparison would treat these as equal; element-wise must not.
	a := OperationIndex{Method: "GET", Path: "/x", Tags: []string{"a", "b"}}
	b := OperationIndex{Method: "GET", Path: "/x", Tags: []string{"a\x00b"}}
	if len(operationChanges(a, b)) == 0 {
		t.Fatal("expected tag diff for distinct tag slices")
	}

	same := OperationIndex{Method: "GET", Path: "/x", Tags: []string{"repository", "user"}}
	if len(operationChanges(same, same)) != 0 {
		t.Fatalf("identical tags should not produce changes: %+v", operationChanges(same, same))
	}
}
