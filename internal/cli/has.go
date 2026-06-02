package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newHas() *cobra.Command {
	return &cobra.Command{
		Use:   "has <key[@version]> <Component[.field[.subfield]]>",
		Short: "Check whether a schema or field path exists",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, key, version, doc, _, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			query := args[1]
			result, err := spec.LookupFieldPath(doc, query)
			if err != nil {
				return err
			}

			return out.JSON(map[string]any{
				"ref":     formatRef(key, version),
				"key":     key,
				"version": version,
				"query":   query,
				"result":  result,
			})
		},
	}
}
