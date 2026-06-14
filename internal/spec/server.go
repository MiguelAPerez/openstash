package spec

import (
	"net/url"
	"strings"
)

// ServerBase extracts the server host and path prefix from the spec.
//
// For OAS3 it reads servers[0].url.
// For Swagger 2.0 it uses the host / schemes / basePath fields.
// Returns ("", "") when no server info is present.
func ServerBase(doc map[string]any) (hostPart, pathPart string) {
	// OAS3
	if servers, ok := doc["servers"].([]any); ok && len(servers) > 0 {
		if s, ok := servers[0].(map[string]any); ok {
			if rawURL, _ := s["url"].(string); rawURL != "" {
				return splitServerURL(rawURL)
			}
		}
	}

	basePath, _ := doc["basePath"].(string)

	// Swagger 2.0 with absolute basePath (rare but valid)
	if strings.HasPrefix(basePath, "http://") || strings.HasPrefix(basePath, "https://") {
		return splitServerURL(basePath)
	}

	// Swagger 2.0 with separate host + schemes + basePath
	if h, _ := doc["host"].(string); h != "" {
		scheme := "https"
		if schemes, _ := doc["schemes"].([]any); len(schemes) > 0 {
			if s, _ := schemes[0].(string); s != "" {
				scheme = s
			}
		}
		return scheme + "://" + h, strings.TrimRight(basePath, "/")
	}

	// Relative basePath only — caller must supply the host
	return "", strings.TrimRight(basePath, "/")
}

func splitServerURL(raw string) (string, string) {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() {
		return "", strings.TrimRight(raw, "/")
	}
	return parsed.Scheme + "://" + parsed.Host, strings.TrimRight(parsed.Path, "/")
}
