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

func TestCompareDetailedDetectsDefinitionChanges(t *testing.T) {
	baseline := map[string]any{
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"summary":    "List pets",
					"parameters": []any{map[string]any{"name": "limit", "in": "query"}},
					"responses":  map[string]any{"200": map[string]any{"description": "ok"}},
				},
				"post": map[string]any{"summary": "Create pet"},
			},
		},
		"components": map[string]any{"schemas": map[string]any{
			"Pet": map[string]any{"type": "object", "properties": map[string]any{
				"id": map[string]any{"type": "integer"},
			}},
			"Same": map[string]any{"type": "string"},
		}},
	}
	target := map[string]any{
		"paths": map[string]any{
			"/pets": map[string]any{
				"get": map[string]any{
					"summary":    "List all pets",
					"parameters": []any{map[string]any{"name": "limit", "in": "query", "required": true}},
					"responses":  map[string]any{"200": map[string]any{"description": "ok"}},
				},
				"post": map[string]any{"summary": "Create pet", "deprecated": true},
			},
		},
		"components": map[string]any{"schemas": map[string]any{
			"Pet": map[string]any{"type": "object", "properties": map[string]any{
				"id": map[string]any{"type": "string"},
			}},
			"Same": map[string]any{"type": "string"},
		}},
	}

	// Plain Compare only sees the summary change.
	if got := Compare(baseline, target).Summary.Operations.Changed; got != 1 {
		t.Fatalf("Compare changed ops = %d, want 1", got)
	}

	res := CompareDetailed(baseline, target)
	if res.Summary.Operations.Changed != 2 || res.Summary.Operations.Unchanged != 0 {
		t.Fatalf("operation summary = %+v", res.Summary.Operations)
	}
	get, post := res.Operations.Changed[0], res.Operations.Changed[1]
	if get.Method != "GET" || post.Method != "POST" {
		t.Fatalf("changed ops not sorted: %+v", res.Operations.Changed)
	}
	if _, ok := get.Changes["summary"]; !ok {
		t.Fatalf("GET lost index-level summary change: %+v", get.Changes)
	}
	if _, ok := get.Changes["parameters"]; !ok {
		t.Fatalf("GET missing parameters change: %+v", get.Changes)
	}
	if _, ok := get.Changes["responses"]; ok {
		t.Fatalf("GET responses unchanged but reported: %+v", get.Changes)
	}
	if _, ok := post.Changes["deprecated"]; !ok || len(post.Changes) != 1 {
		t.Fatalf("POST changes = %+v, want only deprecated", post.Changes)
	}

	if res.Summary.Schemas.Changed != 1 || res.Summary.Schemas.Unchanged != 1 {
		t.Fatalf("schema summary = %+v", res.Summary.Schemas)
	}
	if s := res.Schemas.Changed[0]; s.Name != "Pet" || !s.DefinitionChanged || len(s.FieldsAdded)+len(s.FieldsRemoved) != 0 {
		t.Fatalf("schema diff = %+v", s)
	}
}

func TestEffectiveOperation(t *testing.T) {
	doc := map[string]any{
		"security": []any{map[string]any{"key": []any{}}},
		"components": map[string]any{
			"parameters": map[string]any{
				"Id":    map[string]any{"name": "id", "in": "path", "description": "shared"},
				"Alias": map[string]any{"$ref": "#/components/parameters/Id"},
			},
			"responses": map[string]any{
				"Ok": map[string]any{"description": "ok", "headers": map[string]any{
					"X-Rate": map[string]any{"$ref": "#/components/headers/Rate"},
				}},
			},
			"headers":       map[string]any{"Rate": map[string]any{"description": "rate"}},
			"requestBodies": map[string]any{"Pet": map[string]any{"required": true}},
		},
		"paths": map[string]any{"/a/{id}": map[string]any{
			"parameters": []any{
				map[string]any{"$ref": "#/components/parameters/Alias"},
				map[string]any{"name": "trace", "in": "header"},
			},
			"get": map[string]any{
				"parameters":  []any{map[string]any{"name": "trace", "in": "header", "description": "own"}},
				"requestBody": map[string]any{"$ref": "#/components/requestBodies/Pet"},
				"responses": map[string]any{
					"200": map[string]any{"$ref": "#/components/responses/Ok"},
					"404": map[string]any{"$ref": "other.yaml#/NotFound"},
				},
			},
		}},
	}
	op := EffectiveOperation(doc, "/a/{id}", "GET")
	params, _ := op["parameters"].([]any)
	if len(params) != 2 || params[0].(map[string]any)["description"] != "shared" || params[1].(map[string]any)["description"] != "own" {
		t.Fatalf("parameters = %v, want shared id then own trace", params)
	}
	if op["requestBody"].(map[string]any)["required"] != true {
		t.Fatalf("requestBody not resolved: %v", op["requestBody"])
	}
	ok := op["responses"].(map[string]any)["200"].(map[string]any)
	if ok["description"] != "ok" || ok["headers"].(map[string]any)["X-Rate"].(map[string]any)["description"] != "rate" {
		t.Fatalf("response or header not resolved: %v", ok)
	}
	if op["responses"].(map[string]any)["404"].(map[string]any)["$ref"] != "other.yaml#/NotFound" {
		t.Fatal("external ref should be left alone")
	}
	if op["security"] == nil {
		t.Fatal("document-level security should apply")
	}
	// The source document must not be modified.
	if _, ok := doc["components"].(map[string]any)["responses"].(map[string]any)["Ok"].(map[string]any)["headers"].(map[string]any)["X-Rate"].(map[string]any)["$ref"]; !ok {
		t.Fatal("EffectiveOperation mutated the document")
	}
	if EffectiveOperation(doc, "/a/{id}", "POST") != nil || EffectiveOperation(doc, "/missing", "GET") != nil {
		t.Fatal("expected nil for missing operation")
	}
	if EffectiveOperation(map[string]any{}, "/a", "GET") != nil {
		t.Fatal("expected nil when spec has no paths")
	}
}

