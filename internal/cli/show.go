package cli

import (
	"fmt"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newShow() *cobra.Command {
	var path, method string

	cmd := &cobra.Command{
		Use:   "show <key@version>",
		Short: "Full detail for one operation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" || method == "" {
				return fmt.Errorf("--path and --method are required")
			}

			ref := args[0]
			_, key, version, doc, _, err := mustLoad(ref)
			if err != nil {
				return err
			}

			op, err := spec.GetOperation(doc, path, method)
			if err != nil {
				return err
			}

			return out.JSON(map[string]any{
				"ref":       ref,
				"key":       key,
				"version":   version,
				"operation": op,
			})
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "OpenAPI path (e.g. /user/repos)")
	cmd.Flags().StringVar(&method, "method", "", "HTTP method (e.g. GET)")
	return cmd
}
