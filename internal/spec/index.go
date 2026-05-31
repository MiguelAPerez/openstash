package spec

import "strings"

var httpMethods = []string{
	"get", "post", "put", "patch", "delete", "head", "options", "trace",
}

// BuildIndex flattens paths.* operations for search.
func BuildIndex(doc map[string]any) []OperationIndex {
	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return nil
	}

	var out []OperationIndex
	for path, raw := range paths {
		pathItem, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, method := range httpMethods {
			opRaw, ok := pathItem[method]
			if !ok {
				continue
			}
			op, ok := opRaw.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, OperationIndex{
				Method:      strings.ToUpper(method),
				Path:        path,
				OperationID: strField(op, "operationId"),
				Summary:     strField(op, "summary"),
				Description: strField(op, "description"),
				Tags:        stringSliceField(op, "tags"),
			})
		}
	}
	return out
}

func strField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func stringSliceField(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
