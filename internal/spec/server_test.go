package spec

import "testing"

func TestServerBase(t *testing.T) {
	cases := []struct {
		name     string
		doc      map[string]any
		wantHost string
		wantPath string
	}{
		{
			name: "OAS3 full URL no path",
			doc: map[string]any{
				"openapi": "3.0.0",
				"servers": []any{map[string]any{"url": "https://api.cursor.com"}},
			},
			wantHost: "https://api.cursor.com",
			wantPath: "",
		},
		{
			name: "OAS3 full URL with path",
			doc: map[string]any{
				"openapi": "3.0.0",
				"servers": []any{map[string]any{"url": "https://api.example.com/v2"}},
			},
			wantHost: "https://api.example.com",
			wantPath: "/v2",
		},
		{
			name: "Swagger 2 relative basePath no host",
			doc: map[string]any{
				"swagger":  "2.0",
				"basePath": "/api/v1",
				"schemes":  []any{"https", "http"},
			},
			wantHost: "",
			wantPath: "/api/v1",
		},
		{
			name: "Swagger 2 with host and basePath",
			doc: map[string]any{
				"swagger":  "2.0",
				"host":     "api.example.com",
				"basePath": "/v2",
				"schemes":  []any{"https"},
			},
			wantHost: "https://api.example.com",
			wantPath: "/v2",
		},
		{
			name: "Swagger 2 absolute basePath",
			doc: map[string]any{
				"swagger":  "2.0",
				"basePath": "https://api.example.com/v1",
			},
			wantHost: "https://api.example.com",
			wantPath: "/v1",
		},
		{
			name:     "empty doc",
			doc:      map[string]any{},
			wantHost: "",
			wantPath: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotHost, gotPath := ServerBase(tc.doc)
			if gotHost != tc.wantHost || gotPath != tc.wantPath {
				t.Errorf("ServerBase() = (%q, %q), want (%q, %q)", gotHost, gotPath, tc.wantHost, tc.wantPath)
			}
		})
	}
}
