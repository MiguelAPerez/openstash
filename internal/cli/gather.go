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
	var expand int

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

			// Exact mode: path + method without query
			if path != "" && method != "" {
				op, err := spec.GetOperation(doc, path, method)
				if err != nil {
					return err
				}
				return out.JSON(map[string]any{
					"ref":        formatRef(key, version),
					"key":        key,
					"version":    version,
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
					"mode": "search", "query": query, "hits": hits, "operations": []any{},
				})
			}
			if expand <= 0 {
				expand = len(hits)
			}
			if expand > len(hits) {
				expand = len(hits)
			}

			var operations []*spec.OperationDetail
			for i := 0; i < expand; i++ {
				h := hits[i]
				op, err := spec.GetOperation(doc, h.Operation.Path, h.Operation.Method)
				if err != nil {
					continue
				}
				operations = append(operations, op)
			}

			return out.JSON(map[string]any{
				"ref":        formatRef(key, version),
				"key":        key,
				"version":    version,
				"mode":       "search",
				"query":      query,
				"hits":       hits,
				"operations": operations,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "max search hits")
	cmd.Flags().IntVar(&expand, "expand", 3, "how many hits to expand with full detail")
	cmd.Flags().StringVar(&pathPrefix, "path-prefix", "", "filter paths by prefix")
	cmd.Flags().StringVar(&method, "method", "", "filter by HTTP method")
	cmd.Flags().StringVar(&path, "path", "", "exact path (use with --method)")
	return cmd
}
