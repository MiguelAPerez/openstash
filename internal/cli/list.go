package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/spf13/cobra"
)

func newList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored key@version entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			entries, err := st.List()
			if err != nil {
				return err
			}
			return out.JSON(map[string]any{"entries": entries})
		},
	}
}
