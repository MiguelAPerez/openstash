package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/spf13/cobra"
)

func newDump() *cobra.Command {
	return &cobra.Command{
		Use:   "dump <key[@version]>",
		Short: "Print the full stored OpenAPI document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, doc, _, err := mustLoad(args[0])
			if err != nil {
				return err
			}
			return out.JSON(doc)
		},
	}
}
