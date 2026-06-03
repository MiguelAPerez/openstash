package store

import (
	"testing"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func testDoc(version string) map[string]any {
	return map[string]any{
		"openapi": "3.0.0",
		"info":    map[string]any{"title": "Test", "version": version},
		"paths": map[string]any{
			"/items": map[string]any{
				"get": map[string]any{"summary": "List items"},
			},
		},
	}
}

func TestParseRef(t *testing.T) {
	tests := []struct {
		ref     string
		key     string
		version string
		wantErr bool
	}{
		{"gitea@1.0.0", "gitea", "1.0.0", false},
		{"gitea", "gitea", "", false},
		{"gitea@", "gitea", "", false},
		{"my.api@2.1.0", "my.api", "2.1.0", false},
		{"", "", "", true},
		{"@1.0.0", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got, err := ParseRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Key != tt.key || got.Version != tt.version {
				t.Fatalf("ParseRef(%q) = %+v, want key=%q version=%q", tt.ref, got, tt.key, tt.version)
			}
		})
	}
}

func TestLatestVersion(t *testing.T) {
	st := testStore(t)

	for _, version := range []string{"1.0.0", "1.10.0", "2.0.0", "1.2.0"} {
		if _, _, err := st.Add("api", version, "file.json", "", testDoc(version)); err != nil {
			t.Fatal(err)
		}
	}

	got, err := st.LatestVersion("api")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2.0.0" {
		t.Fatalf("LatestVersion = %q, want 2.0.0", got)
	}
}

func TestLatestVersionMissingKey(t *testing.T) {
	st := testStore(t)
	_, err := st.LatestVersion("missing")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestResolveRefUsesLatest(t *testing.T) {
	st := testStore(t)
	if _, _, err := st.Add("api", "1.0.0", "a.json", "", testDoc("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Add("api", "2.0.0", "b.json", "", testDoc("2.0.0")); err != nil {
		t.Fatal(err)
	}

	got, err := st.ResolveRef("api")
	if err != nil {
		t.Fatal(err)
	}
	if got != (spec.Ref{Key: "api", Version: "2.0.0"}) {
		t.Fatalf("ResolveRef = %+v, want api@2.0.0", got)
	}
}

func TestResolveRefKeepsExplicitVersion(t *testing.T) {
	st := testStore(t)
	if _, _, err := st.Add("api", "1.0.0", "a.json", "", testDoc("1.0.0")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Add("api", "2.0.0", "b.json", "", testDoc("2.0.0")); err != nil {
		t.Fatal(err)
	}

	got, err := st.ResolveRef("api@1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "1.0.0" {
		t.Fatalf("ResolveRef version = %q, want 1.0.0", got.Version)
	}
}

func TestAddListExists(t *testing.T) {
	st := testStore(t)
	meta, _, err := st.Add("gitea", "1.0.0", "/tmp/spec.json", "https://example/api", testDoc("1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Key != "gitea" || meta.Version != "1.0.0" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if !st.Exists("gitea", "1.0.0") {
		t.Fatal("expected spec to exist")
	}

	entries, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].SpecVersion != "1.0.0" {
		t.Fatalf("SpecVersion = %q, want 1.0.0", entries[0].SpecVersion)
	}
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"1.10.0", "1.2.0", 1},
		{"beta", "alpha", 1},
		{"1.0.0", "not-semver", 1},
	}

	for _, tt := range tests {
		if got := compareVersion(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
