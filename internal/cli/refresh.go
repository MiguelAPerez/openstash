package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newRefresh() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <key[@version]>",
		Short: "Check source for a newer spec version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, key, version, doc, _, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			meta, err := st.LoadMeta(key, version)
			if err != nil {
				return err
			}

			remote, err := spec.LoadFrom(meta.Source)
			if err != nil {
				return err
			}

			storedVer := spec.InfoVersion(doc)
			remoteVer := spec.InfoVersion(remote)

			result := map[string]any{
				"ref":             formatRef(key, version),
				"storedVersion":   storedVer,
				"remoteVersion":   remoteVer,
				"updateAvailable": remoteVer != "" && remoteVer != storedVer,
			}

			if result["updateAvailable"].(bool) {
				result["hint"] = "add with a new --version tag to store the updated spec"
			}

			return out.JSON(result)
		},
	}
}
