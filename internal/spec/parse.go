package spec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// LoadFrom reads and normalizes an OpenAPI document from a URL or file path.
func LoadFrom(source string) (map[string]any, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("fetch %s: %s", source, resp.Status)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, err
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty document from %s", source)
	}

	trim := strings.TrimSpace(string(data))
	var doc map[string]any
	if strings.HasPrefix(trim, "{") {
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
	} else {
		var yamlDoc any
		if err := unmarshalYAML(data, &yamlDoc); err != nil {
			return nil, fmt.Errorf("parse yaml: %w", err)
		}
		doc, err = yamlToMap(yamlDoc)
		if err != nil {
			return nil, err
		}
	}

	if _, ok := doc["openapi"]; !ok {
		if _, ok := doc["swagger"]; !ok {
			return nil, fmt.Errorf("not an OpenAPI document (missing openapi/swagger)")
		}
	}
	return doc, nil
}

func yamlToMap(v any) (map[string]any, error) {
	switch t := v.(type) {
	case map[string]any:
		return t, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("yaml map key is not string")
			}
			normalized, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			out[ks] = normalized
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected yaml mapping at root")
	}
}

func normalizeYAMLValue(v any) (any, error) {
	switch t := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("yaml map key is not string")
			}
			nv, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			m[ks] = nv
		}
		return m, nil
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			nv, err := normalizeYAMLValue(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

// InfoVersion returns info.version from the spec if present.
func InfoVersion(doc map[string]any) string {
	info, _ := doc["info"].(map[string]any)
	if info == nil {
		return ""
	}
	v, _ := info["version"].(string)
	return v
}