// Changes that only live in shared components or path-level parameters must
// still flag the operations that use them.
func TestCompareDetailedFollowsSharedDefinitions(t *testing.T) {
	doc := func(pageType, okDesc, bodyType, traceDesc string) map[string]any {
		return map[string]any{
			"components": map[string]any{
				"parameters":    map[string]any{"Page": map[string]any{"name": "page", "in": "query", "schema": map[string]any{"type": pageType}}},
				"responses":     map[string]any{"Ok": map[string]any{"description": okDesc}},
				"requestBodies": map[string]any{"Pet": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": bodyType}}}}},
			},
			"paths": map[string]any{
				"/pets": map[string]any{
					"get": map[string]any{
						"parameters": []any{map[string]any{"$ref": "#/components/parameters/Page"}},
						"responses":  map[string]any{"200": map[string]any{"$ref": "#/components/responses/Ok"}},
					},
					"post": map[string]any{"requestBody": map[string]any{"$ref": "#/components/requestBodies/Pet"}},
				},
				"/pets/{id}": map[string]any{
					"parameters": []any{map[string]any{"name": "trace", "in": "header", "description": traceDesc}},
					"get":        map[string]any{"summary": "Get pet"},
				},
			},
		}
	}
	base := doc("integer", "ok", "object", "trace id")

	cases := []struct {
		name   string
		target map[string]any
		op     string
		field  string
	}{
		{"shared parameter", doc("string", "ok", "object", "trace id"), "GET /pets", "parameters"},
		{"shared response", doc("integer", "fine", "object", "trace id"), "GET /pets", "responses"},
		{"shared request body", doc("integer", "ok", "string", "trace id"), "POST /pets", "requestBody"},
		{"path-level parameter", doc("integer", "ok", "object", "request id"), "GET /pets/{id}", "parameters"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := CompareDetailed(base, tc.target)
			if len(res.Operations.Changed) != 1 {
				t.Fatalf("changed ops = %+v, want just %s", res.Operations.Changed, tc.op)
			}
			c := res.Operations.Changed[0]
			if c.Method+" "+c.Path != tc.op {
				t.Fatalf("changed op = %s %s, want %s", c.Method, c.Path, tc.op)
			}
			if _, ok := c.Changes[tc.field]; !ok || c.ExtensionsOnly {
				t.Fatalf("changes = %+v (extensionsOnly=%v), want %s", c.Changes, c.ExtensionsOnly, tc.field)
			}
			if res.Summary.Operations.Changed != 1 {
				t.Fatalf("summary = %+v", res.Summary.Operations)
			}
		})
	}
}

func TestCompareDetailedFlagsExtensionOnlyChanges(t *testing.T) {
	doc := func(pkg, xFieldType string) map[string]any {
		return map[string]any{"definitions": map[string]any{
			"Repo": map[string]any{"type": "object", "x-go-package": pkg},
			"Flag": map[string]any{"type": "object", "properties": map[string]any{
				// A property literally named x-*: data, not a vendor extension.
				"x-enabled": map[string]any{"type": xFieldType},
			}},
		}}
	}
	res := CompareDetailed(doc("a/models", "boolean"), doc("b/models", "string"))
	got := map[string]bool{}
	for _, s := range res.Schemas.Changed {
		got[s.Name] = s.ExtensionsOnly
	}
	if extOnly, ok := got["Repo"]; !ok || !extOnly {
		t.Fatalf("Repo should be an extension-only change: %+v", res.Schemas.Changed)
	}
	if extOnly, ok := got["Flag"]; !ok || extOnly {
		t.Fatalf("Flag changed a real property and must not be extension-only: %+v", res.Schemas.Changed)
	}
}
