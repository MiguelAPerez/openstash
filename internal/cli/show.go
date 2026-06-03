package cli

import (
	"fmt"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newShow() *cobra.Command {
	var path, method string
	var expand bool
	var depth int

	cmd := &cobra.Command{
		Use:   "show <key[@version]>",
		Short: "Full detail for one operation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" || method == "" {
				return fmt.Errorf("--path and --method are required")
			}

			_, key, version, doc, _, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			// Compute effective depth: --depth wins over --expand.
			effectiveDepth := depth
			if effectiveDepth == 0 && expand {
				effectiveDepth = 1
			}

			op, err := spec.GetOperationDepth(doc, path, method, effectiveDepth)
			if err != nil {
				return err
			}

			return out.JSON(map[string]any{
				"ref":       formatRef(key, version),
				"key":       key,
				"version":   version,
				"depth":     effectiveDepth,
				"operation": op,
			})
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "OpenAPI path (e.g. /user/repos)")
	cmd.Flags().StringVar(&method, "method", "", "HTTP method (e.g. GET)")
	cmd.Flags().BoolVar(&expand, "expand", false, "Inline $ref schemas one level deep (shorthand for --depth 1)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Depth of $ref inlining for schemas (0 = shallow, default)")
	return cmd
}
