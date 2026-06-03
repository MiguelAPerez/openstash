package spec

import (
	"fmt"
	"strings"
)

// OperationDetail is the agent-oriented slice of one operation.
type OperationDetail struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	OperationID string         `json:"operationId,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Parameters  []any          `json:"parameters,omitempty"`
	RequestBody map[string]any `json:"requestBody,omitempty"`
	Responses   map[string]any `json:"responses,omitempty"`
}

// GetOperation returns detail for a single path + method.
func GetOperation(doc map[string]any, path, method string) (*OperationDetail, error) {
	method = stringsToLower(method)
	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return nil, fmt.Errorf("spec has no paths")
	}
	pathItem, ok := paths[path].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path not found: %s", path)
	}
	opRaw, ok := pathItem[method]
	if !ok {
		return nil, fmt.Errorf("method %s not found on %s", stringsToUpper(method), path)
	}
	op, ok := opRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid operation at %s %s", method, path)
	}
	return operationFromMap(stringsToUpper(method), path, op), nil
}

func operationFromMap(method, path string, op map[string]any) *OperationDetail {
	d := &OperationDetail{
		Method:      method,
		Path:        path,
		OperationID: strField(op, "operationId"),
		Summary:     strField(op, "summary"),
		Description: strField(op, "description"),
		Tags:        stringSliceField(op, "tags"),
	}
	if params, ok := op["parameters"].([]any); ok {
		d.Parameters = params
	}
	if rb, ok := op["requestBody"].(map[string]any); ok {
		d.RequestBody = shallowRequestBody(rb)
	}
	if resp, ok := op["responses"].(map[string]any); ok {
		d.Responses = shallowResponses(resp)
	}
	return d
}

func shallowRequestBody(rb map[string]any) map[string]any {
	out := map[string]any{}
	if req, ok := rb["required"].(bool); ok {
		out["required"] = req
	}
	if desc, ok := rb["description"].(string); ok {
		out["description"] = desc
	}
	if content, ok := rb["content"].(map[string]any); ok {
		shallow := make(map[string]any, len(content))
		for ct, body := range content {
			if bm, ok := body.(map[string]any); ok {
				if schema, ok := bm["schema"].(map[string]any); ok {
					shallow[ct] = map[string]any{"schema": shallowSchema(schema)}
				} else {
					shallow[ct] = body
				}
			}
		}
		out["content"] = shallow
	}
	return out
}

func shallowResponses(resp map[string]any) map[string]any {
	out := make(map[string]any, len(resp))
	for code, raw := range resp {
		rm, ok := raw.(map[string]any)
		if !ok {
			out[code] = raw
			continue
		}
		entry := map[string]any{}
		if desc, ok := rm["description"].(string); ok {
			entry["description"] = desc
		}
		if content, ok := rm["content"].(map[string]any); ok {
			shallow := make(map[string]any, len(content))
			for ct, body := range content {
				if bm, ok := body.(map[string]any); ok {
					if schema, ok := bm["schema"].(map[string]any); ok {
						shallow[ct] = map[string]any{"schema": shallowSchema(schema)}
					}
				}
			}
			if len(shallow) > 0 {
				entry["content"] = shallow
			}
		}
		out[code] = entry
	}
	return out
}

func shallowSchema(schema map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"type", "format", "enum", "required", "description", "$ref"} {
		if v, ok := schema[k]; ok {
			out[k] = v
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		names := make([]string, 0, len(props))
		for name := range props {
			names = append(names, name)
		}
		out["properties"] = names
	}
	if items, ok := schema["items"].(map[string]any); ok {
		out["items"] = shallowSchema(items)
	}
	return out
}

// GetOperationDepth returns detail for a single path + method, with schema
// refs inlined to the given depth. When depth <= 0 it delegates to GetOperation
// (preserving the existing shallow behavior). When depth > 0 it resolves
// $ref nodes in request body and response content schemas via ResolveSchema.
func GetOperationDepth(doc map[string]any, path, method string, depth int) (*OperationDetail, error) {
	if depth <= 0 {
		return GetOperation(doc, path, method)
	}
	method = stringsToLower(method)
	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return nil, fmt.Errorf("spec has no paths")
	}
	pathItem, ok := paths[path].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("path not found: %s", path)
	}
	opRaw, ok := pathItem[method]
	if !ok {
		return nil, fmt.Errorf("method %s not found on %s", stringsToUpper(method), path)
	}
	op, ok := opRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid operation at %s %s", method, path)
	}

	d := &OperationDetail{
		Method:      stringsToUpper(method),
		Path:        path,
		OperationID: strField(op, "operationId"),
		Summary:     strField(op, "summary"),
		Description: strField(op, "description"),
		Tags:        stringSliceField(op, "tags"),
	}
	if params, ok := op["parameters"].([]any); ok {
		d.Parameters = expandedParameters(doc, params, depth)
	}
	if rb, ok := op["requestBody"].(map[string]any); ok {
		d.RequestBody = expandedRequestBody(doc, rb, depth)
	}
	if resp, ok := op["responses"].(map[string]any); ok {
		d.Responses = expandedResponses(doc, resp, depth)
	}
	return d, nil
}

// expandedParameters inlines $ref schemas attached to parameters. This covers
// Swagger 2.0 body parameters (in: body, schema: {$ref}) as well as OpenAPI 3.0
// parameter schemas. Parameters without a schema are passed through unchanged.
func expandedParameters(doc map[string]any, params []any, depth int) []any {
	out := make([]any, len(params))
	for i, p := range params {
		pm, ok := p.(map[string]any)
		if !ok {
			out[i] = p
			continue
		}
		schema, ok := pm["schema"].(map[string]any)
		if !ok {
			out[i] = pm
			continue
		}
		np := make(map[string]any, len(pm))
		for k, v := range pm {
			np[k] = v
		}
		np["schema"] = ResolveSchema(doc, schema, depth)
		out[i] = np
	}
	return out
}

func expandedRequestBody(doc map[string]any, rb map[string]any, depth int) map[string]any {
	out := map[string]any{}
	if req, ok := rb["required"].(bool); ok {
		out["required"] = req
	}
	if desc, ok := rb["description"].(string); ok {
		out["description"] = desc
	}
	if content, ok := rb["content"].(map[string]any); ok {
		expanded := make(map[string]any, len(content))
		for ct, body := range content {
			if bm, ok := body.(map[string]any); ok {
				if schema, ok := bm["schema"].(map[string]any); ok {
					expanded[ct] = map[string]any{"schema": ResolveSchema(doc, schema, depth)}
				} else {
					expanded[ct] = body
				}
			}
		}
		out["content"] = expanded
	}
	return out
}

func expandedResponses(doc map[string]any, resp map[string]any, depth int) map[string]any {
	out := make(map[string]any, len(resp))
	for code, raw := range resp {
		rm, ok := raw.(map[string]any)
		if !ok {
			out[code] = raw
			continue
		}
		// Swagger 2.0 named response: the response is itself a $ref into
		// #/responses/... — resolve the whole response node.
		if _, ok := rm["$ref"].(string); ok {
			out[code] = ResolveSchema(doc, rm, depth)
			continue
		}
		entry := map[string]any{}
		if desc, ok := rm["description"].(string); ok {
			entry["description"] = desc
		}
		// Swagger 2.0: response schema lives directly on the response.
		if schema, ok := rm["schema"].(map[string]any); ok {
			entry["schema"] = ResolveSchema(doc, schema, depth)
		}
		// OpenAPI 3.0: response schema lives under content[mediaType].schema.
		if content, ok := rm["content"].(map[string]any); ok {
			expanded := make(map[string]any, len(content))
			for ct, body := range content {
				if bm, ok := body.(map[string]any); ok {
					if schema, ok := bm["schema"].(map[string]any); ok {
						expanded[ct] = map[string]any{"schema": ResolveSchema(doc, schema, depth)}
					}
				}
			}
			if len(expanded) > 0 {
				entry["content"] = expanded
			}
		}
		out[code] = entry
	}
	return out
}

func stringsToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func stringsToUpper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
