package cli

import (
	"fmt"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

var validCompareSections = map[string]bool{
	"operations": true,
	"schemas":    true,
}

func newCompare() *cobra.Command {
	var brief bool
	var limit int
	var sections []string

	cmd := &cobra.Command{
		Use:   "compare <baseline> <target>",
		Short: "Diff two stored specs (operations and schemas)",
		Long: `Compare two stored specs by key@version. Omit the version to use the latest stored version for that key.

The first argument is the baseline; the second is the target.
  added   — present in target only
  removed — present in baseline only
  changed — present in both with differences

Examples:
  openstash compare forgejo@12 forgejo@15
  openstash compare forgejo gitea
  openstash compare forgejo gitea --brief
  openstash compare forgejo gitea --section operations --limit 10`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baselineKey, baselineVersion, baselineDoc, err := loadRef(args[0])
			if err != nil {
				return err
			}
			targetKey, targetVersion, targetDoc, err := loadRef(args[1])
			if err != nil {
				return err
			}

			sectionSet, err := parseCompareSections(sections)
			if err != nil {
				return err
			}

			result := spec.Compare(baselineDoc, targetDoc)
			result.Baseline.Key = baselineKey
			result.Baseline.Version = baselineVersion
			result.Target.Key = targetKey
			result.Target.Version = targetVersion

			output := map[string]any{
				"baseline": formatRef(baselineKey, baselineVersion),
				"target":   formatRef(targetKey, targetVersion),
				"legend": map[string]string{
					"added":   "present in target only",
					"removed": "present in baseline only",
					"changed": "present in both with differences",
				},
				"summary": result.Summary,
			}

			if brief {
				return out.JSON(output)
			}

			if sectionSet["operations"] {
				output["operations"] = limitCompareOps(result.Operations, limit)
			}
			if sectionSet["schemas"] {
				output["schemas"] = limitCompareSchemas(result.Schemas, limit)
			}

			return out.JSON(output)
		},
	}

	cmd.Flags().BoolVar(&brief, "brief", false, "summary only (omit operation and schema lists)")
	cmd.Flags().IntVar(&limit, "limit", 0, "max items per added/removed/changed list (0 = unlimited)")
	cmd.Flags().StringArrayVar(&sections, "section", nil, "sections to include: operations, schemas (default: both)")
	return cmd
}

func parseCompareSections(sections []string) (map[string]bool, error) {
	if len(sections) == 0 {
		return map[string]bool{"operations": true, "schemas": true}, nil
	}
	out := make(map[string]bool)
	for _, raw := range sections {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if !validCompareSections[part] {
				return nil, fmt.Errorf("unknown --section %q; valid values: operations, schemas", part)
			}
			out[part] = true
		}
	}
	return out, nil
}

func limitCompareOps(ops spec.CompareOpsResult, limit int) map[string]any {
	added, addedTotal := limitSlice(ops.Added, limit)
	removed, removedTotal := limitSlice(ops.Removed, limit)
	changed, changedTotal := limitSlice(ops.Changed, limit)

	result := map[string]any{
		"added":   added,
		"removed": removed,
		"changed": changed,
	}
	if limit > 0 {
		result["totals"] = map[string]int{
			"added":   addedTotal,
			"removed": removedTotal,
			"changed": changedTotal,
		}
	}
	return result
}

func limitCompareSchemas(schemas spec.CompareSchemas, limit int) map[string]any {
	added, addedTotal := limitSlice(schemas.Added, limit)
	removed, removedTotal := limitSlice(schemas.Removed, limit)
	changed, changedTotal := limitSlice(schemas.Changed, limit)

	result := map[string]any{
		"added":   added,
		"removed": removed,
		"changed": changed,
	}
	if limit > 0 {
		result["totals"] = map[string]int{
			"added":   addedTotal,
			"removed": removedTotal,
			"changed": changedTotal,
		}
	}
	return result
}

func limitSlice[T any](items []T, limit int) (shown []T, total int) {
	total = len(items)
	if limit <= 0 || len(items) <= limit {
		return items, total
	}
	return items[:limit], total
}

func loadRef(ref string) (key, version string, doc map[string]any, err error) {
	_, key, version, doc, _, err = mustLoad(ref)
	return key, version, doc, err
}
