package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/search"
	"github.com/spf13/cobra"
)

func newSearch() *cobra.Command {
	var limit int
	var pathPrefix, method string

	cmd := &cobra.Command{
		Use:   "search <key[@version]> [query]",
		Short: "Slim search for matching endpoints",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 1 {
				query = args[1]
			}

			_, key, version, _, index, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			hits := search.Query(index, query, limit, pathPrefix, method)
			return out.JSON(map[string]any{
				"ref":     formatRef(key, version),
				"key":     key,
				"version": version,
				"query":   query,
				"hits":    hits,
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "max results")
	cmd.Flags().StringVar(&pathPrefix, "path-prefix", "", "filter paths by prefix")
	cmd.Flags().StringVar(&method, "method", "", "filter by HTTP method")
	return cmd
}
