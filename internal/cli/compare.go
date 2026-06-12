package cli

import (
	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

func newCompare() *cobra.Command {
	return &cobra.Command{
		Use:   "compare <left> <right>",
		Short: "Diff two stored specs (operations and schemas)",
		Long: `Compare two stored specs by key@version. Omit the version to use the latest stored version for that key.

Examples:
  openstash compare forgejo@12 forgejo@15
  openstash compare forgejo gitea`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			leftKey, leftVersion, leftDoc, err := loadRef(args[0])
			if err != nil {
				return err
			}
			rightKey, rightVersion, rightDoc, err := loadRef(args[1])
			if err != nil {
				return err
			}

			result := spec.Compare(leftDoc, rightDoc)
			result.Left.Key = leftKey
			result.Left.Version = leftVersion
			result.Right.Key = rightKey
			result.Right.Version = rightVersion

			return out.JSON(map[string]any{
				"left":       formatRef(leftKey, leftVersion),
				"right":      formatRef(rightKey, rightVersion),
				"summary":    result.Summary,
				"operations": result.Operations,
				"schemas":    result.Schemas,
			})
		},
	}
}

func loadRef(ref string) (key, version string, doc map[string]any, err error) {
	_, key, version, doc, _, err = mustLoad(ref)
	return key, version, doc, err
}
