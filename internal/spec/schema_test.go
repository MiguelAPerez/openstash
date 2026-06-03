package spec

import (
	"strings"
	"testing"
	"time"
)

// ---- fixtures ---------------------------------------------------------------

// openapi3Doc is a minimal OpenAPI 3 doc with two schemas, enums, refs, arrays.
func openapi3Doc() map[string]any {
	return map[string]any{
		"openapi": "3.0.0",
		"info":    map[string]any{"title": "Test", "version": "1.0.0"},
		"components": map[string]any{
			"schemas": map[string]any{
				"Status": map[string]any{
					"type":        "string",
					"description": "PR status",
					"enum":        []any{"open", "closed", "draft"},
				},
				"Label": map[string]any{
					"type":  "object",
					"title": "Label",
					"properties": map[string]any{
						"id":   map[string]any{"type": "integer"},
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"id", "name"},
				},
				"CreatePROption": map[string]any{
					"type":  "object",
					"title": "CreatePROption",
					"properties": map[string]any{
						"title": map[string]any{"type": "string", "description": "PR title"},
						"draft": map[string]any{"type": "boolean"},
						"state": map[string]any{"$ref": "#/components/schemas/Status"},
						"labels": map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/Label"},
						},
					},
					"required": []any{"title"},
				},
			},
		},
	}
}

// swagger2Doc is a minimal Swagger 2.0 doc with the same logical structure.
func swagger2Doc() map[string]any {
	return map[string]any{
		"swagger": "2.0",
		"info":    map[string]any{"title": "Test", "version": "2.0.0"},
		"definitions": map[string]any{
			"Status": map[string]any{
				"type":        "string",
				"description": "PR status",
				"enum":        []any{"open", "closed", "draft"},
			},
			"Label": map[string]any{
				"type":  "object",
				"title": "Label",
				"properties": map[string]any{
					"id":   map[string]any{"type": "integer"},
					"name": map[string]any{"type": "string"},
				},
				"required": []any{"id", "name"},
			},
			"CreatePROption": map[string]any{
				"type":  "object",
				"title": "CreatePROption",
				"properties": map[string]any{
					"title": map[string]any{"type": "string", "description": "PR title"},
					"draft": map[string]any{"type": "boolean"},
					"state": map[string]any{"$ref": "#/definitions/Status"},
					"labels": map[string]any{
						"type":  "array",
						"items": map[string]any{"$ref": "#/definitions/Label"},
					},
				},
				"required": []any{"title"},
			},
		},
	}
}

// ---- SchemaNames ------------------------------------------------------------

func TestSchemaNames_OpenAPI3(t *testing.T) {
	names := SchemaNames(openapi3Doc())
	if len(names) != 3 {
		t.Fatalf("expected 3 schema names, got %d: %v", len(names), names)
	}
	// Must be sorted.
	if names[0] != "CreatePROption" || names[1] != "Label" || names[2] != "Status" {
		t.Fatalf("unexpected sorted names: %v", names)
	}
}

func TestSchemaNames_Swagger2(t *testing.T) {
	names := SchemaNames(swagger2Doc())
	if len(names) != 3 {
		t.Fatalf("expected 3 schema names, got %d: %v", len(names), names)
	}
}

func TestSchemaNames_NoContainer(t *testing.T) {
	names := SchemaNames(map[string]any{"openapi": "3.0.0"})
	if names != nil {
		t.Fatalf("expected nil for doc with no schemas container, got %v", names)
	}
}

// ---- GetSchema --------------------------------------------------------------

func TestGetSchema_Found(t *testing.T) {
	for _, doc := range []map[string]any{openapi3Doc(), swagger2Doc()} {
		node, err := GetSchema(doc, "Label")
		if err != nil {
			t.Fatalf("GetSchema: %v", err)
		}
		if node["type"] != "object" {
			t.Fatalf("expected type=object, got %v", node["type"])
		}
	}
}

func TestGetSchema_Missing(t *testing.T) {
	_, err := GetSchema(openapi3Doc(), "Nonexistent")
	if err == nil {
		t.Fatal("expected error for missing schema")
	}
	// Error message should mention count of available schemas.
	if !strings.Contains(err.Error(), "3 schemas available") {
		t.Fatalf("error message missing schema count: %v", err)
	}
}

func TestGetSchema_NoContainer(t *testing.T) {
	_, err := GetSchema(map[string]any{}, "Anything")
	if err == nil {
		t.Fatal("expected error when no schemas container")
	}
}

