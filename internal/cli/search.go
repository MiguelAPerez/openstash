package cli

import (
	"fmt"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/search"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

var validInValues = map[string]bool{
	"paths":        true,
	"schemas":      true,
	"descriptions": true,
}

func newSearch() *cobra.Command {
	var limit int
	var pathPrefix, method string
	var inScopes []string

	cmd := &cobra.Command{
		Use:   "search <key[@version]> [query]",
		Short: "Slim search for matching endpoints",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 1 {
				query = args[1]
			}

			// Parse and validate --in scopes.
			scopeSet := make(map[string]bool)
			for _, raw := range inScopes {
				for _, part := range strings.Split(raw, ",") {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					if !validInValues[part] {
						var valid []string
						for k := range validInValues {
							valid = append(valid, k)
						}
						return fmt.Errorf("unknown --in value %q; valid values: %s", part, strings.Join(valid, ", "))
					}
					scopeSet[part] = true
				}
			}
			// Default: paths only (current behavior).
			if len(scopeSet) == 0 {
				scopeSet["paths"] = true
			}

			wantPaths := scopeSet["paths"]
			wantSchemas := scopeSet["schemas"]

			st, key, version, doc, index, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			result := map[string]any{
				"ref":     formatRef(key, version),
				"key":     key,
				"version": version,
				"query":   query,
			}

			if wantPaths {
				hits := search.Query(index, query, limit, pathPrefix, method)
				result["hits"] = hits
			}

			if wantSchemas {
				var schemaIdx []spec.SchemaIndex
				loaded, loadErr := st.LoadSchemaIndex(key, version)
				if loadErr != nil {
					return loadErr
				}
				if loaded != nil {
					schemaIdx = loaded
				} else {
					schemaIdx = spec.BuildSchemaIndex(doc)
				}
				schemaHits := search.SearchSchemas(schemaIdx, query, limit)
				result["schemas"] = schemaHits
			}

			return out.JSON(result)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "max results")
	cmd.Flags().StringVar(&pathPrefix, "path-prefix", "", "filter paths by prefix")
	cmd.Flags().StringVar(&method, "method", "", "filter by HTTP method")
	cmd.Flags().StringArrayVar(&inScopes, "in", nil, "scopes to search: paths, schemas, descriptions (default: paths)")
	return cmd
}
