package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MiguelAPerez/openstash/internal/store"
)

// seedHistory stores three versions of "api" whose changes are known:
// 1.0.0 -> 1.1.0 adds DELETE /items/{id}; 1.1.0 -> 1.10.0 changes the
// parameters of GET /items/{id} and removes GET /items.
func seedHistory(t *testing.T, st *store.Store) {
	t.Helper()
	v1 := testDoc("1.0.0")
	v2 := testDoc("1.1.0")
	v2["paths"].(map[string]any)["/items/{id}"].(map[string]any)["delete"] = map[string]any{"summary": "Delete item"}
	v3 := testDoc("1.10.0")
	paths := v3["paths"].(map[string]any)
	delete(paths, "/items")
	item := paths["/items/{id}"].(map[string]any)
	item["delete"] = map[string]any{"summary": "Delete item"}
	item["get"].(map[string]any)["parameters"] = []any{map[string]any{"name": "id", "in": "path", "required": true}}
	for _, d := range []map[string]any{v3, v1, v2} { // insertion order must not matter
		ver := d["info"].(map[string]any)["version"].(string)
		if _, _, err := st.Add("api", ver, "file.json", "", d); err != nil {
			t.Fatal(err)
		}
	}
}

func get(t *testing.T, srv *Server, url string, wantStatus int) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != wantStatus {
		t.Fatalf("GET %s status = %d, want %d: %s", url, rec.Code, wantStatus, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET %s: invalid JSON: %v", url, err)
	}
	return body
}

func TestTimeline(t *testing.T) {
	srv, st := testServer(t)
	seedHistory(t, st)

	body := get(t, srv, "/v1/specs/api/timeline", http.StatusOK)
	versions := body["versions"].([]any)
	var order []string
	for _, v := range versions {
		order = append(order, v.(map[string]any)["version"].(string))
	}
	if len(order) != 3 || order[0] != "1.0.0" || order[1] != "1.1.0" || order[2] != "1.10.0" {
		t.Fatalf("versions order = %v, want semver order", order)
	}
	first := versions[0].(map[string]any)
	if first["title"] != "Test" || first["operations"].(float64) != 2 {
		t.Fatalf("version summary = %v", first)
	}

	steps := body["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(steps))
	}
	ops := func(i int) map[string]any {
		s := steps[i].(map[string]any)
		return s["summary"].(map[string]any)["operations"].(map[string]any)
	}
	if ops(0)["added"].(float64) != 1 || ops(0)["changed"].(float64) != 0 {
		t.Fatalf("step 1 ops = %v", ops(0))
	}
	if ops(1)["removed"].(float64) != 1 || ops(1)["changed"].(float64) != 1 {
		t.Fatalf("step 2 ops = %v (parameter change must count as changed)", ops(1))
	}

	get(t, srv, "/v1/specs/missing/timeline", http.StatusNotFound)
	get(t, srv, "/v1/specs/a..b/timeline", http.StatusBadRequest)
}

func TestCompareEndpoint(t *testing.T) {
	srv, st := testServer(t)
	seedHistory(t, st)

	body := get(t, srv, "/v1/specs/api/compare?from=1.1.0&to=1.10.0", http.StatusOK)
	if body["baseline"].(map[string]any)["version"] != "1.1.0" || body["target"].(map[string]any)["version"] != "1.10.0" {
		t.Fatalf("sides = %v / %v", body["baseline"], body["target"])
	}
	changed := body["operations"].(map[string]any)["changed"].([]any)
	if len(changed) != 1 {
		t.Fatalf("changed = %v", changed)
	}
	c := changed[0].(map[string]any)
	if c["path"] != "/items/{id}" || c["method"] != "GET" {
		t.Fatalf("changed op = %v", c)
	}
	if _, ok := c["changes"].(map[string]any)["parameters"]; !ok {
		t.Fatalf("changes = %v, want parameters", c["changes"])
	}

	get(t, srv, "/v1/specs/api/compare?from=1.1.0", http.StatusBadRequest)
	get(t, srv, "/v1/specs/api/compare?from=1.1.0&to=9.9.9", http.StatusNotFound)
	get(t, srv, "/v1/specs/api/compare?from=..&to=1.1.0", http.StatusBadRequest)
	get(t, srv, "/v1/specs/api/compare?from=1.1.0&to=a/b", http.StatusBadRequest)
}

func TestCompareDefinition(t *testing.T) {
	srv, st := testServer(t)
	seedHistory(t, st)

	body := get(t, srv, "/v1/specs/api/compare/definition?from=1.1.0&to=1.10.0&method=get&path=/items/{id}", http.StatusOK)
	if body["item"].(map[string]any)["method"] != "GET" {
		t.Fatalf("item = %v", body["item"])
	}
	if _, ok := body["baseline"].(map[string]any)["parameters"]; ok {
		t.Fatalf("baseline should have no parameters: %v", body["baseline"])
	}
	if _, ok := body["target"].(map[string]any)["parameters"]; !ok {
		t.Fatalf("target should have parameters: %v", body["target"])
	}

	// Removed operation: present only in the baseline.
	body = get(t, srv, "/v1/specs/api/compare/definition?from=1.1.0&to=1.10.0&method=GET&path=/items", http.StatusOK)
	if body["baseline"] == nil || body["target"] != nil {
		t.Fatalf("removed op sides = %v / %v", body["baseline"], body["target"])
	}

	get(t, srv, "/v1/specs/api/compare/definition?from=1.1.0&to=1.10.0&method=GET&path=/nope", http.StatusNotFound)
	get(t, srv, "/v1/specs/api/compare/definition?from=1.1.0&to=1.10.0&schema=Nope", http.StatusNotFound)
	get(t, srv, "/v1/specs/api/compare/definition?from=1.1.0&to=1.10.0", http.StatusBadRequest)
}

