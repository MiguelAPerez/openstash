package search

import (
	"testing"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

func sampleIndex() []spec.OperationIndex {
	return []spec.OperationIndex{
		{Method: "GET", Path: "/user/repos", OperationID: "listUserRepos", Summary: "List user repositories", Tags: []string{"repos"}},
		{Method: "POST", Path: "/user/repos", OperationID: "createUserRepo", Summary: "Create a repository", Tags: []string{"repos"}},
		{Method: "GET", Path: "/users", OperationID: "listUsers", Summary: "List users", Tags: []string{"users"}},
	}
}

func TestQueryMatchesPath(t *testing.T) {
	hits := Query(sampleIndex(), "user repos", 5, "", "")
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Operation.Path != "/user/repos" {
		t.Fatalf("top hit path = %q, want /user/repos", hits[0].Operation.Path)
	}
}

func TestQueryMethodFilter(t *testing.T) {
	hits := Query(sampleIndex(), "", 5, "", "POST")
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Operation.Method != "POST" {
		t.Fatalf("method = %q, want POST", hits[0].Operation.Method)
	}
}

func TestQueryPathPrefixFilter(t *testing.T) {
	hits := Query(sampleIndex(), "", 5, "/user/", "")
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
}

func TestQueryLimit(t *testing.T) {
	hits := Query(sampleIndex(), "", 5, "", "")
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}

	hits = Query(sampleIndex(), "", 1, "", "")
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit with limit, got %d", len(hits))
	}
}

func TestQueryNoMatch(t *testing.T) {
	hits := Query(sampleIndex(), "subscriptions billing", 5, "", "")
	if len(hits) != 0 {
		t.Fatalf("expected no hits, got %d", len(hits))
	}
}
