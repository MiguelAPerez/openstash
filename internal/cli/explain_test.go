package cli

import (
	"strings"
	"testing"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

// --- topTags ---

func TestTopTagsEmpty(t *testing.T) {
	result := topTags(nil, 8)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %v", result)
	}
}

func TestTopTagsOrdering(t *testing.T) {
	index := []spec.OperationIndex{
		{Tags: []string{"pets", "users"}},
		{Tags: []string{"pets"}},
		{Tags: []string{"orders", "users"}},
		{Tags: []string{"pets"}},
	}
	// pets: 3, users: 2, orders: 1
	got := topTags(index, 8)
	if len(got) != 3 {
		t.Fatalf("expected 3 tags, got %d: %v", len(got), got)
	}
	if got[0] != "pets" {
		t.Errorf("expected first tag=pets, got %q", got[0])
	}
	if got[1] != "users" {
		t.Errorf("expected second tag=users, got %q", got[1])
	}
	if got[2] != "orders" {
		t.Errorf("expected third tag=orders, got %q", got[2])
	}
}

func TestTopTagsTieBreaksAlphabetically(t *testing.T) {
	index := []spec.OperationIndex{
		{Tags: []string{"zebra", "alpha"}},
		{Tags: []string{"zebra", "alpha"}},
	}
	// both count = 2; alpha < zebra
	got := topTags(index, 8)
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(got), got)
	}
	if got[0] != "alpha" {
		t.Errorf("expected alpha first (tie-break), got %q", got[0])
	}
}

func TestTopTagsCap(t *testing.T) {
	index := []spec.OperationIndex{
		{Tags: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
	}
	got := topTags(index, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 tags (cap=3), got %d: %v", len(got), got)
	}
}

// --- buildRecipes ---

func TestBuildRecipesEmpty(t *testing.T) {
	// With no data, should still return some minimal recipes without panicking.
	recipes := buildRecipes("myapi", nil, nil, nil)
	if len(recipes) == 0 {
		t.Fatal("expected at least one recipe even with empty input")
	}
	for _, r := range recipes {
		if !strings.HasPrefix(r, "openstash ") {
			t.Errorf("recipe should start with 'openstash ', got: %q", r)
		}
	}
}

func TestBuildRecipesReferencesRealData(t *testing.T) {
	index := []spec.OperationIndex{
		{Method: "GET", Path: "/pets", Tags: []string{"pets"}},
		{Method: "POST", Path: "/pets", Tags: []string{"pets"}},
	}
	schemaNames := []string{"Error", "Pet"}
	schemaFields := map[string][]string{
		"Pet":   {"id", "name"},
		"Error": {"code", "message"},
	}

	recipes := buildRecipes("petstore", index, schemaNames, schemaFields)
	if len(recipes) < 5 {
		t.Fatalf("expected >=5 recipes, got %d: %v", len(recipes), recipes)
	}

	joined := strings.Join(recipes, "\n")

	// Should reference the real tag.
	if !strings.Contains(joined, "pets") {
		t.Error("recipes should reference tag 'pets'")
	}
	// Should reference a real path.
	if !strings.Contains(joined, "/pets") {
		t.Error("recipes should reference path '/pets'")
	}
	// Should reference a real schema.
	if !strings.Contains(joined, "Error") && !strings.Contains(joined, "Pet") {
		t.Error("recipes should reference a real schema name")
	}
	// Should include --expand variant.
	if !strings.Contains(joined, "--expand") {
		t.Error("recipes should include --expand variant")
	}
	// Should include --fields variant.
	if !strings.Contains(joined, "--fields") {
		t.Error("recipes should include --fields variant")
	}
	// Should include has command with real field.
	if !strings.Contains(joined, "openstash has") {
		t.Error("recipes should include 'openstash has' command")
	}
	// Should include gather command.
	if !strings.Contains(joined, "openstash gather") {
		t.Error("recipes should include 'openstash gather' command")
	}
}

func TestBuildRecipesMethodVariety(t *testing.T) {
	index := []spec.OperationIndex{
		{Method: "DELETE", Path: "/pets/{id}", Tags: []string{"pets"}},
		{Method: "GET", Path: "/pets", Tags: []string{"pets"}},
		{Method: "POST", Path: "/pets", Tags: []string{"pets"}},
	}
	recipes := buildRecipes("myapi", index, nil, nil)
	joined := strings.Join(recipes, "\n")
	// Two different methods should appear as show commands.
	hasGet := strings.Contains(joined, "--method GET")
	hasDelete := strings.Contains(joined, "--method DELETE")
	hasPost := strings.Contains(joined, "--method POST")
	methodCount := 0
	if hasGet {
		methodCount++
	}
	if hasDelete {
		methodCount++
	}
	if hasPost {
		methodCount++
	}
	if methodCount < 2 {
		t.Errorf("expected at least 2 different methods in show recipes, got: %s", joined)
	}
}

// --- entryHints ---

func TestEntryHintsEmpty(t *testing.T) {
	hints := entryHints("myapi", nil, nil)
	if len(hints) == 0 {
		t.Fatal("expected at least one hint even with empty input")
	}
	for _, h := range hints {
		if !strings.HasPrefix(h, "openstash ") {
			t.Errorf("hint should start with 'openstash ', got: %q", h)
		}
	}
}

func TestEntryHintsRealData(t *testing.T) {
	index := []spec.OperationIndex{
		{Method: "GET", Path: "/users", Tags: []string{"users"}},
		{Method: "POST", Path: "/users", Tags: []string{"users"}},
	}
	schemaNames := []string{"User"}

	hints := entryHints("myapi", index, schemaNames)
	if len(hints) != 3 {
		t.Fatalf("expected 3 hints, got %d: %v", len(hints), hints)
	}
	joined := strings.Join(hints, "\n")
	if !strings.Contains(joined, "users") {
		t.Error("hints should reference real tag 'users'")
	}
	if !strings.Contains(joined, "/users") {
		t.Error("hints should reference real path '/users'")
	}
	if !strings.Contains(joined, "User") {
		t.Error("hints should reference real schema 'User'")
	}
	if !strings.Contains(joined, "--fields") {
		t.Error("hints should include --fields")
	}
}

func TestEntryHintsNoSchemas(t *testing.T) {
	index := []spec.OperationIndex{
		{Method: "GET", Path: "/items", Tags: []string{"items"}},
	}
	hints := entryHints("myapi", index, nil)
	// Should have search + show, no schema hint.
	if len(hints) != 2 {
		t.Fatalf("expected 2 hints (no schemas), got %d: %v", len(hints), hints)
	}
}