func TestSchemaEndpoint(t *testing.T) {
	srv, st := testServer(t)
	doc := testDoc("1.0.0")
	doc["components"] = map[string]any{"schemas": map[string]any{
		"Item":  map[string]any{"type": "object", "properties": map[string]any{"owner": map[string]any{"$ref": "#/components/schemas/Owner"}}},
		"Owner": map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}},
	}}
	if _, _, err := st.Add("api", "1.0.0", "file.json", "", doc); err != nil {
		t.Fatal(err)
	}

	body := get(t, srv, "/v1/specs/api/versions/1.0.0/schemas/Item", http.StatusOK)
	owner := body["schema"].(map[string]any)["properties"].(map[string]any)["owner"].(map[string]any)
	if owner["$from"] != "Owner" {
		t.Fatalf("default depth should inline one $ref: %v", owner)
	}
	body = get(t, srv, "/v1/specs/api/versions/1.0.0/schemas/Item?depth=0", http.StatusOK)
	owner = body["schema"].(map[string]any)["properties"].(map[string]any)["owner"].(map[string]any)
	if owner["$ref"] != "Owner" {
		t.Fatalf("depth=0 should keep $ref: %v", owner)
	}
	get(t, srv, "/v1/specs/api/versions/1.0.0/schemas/Missing", http.StatusNotFound)
	get(t, srv, "/v1/specs/api/versions/9.9.9/schemas/Item", http.StatusNotFound)
}

// A stray directory or an unreadable version must not take the timeline down,
// and store errors must not leak filesystem paths.
func TestTimelineSkipsBrokenVersions(t *testing.T) {
	srv, st := testServer(t)
	seedHistory(t, st)
	keyDir := filepath.Join(st.Root, "specs", "api")
	if err := os.MkdirAll(filepath.Join(keyDir, "stray"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(keyDir, "1.1.0", "spec.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	body := get(t, srv, "/v1/specs/api/timeline", http.StatusOK)
	var order []string
	for _, v := range body["versions"].([]any) {
		order = append(order, v.(map[string]any)["version"].(string))
	}
	if strings.Join(order, ",") != "1.0.0,1.10.0" {
		t.Fatalf("versions = %v, want the two readable ones", order)
	}
	steps := body["steps"].([]any)
	if len(steps) != 1 || steps[0].(map[string]any)["from"] != "1.0.0" || steps[0].(map[string]any)["to"] != "1.10.0" {
		t.Fatalf("steps = %v", steps)
	}

	body = get(t, srv, "/v1/specs/api/versions/1.1.0", http.StatusInternalServerError)
	if msg, _ := body["error"].(string); strings.Contains(msg, st.Root) {
		t.Fatalf("500 leaked a store path: %q", msg)
	}
	get(t, srv, "/v1/specs/a%00b/timeline", http.StatusBadRequest)
}

func TestCompareReturnsEmptyArrays(t *testing.T) {
	srv, st := testServer(t)
	seedSpec(t, st, "api", "1.0.0")
	seedSpec(t, st, "api", "1.0.1")
	body := get(t, srv, "/v1/specs/api/compare?from=1.0.0&to=1.0.1", http.StatusOK)
	for _, group := range []string{"operations", "schemas"} {
		for _, kind := range []string{"added", "removed", "changed"} {
			if list, ok := body[group].(map[string]any)[kind].([]any); !ok || len(list) != 0 {
				t.Fatalf("%s.%s = %#v, want []", group, kind, body[group].(map[string]any)[kind])
			}
		}
	}
}

// The definition endpoint returns what CompareDetailed compared: shared $refs
// resolved, so a change behind a ref shows up in the line diff. OpenAPI 3.1
// boolean schemas are definitions too.
func TestCompareDefinitionResolvesRefs(t *testing.T) {
	srv, st := testServer(t)
	for _, v := range []struct{ ver, desc string }{{"1.0.0", "Items"}, {"1.1.0", "All items"}} {
		doc := testDoc(v.ver)
		doc["components"] = map[string]any{
			"responses": map[string]any{"Items": map[string]any{"description": v.desc}},
			"schemas":   map[string]any{"Anything": v.ver == "1.0.0"},
		}
		doc["paths"].(map[string]any)["/items"].(map[string]any)["get"].(map[string]any)["responses"] = map[string]any{
			"200": map[string]any{"$ref": "#/components/responses/Items"},
		}
		if _, _, err := st.Add("api", v.ver, "file.json", "", doc); err != nil {
			t.Fatal(err)
		}
	}

	cmp := get(t, srv, "/v1/specs/api/compare?from=1.0.0&to=1.1.0", http.StatusOK)
	changed := cmp["operations"].(map[string]any)["changed"].([]any)
	if len(changed) != 1 || changed[0].(map[string]any)["path"] != "/items" {
		t.Fatalf("changed ops = %v, want GET /items", changed)
	}

	body := get(t, srv, "/v1/specs/api/compare/definition?from=1.0.0&to=1.1.0&method=GET&path=/items", http.StatusOK)
	desc := func(side string) any {
		return body[side].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["description"]
	}
	if desc("baseline") != "Items" || desc("target") != "All items" {
		t.Fatalf("definition sides = %v / %v", body["baseline"], body["target"])
	}

	body = get(t, srv, "/v1/specs/api/compare/definition?from=1.0.0&to=1.1.0&schema=Anything", http.StatusOK)
	if body["baseline"] != true || body["target"] != false {
		t.Fatalf("boolean schema sides = %v / %v", body["baseline"], body["target"])
	}
}
