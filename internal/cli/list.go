package cli

import (
	"sort"

	"github.com/MiguelAPerez/openstash/internal/out"
	"github.com/MiguelAPerez/openstash/internal/spec"
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

			enriched := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				key := e.Key
				version := e.Version

				// Load operation index for hints.
				index, _ := st.LoadIndex(key, version)

				// Load or build schema index; extract sorted schema names.
				schemaIdx, _ := st.LoadSchemaIndex(key, version)
				if schemaIdx == nil {
					doc, err := st.LoadSpec(key, version)
					if err == nil {
						schemaIdx = spec.BuildSchemaIndex(doc)
					}
				}
				schemaNames := make([]string, 0, len(schemaIdx))
				for _, s := range schemaIdx {
					schemaNames = append(schemaNames, s.Name)
				}
				sort.Strings(schemaNames)

				hints := entryHints(key, index, schemaNames)

				row := map[string]any{
					"key":         e.Key,
					"version":     e.Version,
					"source":      e.Source,
					"endpoint":    e.Endpoint,
					"fetchedAt":   e.FetchedAt,
					"specVersion": e.SpecVersion,
					"hints":       hints,
				}
				enriched = append(enriched, row)
			}

			return out.JSON(map[string]any{"entries": enriched})
		},
	}
}
