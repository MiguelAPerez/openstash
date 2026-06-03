package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newSchema() *cobra.Command {
	var depth int
	var fields bool

	cmd := &cobra.Command{
		Use:   "schema <key[@version]> <ComponentName>",
		Short: "Inspect a component schema (resolve $refs or list fields)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, key, version, doc, _, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			name := args[1]
			node, err := spec.GetSchema(doc, name)
			if err != nil {
				return err
			}

			ref := formatRef(key, version)

			if fields {
				return out.JSON(map[string]any{
					"ref":      ref,
					"key":      key,
					"version":  version,
					"name":     name,
					"required": node["required"],
					"fields":   spec.SchemaFields(node),
				})
			}

			return out.JSON(map[string]any{
				"ref":     ref,
				"key":     key,
				"version": version,
				"name":    name,
				"depth":   depth,
				"schema":  spec.ResolveSchema(doc, node, depth),
			})
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 1, "number of $ref hops to inline (0 = keep refs as short names)")
	cmd.Flags().BoolVar(&fields, "fields", false, "list fields instead of resolving schema")
	return cmd
}