// ---- ResolveRef -------------------------------------------------------------

func TestResolveRef_OpenAPI3(t *testing.T) {
	doc := openapi3Doc()
	node, err := ResolveRef(doc, "#/components/schemas/Status")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if node["type"] != "string" {
		t.Fatalf("expected type=string, got %v", node["type"])
	}
}

func TestResolveRef_Swagger2(t *testing.T) {
	doc := swagger2Doc()
	node, err := ResolveRef(doc, "#/definitions/Label")
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if node["title"] != "Label" {
		t.Fatalf("expected title=Label, got %v", node["title"])
	}
}

func TestResolveRef_Missing(t *testing.T) {
	_, err := ResolveRef(openapi3Doc(), "#/components/schemas/Missing")
	if err == nil {
		t.Fatal("expected error for missing ref")
	}
}

func TestResolveRef_ExternalError(t *testing.T) {
	_, err := ResolveRef(openapi3Doc(), "https://example.com/schema.json")
	if err == nil {
		t.Fatal("expected error for external ref")
	}
}

func TestResolveRef_Unescape(t *testing.T) {
	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Foo/Bar": map[string]any{"type": "object"},
			},
		},
	}
	// ~1 should unescape to /
	node, err := ResolveRef(doc, "#/components/schemas/Foo~1Bar")
	if err != nil {
		t.Fatalf("ResolveRef with ~1: %v", err)
	}
	if node["type"] != "object" {
		t.Fatalf("expected type=object, got %v", node["type"])
	}
}

// ---- ResolveSchema ----------------------------------------------------------

func TestResolveSchema_Depth0_LeavesRef(t *testing.T) {
	doc := openapi3Doc()
	node := map[string]any{"$ref": "#/components/schemas/Status"}
	result := ResolveSchema(doc, node, 0)
	if result["$ref"] != "Status" {
		t.Fatalf("depth=0 should keep short ref name, got %v", result)
	}
	if _, hasFrom := result["$from"]; hasFrom {
		t.Fatal("depth=0 should not set $from")
	}
}

func TestResolveSchema_Depth1_ExpandsRef(t *testing.T) {
	doc := openapi3Doc()
	node := map[string]any{"$ref": "#/components/schemas/Status"}
	result := ResolveSchema(doc, node, 1)
	if result["$ref"] != nil {
		t.Fatalf("depth=1 should not leave $ref, got %v", result["$ref"])
	}
	if result["$from"] != "Status" {
		t.Fatalf("expected $from=Status, got %v", result["$from"])
	}
	if result["type"] != "string" {
		t.Fatalf("expected type=string from inlined Status, got %v", result["type"])
	}
}

func TestResolveSchema_PropertiesRefExpanded(t *testing.T) {
	doc := openapi3Doc()
	node, _ := GetSchema(doc, "CreatePROption")
	result := ResolveSchema(doc, node, 1)
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}
	stateProp, ok := props["state"].(map[string]any)
	if !ok {
		t.Fatal("expected state property")
	}
	if stateProp["$from"] != "Status" {
		t.Fatalf("expected state.$from=Status, got %v", stateProp["$from"])
	}
}

func TestResolveSchema_ArrayItemsExpanded(t *testing.T) {
	doc := openapi3Doc()
	node, _ := GetSchema(doc, "CreatePROption")
	result := ResolveSchema(doc, node, 1)
	props, _ := result["properties"].(map[string]any)
	labelsProp, ok := props["labels"].(map[string]any)
	if !ok {
		t.Fatal("expected labels property")
	}
	items, ok := labelsProp["items"].(map[string]any)
	if !ok {
		t.Fatal("expected items in labels")
	}
	if items["$from"] != "Label" {
		t.Fatalf("expected items.$from=Label, got %v", items["$from"])
	}
}

func TestResolveSchema_Swagger2(t *testing.T) {
	doc := swagger2Doc()
	node := map[string]any{"$ref": "#/definitions/Label"}
	result := ResolveSchema(doc, node, 1)
	if result["$from"] != "Label" {
		t.Fatalf("expected $from=Label in swagger2, got %v", result["$from"])
	}
}

func TestResolveSchema_UnresolvableRef(t *testing.T) {
	doc := openapi3Doc()
	node := map[string]any{"$ref": "#/components/schemas/DoesNotExist"}
	result := ResolveSchema(doc, node, 1)
	if result["$error"] != "unresolved" {
		t.Fatalf("expected $error=unresolved, got %v", result)
	}
}

