package cli

import (
	"fmt"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/search"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newGather() *cobra.Command {
	var limit int
	var pathPrefix, method, path string
	var expand bool
	var depth int

	cmd := &cobra.Command{
		Use:   "gather <key[@version]> [query]",
		Short: "Search plus expanded operation details",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 1 {
				query = args[1]
			}

			_, key, version, doc, index, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			effectiveDepth := depth
			if effectiveDepth == 0 && expand {
				effectiveDepth = 1
			}

			// Exact mode: path + method without query
			if path != "" && method != "" {
				op, err := spec.GetOperationDepth(doc, path, method, effectiveDepth)
				if err != nil {
					return err
				}
				return out.JSON(map[string]any{
					"ref":        formatRef(key, version),
					"key":        key,
					"version":    version,
					"depth":      effectiveDepth,
					"mode":       "exact",
					"operations": []any{op},
				})
			}

			if query == "" && pathPrefix == "" && method == "" {
				return fmt.Errorf("provide a query, --path-prefix, --method, or --path with --method")
			}

			hits := search.Query(index, query, limit, pathPrefix, method)
			if len(hits) == 0 {
				return out.JSON(map[string]any{
					"ref": formatRef(key, version), "key": key, "version": version,
					"depth": effectiveDepth, "mode": "search", "query": query,
					"hits": hits, "operations": []any{},
				})
			}

			var operations []*spec.OperationDetail
			for _, h := range hits {
				op, err := spec.GetOperationDepth(doc, h.Operation.Path, h.Operation.Method, effectiveDepth)
				if err != nil {
					continue
				}
				operations = append(operations, op)
			}

			return out.JSON(map[string]any{
				"ref":        formatRef(key, version),
				"key":        key,
				"version":    version,
				"depth":      effectiveDepth,
				"mode":       "search",
				"query":      query,
				"hits":       hits,
				"operations": operations,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "max search hits")
	cmd.Flags().BoolVar(&expand, "expand", false, "Inline $ref schemas one level deep (shorthand for --depth 1)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Depth of $ref inlining for schemas (0 = shallow, default)")
	cmd.Flags().StringVar(&pathPrefix, "path-prefix", "", "filter paths by prefix")
	cmd.Flags().StringVar(&method, "method", "", "filter by HTTP method")
	cmd.Flags().StringVar(&path, "path", "", "exact path (use with --method)")
	return cmd
}
