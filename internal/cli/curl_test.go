package cli

import (
	"reflect"
	"testing"
)

func TestParseParams(t *testing.T) {
	cases := []struct {
		raw     []string
		want    map[string]string
		wantErr bool
	}{
		{[]string{"owner=alice", "repo=myrepo"}, map[string]string{"owner": "alice", "repo": "myrepo"}, false},
		{[]string{"body=hello=world"}, map[string]string{"body": "hello=world"}, false}, // value may contain =
		{[]string{"noequals"}, nil, true},
		{[]string{"=nope"}, nil, true},
		{nil, map[string]string{}, false},
	}
	for _, tc := range cases {
		got, err := parseParams(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseParams(%v): expected error, got nil", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseParams(%v): unexpected error: %v", tc.raw, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseParams(%v) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestPathParamNames(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{"/repos/{owner}/{repo}/issues", []string{"owner", "repo"}},
		{"/users/{username}", []string{"username"}},
		{"/repos", nil},
		{"/admin/runners/{runner_id}/labels/{id}", []string{"runner_id", "id"}},
	}
	for _, tc := range cases {
		got := pathParamNames(tc.path)
		if len(got) != len(tc.want) {
			t.Errorf("pathParamNames(%q) = %v, want %v", tc.path, got, tc.want)
			continue
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("pathParamNames(%q)[%d] = %q, want %q", tc.path, i, got[i], tc.want[i])
			}
		}
	}
}

func TestSpecQueryParamNames(t *testing.T) {
	params := []any{
		map[string]any{"name": "page", "in": "query"},
		map[string]any{"name": "limit", "in": "query"},
		map[string]any{"name": "token", "in": "header"},
		map[string]any{"name": "owner", "in": "path"},
		"not a map",
	}
	got := specQueryParamNames(params)
	want := map[string]bool{"page": true, "limit": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("specQueryParamNames() = %v, want %v", got, want)
	}
}