// ---- SchemaFields -----------------------------------------------------------

func TestSchemaFields_RequiredAndRef(t *testing.T) {
	doc := openapi3Doc()
	node, _ := GetSchema(doc, "CreatePROption")
	fields := SchemaFields(node)
	if len(fields) == 0 {
		t.Fatal("expected fields, got none")
	}
	fieldMap := make(map[string]Field, len(fields))
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	// title must be required.
	if !fieldMap["title"].Required {
		t.Fatal("expected title to be required")
	}
	if fieldMap["draft"].Required {
		t.Fatal("expected draft to NOT be required")
	}
	// state must have Ref set.
	if fieldMap["state"].Ref != "Status" {
		t.Fatalf("expected state.Ref=Status, got %q", fieldMap["state"].Ref)
	}
	// labels must have Items set.
	if fieldMap["labels"].Items != "Label" {
		t.Fatalf("expected labels.Items=Label, got %q", fieldMap["labels"].Items)
	}
	if fieldMap["labels"].Type != "array" {
		t.Fatalf("expected labels.Type=array, got %q", fieldMap["labels"].Type)
	}
}

func TestSchemaFields_Sorted(t *testing.T) {
	node, _ := GetSchema(openapi3Doc(), "CreatePROption")
	fields := SchemaFields(node)
	for i := 1; i < len(fields); i++ {
		if fields[i].Name < fields[i-1].Name {
			t.Fatalf("fields not sorted: %q before %q", fields[i-1].Name, fields[i].Name)
		}
	}
}

func TestSchemaFields_Swagger2(t *testing.T) {
	doc := swagger2Doc()
	node, _ := GetSchema(doc, "Label")
	fields := SchemaFields(node)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	for _, f := range fields {
		if !f.Required {
			t.Fatalf("expected field %q to be required", f.Name)
		}
	}
}

// ---- LookupFieldPath --------------------------------------------------------

func TestLookupFieldPath_Found_Simple(t *testing.T) {
	doc := openapi3Doc()
	result, err := LookupFieldPath(doc, "CreatePROption.title")
	if err != nil {
		t.Fatalf("LookupFieldPath: %v", err)
	}
	if !result.Exists {
		t.Fatal("expected Exists=true")
	}
	if result.Resolved != "CreatePROption.title" {
		t.Fatalf("expected resolved=CreatePROption.title, got %q", result.Resolved)
	}
	if result.Type != "string" {
		t.Fatalf("expected type=string, got %q", result.Type)
	}
}

func TestLookupFieldPath_Found_ThroughRef(t *testing.T) {
	// CreatePROption.state is a $ref to Status; Status.enum should be accessible
	// but we can't go "through" an enum. Let's test walking into labels (array) -> Label.id.
	doc := openapi3Doc()
	result, err := LookupFieldPath(doc, "CreatePROption.labels.id")
	if err != nil {
		t.Fatalf("LookupFieldPath through array+ref: %v", err)
	}
	if !result.Exists {
		t.Fatalf("expected Exists=true, got result=%+v", result)
	}
	if result.Resolved != "CreatePROption.labels.id" {
		t.Fatalf("unexpected resolved: %q", result.Resolved)
	}
	// Should have checked Label schema.
	foundLabel := false
	for _, c := range result.Checked {
		if c == "Label" {
			foundLabel = true
		}
	}
	if !foundLabel {
		t.Fatalf("expected Label in Checked, got %v", result.Checked)
	}
}

func TestLookupFieldPath_MissingField(t *testing.T) {
	doc := openapi3Doc()
	result, err := LookupFieldPath(doc, "CreatePROption.nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Exists {
		t.Fatal("expected Exists=false")
	}
	if result.Missing != "nonexistent" {
		t.Fatalf("expected Missing=nonexistent, got %q", result.Missing)
	}
	if len(result.Available) == 0 {
		t.Fatal("expected Available to be populated")
	}
}

func TestLookupFieldPath_MissingSchema_FuzzySuggestion(t *testing.T) {
	doc := openapi3Doc()
	// "createpr" should fuzzy-match "CreatePROption".
	result, err := LookupFieldPath(doc, "createpr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Exists {
		t.Fatal("expected Exists=false for missing schema")
	}
	if result.Missing != "createpr" {
		t.Fatalf("expected Missing=createpr, got %q", result.Missing)
	}
	foundSuggestion := false
	for _, a := range result.Available {
		if a == "CreatePROption" {
			foundSuggestion = true
		}
	}
	if !foundSuggestion {
		t.Fatalf("expected CreatePROption in suggestions, got %v", result.Available)
	}
}

