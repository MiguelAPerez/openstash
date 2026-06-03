package search

import (
	"testing"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

func sampleSchemaIndex() []spec.SchemaIndex {
	return []spec.SchemaIndex{
		{
			Name:        "User",
			Type:        "object",
			Title:       "User account",
			Description: "Represents a user in the system",
			Properties:  []string{"id", "name", "email"},
			Required:    []string{"id", "name"},
		},
		{
			Name:        "Repository",
			Type:        "object",
			Title:       "Code repository",
			Description: "A git repository with branches and commits",
			Properties:  []string{"id", "name", "owner", "branches"},
		},
		{
			Name:        "Branch",
			Type:        "object",
			Title:       "Repository branch",
			Description: "A branch within a repository",
			Properties:  []string{"name", "commit", "protected"},
		},
		{
			Name:        "Commit",
			Type:        "object",
			Title:       "Git commit",
			Description: "A single commit in a repository",
			Properties:  []string{"sha", "message", "author", "timestamp"},
		},
	}
}

// TestSearchSchemasExactNameRanksFirst checks that an exact name match scores highest.
func TestSearchSchemasExactNameRanksFirst(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "user", 5)
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Schema.Name != "User" {
		t.Fatalf("top hit name = %q, want User", hits[0].Schema.Name)
	}
}

// TestSearchSchemasPropertyMatch checks that property-name matches are returned.
func TestSearchSchemasPropertyMatch(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "owner", 5)
	if len(hits) == 0 {
		t.Fatal("expected hits for property 'owner'")
	}
	found := false
	for _, h := range hits {
		if h.Schema.Name == "Repository" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Repository (which has 'owner' property) in results")
	}
}

// TestSearchSchemasDescriptionMatch checks that description text is matched.
func TestSearchSchemasDescriptionMatch(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "git repository", 5)
	if len(hits) == 0 {
		t.Fatal("expected hits for 'git repository'")
	}
	// Repository has "git repository" in description; Commit has "git" in title.
	// Repository should rank at top.
	if hits[0].Schema.Name != "Repository" {
		t.Fatalf("top hit = %q, want Repository", hits[0].Schema.Name)
	}
}

// TestSearchSchemasLimitRespected checks that limit is honored.
func TestSearchSchemasLimitRespected(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "", 2)
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits with limit=2, got %d", len(hits))
	}
}

// TestSearchSchemasEmptyQueryReturnsAll checks that empty query returns all entries (up to limit).
func TestSearchSchemasEmptyQueryReturnsAll(t *testing.T) {
	idx := sampleSchemaIndex()
	hits := SearchSchemas(idx, "", 10)
	if len(hits) != len(idx) {
		t.Fatalf("expected %d hits for empty query, got %d", len(idx), len(hits))
	}
	// All scores should be 1.
	for _, h := range hits {
		if h.Score != 1 {
			t.Fatalf("expected score 1 for empty query, got %d for %q", h.Score, h.Schema.Name)
		}
	}
}

// TestSearchSchemasNoMatch checks zero hits for an unrelated query.
func TestSearchSchemasNoMatch(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "billing invoice payment", 5)
	if len(hits) != 0 {
		t.Fatalf("expected no hits, got %d", len(hits))
	}
}

// TestSearchSchemasTitleMatch checks that title text is matched.
func TestSearchSchemasTitleMatch(t *testing.T) {
	hits := SearchSchemas(sampleSchemaIndex(), "account", 5)
	if len(hits) == 0 {
		t.Fatal("expected hits for 'account' (User title contains 'account')")
	}
	if hits[0].Schema.Name != "User" {
		t.Fatalf("top hit = %q, want User", hits[0].Schema.Name)
	}
}
