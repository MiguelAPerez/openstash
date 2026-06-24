package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/MiguelAPerez/openstash/internal/server"
	"github.com/spf13/cobra"
)

func newServe() *cobra.Command {
	var addr string
	var maxBodyBytes int64

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP API server",
		Long:  "Expose the openstash store over HTTP for agents and tools. See api/serve.openapi.yaml for the contract.",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			limit, err := resolveMaxBodyBytes(maxBodyBytes)
			if err != nil {
				return err
			}
			srv := server.New(st, addr, limit)
			fmt.Fprintf(os.Stderr, "openstash serve listening on %s\n", addr)
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	cmd.Flags().Int64Var(&maxBodyBytes, "max-body-bytes", 0,
		"max POST /v1/specs request body in bytes (0 = OPENSTASH_MAX_BODY_BYTES or built-in default)")
	return cmd
}

// resolveMaxBodyBytes applies precedence: --max-body-bytes flag (when > 0), then
// the OPENSTASH_MAX_BODY_BYTES env var, then 0 (server falls back to its default).
func resolveMaxBodyBytes(flagVal int64) (int64, error) {
	if flagVal > 0 {
		return flagVal, nil
	}
	raw := os.Getenv("OPENSTASH_MAX_BODY_BYTES")
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid OPENSTASH_MAX_BODY_BYTES %q: want a positive integer (bytes)", raw)
	}
	return n, nil
}