func TestLookupFieldPath_NoSchemasContainer_ReturnsError(t *testing.T) {
	doc := map[string]any{"openapi": "3.0.0"}
	_, err := LookupFieldPath(doc, "Foo.bar")
	if err == nil {
		t.Fatal("expected error when no schemas container")
	}
}

func TestLookupFieldPath_Swagger2_ThroughRef(t *testing.T) {
	doc := swagger2Doc()
	result, err := LookupFieldPath(doc, "CreatePROption.labels.name")
	if err != nil {
		t.Fatalf("LookupFieldPath swagger2: %v", err)
	}
	if !result.Exists {
		t.Fatalf("expected Exists=true, got %+v", result)
	}
}

// ---- BuildSchemaIndex -------------------------------------------------------

func TestBuildSchemaIndex_Count(t *testing.T) {
	for _, doc := range []map[string]any{openapi3Doc(), swagger2Doc()} {
		idx := BuildSchemaIndex(doc)
		if len(idx) != 3 {
			t.Fatalf("expected 3 schema index entries, got %d", len(idx))
		}
	}
}

func TestBuildSchemaIndex_Sorted(t *testing.T) {
	idx := BuildSchemaIndex(openapi3Doc())
	for i := 1; i < len(idx); i++ {
		if idx[i].Name < idx[i-1].Name {
			t.Fatalf("schema index not sorted: %q before %q", idx[i-1].Name, idx[i].Name)
		}
	}
}

func TestBuildSchemaIndex_Properties(t *testing.T) {
	idx := BuildSchemaIndex(openapi3Doc())
	var pr *SchemaIndex
	for i := range idx {
		if idx[i].Name == "CreatePROption" {
			pr = &idx[i]
			break
		}
	}
	if pr == nil {
		t.Fatal("CreatePROption not found in index")
	}
	if len(pr.Properties) != 4 {
		t.Fatalf("expected 4 properties, got %d: %v", len(pr.Properties), pr.Properties)
	}
	if len(pr.Required) != 1 || pr.Required[0] != "title" {
		t.Fatalf("expected required=[title], got %v", pr.Required)
	}
}

func TestBuildSchemaIndex_NoContainer(t *testing.T) {
	idx := BuildSchemaIndex(map[string]any{})
	if idx != nil {
		t.Fatalf("expected nil, got %v", idx)
	}
}

// ---- regression tests for review fixes --------------------------------------

func TestRefName_Unescape(t *testing.T) {
	if got := RefName("#/components/schemas/Foo~1Bar"); got != "Foo/Bar" {
		t.Fatalf("expected decoded name Foo/Bar, got %q", got)
	}
	if got := RefName("#/definitions/Tilde~0Name"); got != "Tilde~Name" {
		t.Fatalf("expected decoded name Tilde~Name, got %q", got)
	}
}

func TestResolveRef_WholeDocument(t *testing.T) {
	doc := openapi3Doc()
	got, err := ResolveRef(doc, "#")
	if err != nil {
		t.Fatalf("ResolveRef(#): %v", err)
	}
	if got["openapi"] != "3.0.0" {
		t.Fatalf("expected whole document, got %v", got)
	}
}

func TestResolveSchema_NegativeDepthClamped(t *testing.T) {
	doc := openapi3Doc()
	node := map[string]any{"$ref": "#/components/schemas/Status"}
	result := ResolveSchema(doc, node, -5)
	if result["$ref"] != "Status" {
		t.Fatalf("negative depth should behave like depth 0, got %v", result)
	}
}

func TestLookupFieldPath_EmptyPath(t *testing.T) {
	for _, p := range []string{"", "   "} {
		if _, err := LookupFieldPath(openapi3Doc(), p); err == nil {
			t.Fatalf("expected error for empty path %q", p)
		}
	}
}

func TestLookupFieldPath_CycleTerminates(t *testing.T) {
	// Foo -> Bar -> Foo: a hostile cyclic spec must not hang `has`.
	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Foo": map[string]any{"$ref": "#/components/schemas/Bar"},
				"Bar": map[string]any{"$ref": "#/components/schemas/Foo"},
			},
		},
	}
	done := make(chan struct{})
	go func() {
		_, _ = LookupFieldPath(doc, "Foo.whatever")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("LookupFieldPath did not terminate on a cyclic ref")
	}
}
