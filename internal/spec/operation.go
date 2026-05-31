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

func stringsToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func stringsToUpper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
