package cli

import (
	"reflect"
	"testing"
)

func TestParseCompareSectionsDefault(t *testing.T) {
	got, err := parseCompareSections(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"operations": true, "schemas": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseCompareSectionsSingle(t *testing.T) {
	got, err := parseCompareSections([]string{"operations"})
	if err != nil {
		t.Fatal(err)
	}
	if !got["operations"] || got["schemas"] {
		t.Fatalf("got %v", got)
	}
}

func TestParseCompareSectionsInvalid(t *testing.T) {
	_, err := parseCompareSections([]string{"paths"})
	if err == nil {
		t.Fatal("expected error for invalid section")
	}
}

func TestLimitSlice(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	shown, total := limitSlice(items, 2)
	if total != 5 || len(shown) != 2 {
		t.Fatalf("limitSlice = %v total=%d, want 2 shown total 5", shown, total)
	}

	shown, total = limitSlice(items, 0)
	if total != 5 || len(shown) != 5 {
		t.Fatalf("unlimited = %v total=%d", shown, total)
	}
}
