package cli

import (
	"fmt"
	"os"

	"github.com/MiguelAPerez/openstash/internal/server"
	"github.com/spf13/cobra"
)

func newServe() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP API server",
		Long:  "Expose the openstash store over HTTP for agents and tools. See api/serve.openapi.yaml for the contract.",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			srv := server.New(st, addr)
			fmt.Fprintf(os.Stderr, "openstash serve listening on %s\n", addr)
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	return cmd
}
