package cli

import (
	"fmt"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newAdd() *cobra.Command {
	var from, endpoint, version string

	cmd := &cobra.Command{
		Use:   "add <key>",
		Short: "Ingest an OpenAPI spec from a URL or file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if from == "" {
				return fmt.Errorf("--from is required (url or file path)")
			}

			doc, err := spec.LoadFrom(from)
			if err != nil {
				return err
			}

			if version == "" {
				version = spec.InfoVersion(doc)
				if version == "" {
					return fmt.Errorf("--version required (spec has no info.version)")
				}
			}

			st, err := openStore()
			if err != nil {
				return err
			}
			if st.Exists(key, version) {
				return fmt.Errorf("already exists: %s@%s", key, version)
			}

			meta, err := st.Add(key, version, from, endpoint, doc)
			if err != nil {
				return err
			}

			return out.JSON(map[string]any{
				"status":  "added",
				"meta":    meta,
				"indexed": len(spec.BuildIndex(doc)),
			})
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "source URL or file path")
	cmd.Flags().StringVar(&version, "version", "", "version tag (default: info.version from spec)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "API base URL described by the spec")
	return cmd
}
