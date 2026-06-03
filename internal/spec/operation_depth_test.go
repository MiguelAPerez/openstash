package spec

import (
	"encoding/json"
	"testing"
)

// operationDepthDoc builds a minimal OpenAPI 3 document with:
//   - POST /items whose requestBody references "#/components/schemas/NewItem"
//   - 200 response references "#/components/schemas/Item"
//   - NewItem has a nested $ref to Address in one of its properties
//   - Item is a plain object with typed properties
func operationDepthDoc() map[string]any {
	return map[string]any{
		"openapi": "3.0.0",
		"info":    map[string]any{"title": "Test", "version": "1.0.0"},
		"components": map[string]any{
			"schemas": map[string]any{
				"Address": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"street": map[string]any{"type": "string"},
						"city":   map[string]any{"type": "string"},
					},
				},
				"NewItem": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":    map[string]any{"type": "string"},
						"address": map[string]any{"$ref": "#/components/schemas/Address"},
					},
				},
				"Item": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "integer"},
						"name": map[string]any{"type": "string"},
					},
				},
			},
		},
		"paths": map[string]any{
			"/items": map[string]any{
				"post": map[string]any{
					"operationId": "createItem",
					"summary":     "Create an item",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"$ref": "#/components/schemas/NewItem",
								},
							},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Created item",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"$ref": "#/components/schemas/Item",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// jsonEqual compares two values by marshaling both to JSON and comparing strings.
func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	ba, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal a: %v", err)
	}
	bb, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal b: %v", err)
	}
	return string(ba) == string(bb)
}

// TestGetOperationDepthZeroMatchesGetOperation verifies that depth=0 (default)
// produces the same output as GetOperation (shallow, refs not expanded).
func TestGetOperationDepthZeroMatchesGetOperation(t *testing.T) {
	doc := operationDepthDoc()

	shallow, err := GetOperation(doc, "/items", "POST")
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}

	depth0, err := GetOperationDepth(doc, "/items", "POST", 0)
	if err != nil {
		t.Fatalf("GetOperationDepth(0): %v", err)
	}

	if !jsonEqual(t, shallow, depth0) {
		a, _ := json.MarshalIndent(shallow, "", "  ")
		b, _ := json.MarshalIndent(depth0, "", "  ")
		t.Fatalf("depth=0 diverges from GetOperation:\nGetOperation:\n%s\nDepth0:\n%s", a, b)
	}

	// Confirm requestBody schema is shallow: has "$ref" key, no expanded properties.
	content, ok := shallow.RequestBody["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content in requestBody, got: %v", shallow.RequestBody)
	}
	body, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("expected application/json in content, got: %v", content)
	}
	schema, ok := body["schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema in body, got: %v", body)
	}
	if _, hasRef := schema["$ref"]; !hasRef {
		t.Fatalf("expected shallow $ref in requestBody schema, got: %v", schema)
	}
	if _, hasProps := schema["properties"]; hasProps {
		t.Fatalf("expected no 'properties' in shallow requestBody schema, got: %v", schema)
	}
}

