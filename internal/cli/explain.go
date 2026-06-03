package cli

import (
	"fmt"
	"sort"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

// tagCount holds a tag and how many operations reference it.
type tagCount struct {
	Tag   string
	Count int
}

// topTags counts operation tags across the index, sorts by count desc then name asc,
// and returns up to cap entries.
func topTags(index []spec.OperationIndex, cap int) []string {
	counts := map[string]int{}
	for _, op := range index {
		for _, t := range op.Tags {
			counts[t]++
		}
	}
	tags := make([]tagCount, 0, len(counts))
	for t, c := range counts {
		tags = append(tags, tagCount{Tag: t, Count: c})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Tag < tags[j].Tag
	})
	result := make([]string, 0, cap)
	for i, tc := range tags {
		if i >= cap {
			break
		}
		result = append(result, tc.Tag)
	}
	return result
}

// buildRecipes returns 5-10 deterministic ready-to-paste command strings using
// real names from the spec. It gracefully omits recipes when data is absent.
func buildRecipes(key string, index []spec.OperationIndex, schemaNames []string, schemaFields map[string][]string) []string {
	var recipes []string

	// --- search by tag ---
	tags := topTags(index, 8)
	if len(tags) > 0 {
		recipes = append(recipes, fmt.Sprintf("openstash search %s %q", key, tags[0]))
	} else {
		recipes = append(recipes, fmt.Sprintf("openstash search %s", key))
	}

	// --- show a couple representative ops (prefer method variety) ---
	// Sort index deterministically: method asc, path asc.
	sorted := make([]spec.OperationIndex, len(index))
	copy(sorted, index)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Method != sorted[j].Method {
			return sorted[i].Method < sorted[j].Method
		}
		return sorted[i].Path < sorted[j].Path
	})

	// Pick up to 2 ops with different methods.
	var showOps []spec.OperationIndex
	seenMethods := map[string]bool{}
	for _, op := range sorted {
		if len(showOps) >= 2 {
			break
		}
		if !seenMethods[op.Method] {
			showOps = append(showOps, op)
			seenMethods[op.Method] = true
		}
	}
	// If we didn't get 2 distinct methods, just take the first 2.
	if len(showOps) < 2 && len(sorted) >= 2 {
		showOps = sorted[:2]
	} else if len(showOps) == 0 && len(sorted) == 1 {
		showOps = sorted[:1]
	}

	for i, op := range showOps {
		recipes = append(recipes, fmt.Sprintf("openstash show %s --method %s --path %q", key, op.Method, op.Path))
		// Add --expand variant for the first op.
		if i == 0 {
			recipes = append(recipes, fmt.Sprintf("openstash show %s --method %s --path %q --expand", key, op.Method, op.Path))
		}
	}

	// --- schema --fields ---
	if len(schemaNames) > 0 {
		recipes = append(recipes, fmt.Sprintf("openstash schema %s %s --fields", key, schemaNames[0]))
	}

	// --- has SchemaName.field ---
	if len(schemaNames) > 0 {
		sn := schemaNames[0]
		if fields, ok := schemaFields[sn]; ok && len(fields) > 0 {
			recipes = append(recipes, fmt.Sprintf("openstash has %s %s.%s", key, sn, fields[0]))
		} else if len(schemaNames) > 1 {
			// Try the second schema.
			sn2 := schemaNames[1]
			if fields2, ok := schemaFields[sn2]; ok && len(fields2) > 0 {
				recipes = append(recipes, fmt.Sprintf("openstash has %s %s.%s", key, sn2, fields2[0]))
			}
		}
	}

	// --- gather keyword ---
	if len(tags) > 0 {
		recipes = append(recipes, fmt.Sprintf("openstash gather %s %q", key, tags[0]))
	} else if len(index) > 0 {
		recipes = append(recipes, fmt.Sprintf("openstash gather %s", key))
	}

	return recipes
}

// entryHints returns ~3 command strings for a list entry using real spec data.
func entryHints(key string, index []spec.OperationIndex, schemaNames []string) []string {
	var hints []string

	// search hint.
	tags := topTags(index, 1)
	if len(tags) > 0 {
		hints = append(hints, fmt.Sprintf("openstash search %s %q", key, tags[0]))
	} else {
		hints = append(hints, fmt.Sprintf("openstash search %s", key))
	}

	// show hint — pick the first sorted op.
	sorted := make([]spec.OperationIndex, len(index))
	copy(sorted, index)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Method != sorted[j].Method {
			return sorted[i].Method < sorted[j].Method
		}
		return sorted[i].Path < sorted[j].Path
	})
	if len(sorted) > 0 {
		op := sorted[0]
		hints = append(hints, fmt.Sprintf("openstash show %s --method %s --path %q", key, op.Method, op.Path))
	}

	// schema --fields hint.
	if len(schemaNames) > 0 {
		hints = append(hints, fmt.Sprintf("openstash schema %s %s --fields", key, schemaNames[0]))
	}

	return hints
}

func newExplain() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <key[@version]>",
		Short: "Quick-start for one spec, auto-derived from its actual contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, key, version, doc, index, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			// Load or build schema index.
			schemaIdx, _ := st.LoadSchemaIndex(key, version)
			if schemaIdx == nil && doc != nil {
				schemaIdx = spec.BuildSchemaIndex(doc)
			}

			// Collect sorted schema names.
			schemaNames := make([]string, 0, len(schemaIdx))
			for _, s := range schemaIdx {
				schemaNames = append(schemaNames, s.Name)
			}
			sort.Strings(schemaNames)

			// Build schema→fields map for recipe generation.
			schemaFieldsMap := map[string][]string{}
			for _, s := range schemaIdx {
				if len(s.Properties) > 0 {
					props := make([]string, len(s.Properties))
					copy(props, s.Properties)
					sort.Strings(props)
					schemaFieldsMap[s.Name] = props
				}
			}

			tops := topTags(index, 8)
			recipes := buildRecipes(key, index, schemaNames, schemaFieldsMap)

			return out.JSON(map[string]any{
				"ref":     formatRef(key, version),
				"key":     key,
				"version": version,
				"counts": map[string]any{
					"operations": len(index),
					"schemas":    len(schemaNames),
				},
				"topTags": tops,
				"recipes": recipes,
			})
		},
	}
}
