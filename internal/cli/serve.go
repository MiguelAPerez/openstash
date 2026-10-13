package cli

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/server"
	"github.com/spf13/cobra"
)

func newServe() *cobra.Command {
	var addr string
	var maxBodyBytes int64
	var noUI bool
	var allowedHosts []string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP API server",
		Long: "Expose the openstash store over HTTP for agents and tools, with a browser UI at /. " +
			"See api/serve.openapi.yaml for the contract.",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			limit, err := resolveMaxBodyBytes(maxBodyBytes)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("allowed-hosts") {
				allowedHosts = splitList(os.Getenv("OPENSTASH_ALLOWED_HOSTS"))
			}
			srv := server.NewWithOptions(st, addr, limit, server.Options{DisableUI: noUI, AllowedHosts: allowedHosts})
			fmt.Fprintf(os.Stderr, "openstash serve listening on %s\n", addr)
			if !noUI {
				fmt.Fprintf(os.Stderr, "web UI: %s\n", uiURL(addr))
			}
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	cmd.Flags().Int64Var(&maxBodyBytes, "max-body-bytes", 0,
		"max POST /v1/specs request body in bytes (0 = OPENSTASH_MAX_BODY_BYTES or built-in default)")
	cmd.Flags().BoolVar(&noUI, "no-ui", false, "disable the browser UI served at /")
	cmd.Flags().StringSliceVar(&allowedHosts, "allowed-hosts", nil,
		"extra Host names to accept besides localhost and IPs, or * for any "+
			"(default: OPENSTASH_ALLOWED_HOSTS; any host when bound to a wildcard address)")
	return cmd
}

// splitList parses a comma-separated env value, dropping empty items.
func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// uiURL turns a listen address into a clickable URL for the startup log.
func uiURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr + "/"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
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
