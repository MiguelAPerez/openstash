package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	storeRoot string
	version   = "dev"
)

func Execute() {
	if err := NewRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:     "openstash",
		Short:   "Local OpenAPI cache and search for agents",
		Long:    "Store OpenAPI specs by key@version. Use search (slim), show (one op), or gather (search + details).",
		Version: version,
	}
	root.PersistentFlags().StringVar(&storeRoot, "store", "", "store directory (default: ~/.openstash)")

	root.AddCommand(newAdd())
	root.AddCommand(newList())
	root.AddCommand(newSearch())
	root.AddCommand(newShow())
	root.AddCommand(newGather())
	root.AddCommand(newRefresh())
	root.AddCommand(newSchema())
	root.AddCommand(newHas())

	return root
}