// TestGetOperationDepthOneInlinesRefs verifies that depth=1 resolves top-level
// $refs in requestBody and response schemas, revealing typed properties and the
// $from marker. Nested $refs (e.g. address -> Address) stay as {$ref: "Address"}
// since the one depth hop is consumed resolving NewItem.
func TestGetOperationDepthOneInlinesRefs(t *testing.T) {
	doc := operationDepthDoc()

	op, err := GetOperationDepth(doc, "/items", "POST", 1)
	if err != nil {
		t.Fatalf("GetOperationDepth(1): %v", err)
	}

	// --- requestBody ---
	content, ok := op.RequestBody["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content in requestBody, got: %v", op.RequestBody)
	}
	body, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("expected application/json in content, got: %v", content)
	}
	schema, ok := body["schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema in body, got: %v", body)
	}

	// $from marker must appear (provenance of expanded ref).
	from, ok := schema["$from"].(string)
	if !ok || from != "NewItem" {
		t.Fatalf("expected $from=NewItem in requestBody schema, got: %v", schema)
	}

	// Properties must be a map (expanded), not a []string list (shallow).
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map in expanded requestBody schema, got type %T: %v", schema["properties"], schema["properties"])
	}

	// 'name' property should be {type: string}.
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatalf("expected name property to be map, got: %v", props["name"])
	}
	if nameProp["type"] != "string" {
		t.Fatalf("expected name.type=string, got: %v", nameProp["type"])
	}

	// 'address' is a nested $ref — at depth=1 it stays as {$ref: "Address"}
	// because the one depth hop was consumed resolving NewItem.
	addrProp, ok := props["address"].(map[string]any)
	if !ok {
		t.Fatalf("expected address property to be map, got: %v", props["address"])
	}
	if addrRef, ok := addrProp["$ref"].(string); !ok || addrRef != "Address" {
		t.Fatalf("expected address.$ref=Address (unexpanded at depth=1), got: %v", addrProp)
	}

	// --- responses ---
	resp200, ok := op.Responses["200"].(map[string]any)
	if !ok {
		t.Fatalf("expected 200 response map, got: %v", op.Responses["200"])
	}
	respContent, ok := resp200["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content in 200 response, got: %v", resp200)
	}
	respBody, ok := respContent["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("expected application/json in response content, got: %v", respContent)
	}
	respSchema, ok := respBody["schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema in response body, got: %v", respBody)
	}

	respFrom, ok := respSchema["$from"].(string)
	if !ok || respFrom != "Item" {
		t.Fatalf("expected $from=Item in response schema, got: %v", respSchema)
	}

	respProps, ok := respSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map in expanded response schema, got: %v", respSchema["properties"])
	}
	idProp, ok := respProps["id"].(map[string]any)
	if !ok {
		t.Fatalf("expected id property map, got: %v", respProps["id"])
	}
	if idProp["type"] != "integer" {
		t.Fatalf("expected id.type=integer, got: %v", idProp["type"])
	}
}

// TestGetOperationDepthTwoInlinesNestedRefs verifies that depth=2 also resolves
// the nested $ref (address -> Address), revealing street/city properties.
func TestGetOperationDepthTwoInlinesNestedRefs(t *testing.T) {
	doc := operationDepthDoc()

	op, err := GetOperationDepth(doc, "/items", "POST", 2)
	if err != nil {
		t.Fatalf("GetOperationDepth(2): %v", err)
	}

	content, ok := op.RequestBody["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content in requestBody, got: %v", op.RequestBody)
	}
	body, ok := content["application/json"].(map[string]any)
	if !ok {
		t.Fatalf("expected application/json in content, got: %v", content)
	}
	schema, ok := body["schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema in body, got: %v", body)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map at depth=2, got: %v", schema["properties"])
	}

	addrProp, ok := props["address"].(map[string]any)
	if !ok {
		t.Fatalf("expected address to be map at depth=2, got: %v", props["address"])
	}
	// At depth=2, address $ref should have been expanded — $from=Address present.
	addrFrom, ok := addrProp["$from"].(string)
	if !ok || addrFrom != "Address" {
		t.Fatalf("expected $from=Address in expanded address property, got: %v", addrProp)
	}
	addrProps, ok := addrProp["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected Address properties map, got: %v", addrProp["properties"])
	}
	streetProp, ok := addrProps["street"].(map[string]any)
	if !ok {
		t.Fatalf("expected street property map, got: %v", addrProps["street"])
	}
	if streetProp["type"] != "string" {
		t.Fatalf("expected street.type=string, got: %v", streetProp["type"])
	}
}

// TestGetOperationDepthNegativeMatchesGetOperation verifies negative depth
// delegates to GetOperation (shallow).
func TestGetOperationDepthNegativeMatchesGetOperation(t *testing.T) {
	doc := operationDepthDoc()

	shallow, err := GetOperation(doc, "/items", "POST")
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	depthNeg, err := GetOperationDepth(doc, "/items", "POST", -1)
	if err != nil {
		t.Fatalf("GetOperationDepth(-1): %v", err)
	}

	if !jsonEqual(t, shallow, depthNeg) {
		t.Fatalf("depth=-1 should equal GetOperation output")
	}
}
