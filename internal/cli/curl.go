package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/spf13/cobra"
)

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func newCurl() *cobra.Command {
	var operationID, host, token, username, password string
	var params []string

	cmd := &cobra.Command{
		Use:   "curl <key[@version]>",
		Short: "Execute an API call for one operation",
		Long: `Execute a request against a cached spec operation.

Identify the operation by its operationId with --operation. The host defaults
to the endpoint stored with the spec and can be overridden with --host.

Params (path, query, and body) are passed as --param key=value and placed in
the correct location based on the spec. Auth is --token (Bearer) or
--username + --password (basic auth).

Examples:
  openstash curl gitea --operation issueListIssues \
    --token mytoken \
    --param owner=alice --param repo=myrepo

  openstash curl gitea --operation issueCreateIssue \
    --username alice --password secret \
    --param owner=alice --param repo=myrepo --param title=Bug --param body="Repro steps"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if operationID == "" {
				return fmt.Errorf("--operation is required")
			}
			if token == "" && username == "" {
				return fmt.Errorf("auth required: --token or --username + --password")
			}

			st, key, version, doc, index, err := mustLoad(args[0])
			if err != nil {
				return err
			}

			specHost, specPath := spec.ServerBase(doc)

			if host != "" {
				host = strings.TrimRight(host, "/")
			} else {
				meta, _ := st.LoadMeta(key, version)
				endpoint := strings.TrimRight(meta.Endpoint, "/")
				switch {
				case endpoint != "":
					host = endpoint
				case specHost != "":
					host = specHost + specPath
					specPath = "" // already applied
				default:
					return fmt.Errorf("--host required: no endpoint stored for %s@%s and spec has no absolute server URL", key, version)
				}
			}

			// Append the spec path prefix when the base has no path yet
			// (e.g. stored endpoint is https://gitea.example.com, basePath is /api/v1)
			if specPath != "" {
				if parsed, perr := url.Parse(host); perr == nil && (parsed.Path == "" || parsed.Path == "/") {
					host += specPath
				}
			}

			var opPath, opMethod string
			for _, idx := range index {
				if idx.OperationID == operationID {
					opPath = idx.Path
					opMethod = idx.Method
					break
				}
			}
			if opPath == "" {
				return fmt.Errorf("operation %q not found in %s@%s", operationID, key, version)
			}

			op, err := spec.GetOperation(doc, opPath, opMethod)
			if err != nil {
				return err
			}

			kv, err := parseParams(params)
			if err != nil {
				return err
			}

			return runCurl(runCurlArgs{
				method:   opMethod,
				path:     opPath,
				host:     host,
				token:    token,
				username: username,
				password: password,
				op:       op,
				params:   kv,
			})
		},
	}

	cmd.Flags().StringVar(&operationID, "operation", "", "operationId of the API operation to call")
	cmd.Flags().StringVar(&host, "host", "", "base URL override (defaults to the endpoint stored with the spec)")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for Authorization header")
	cmd.Flags().StringVar(&username, "username", "", "Username for HTTP basic auth")
	cmd.Flags().StringVar(&password, "password", "", "Password for HTTP basic auth")
	cmd.Flags().StringArrayVar(&params, "param", nil, "key=value param (repeatable); placed in path, query, or body based on the spec")
	return cmd
}

type runCurlArgs struct {
	method, path, host    string
	token, username, password string
	op     *spec.OperationDetail
	params map[string]string
}

func parseParams(raw []string) (map[string]string, error) {
	out := map[string]string{}
	for _, s := range raw {
		k, v, ok := strings.Cut(s, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("--param %q: expected key=value", s)
		}
		out[k] = v
	}
	return out, nil
}

func runCurl(a runCurlArgs) error {
	pathNames := pathParamNames(a.path)
	queryNames := specQueryParamNames(a.op.Parameters)

	remaining := make(map[string]string, len(a.params))
	for k, v := range a.params {
		remaining[k] = v
	}

	urlPath := a.path
	for _, name := range pathNames {
		v, ok := remaining[name]
		if !ok {
			return fmt.Errorf("missing required path param: --param %s=<value>", name)
		}
		urlPath = strings.ReplaceAll(urlPath, "{"+name+"}", url.PathEscape(v))
		delete(remaining, name)
	}

	query := url.Values{}
	body := map[string]string{}
	for name, val := range remaining {
		if queryNames[name] {
			query.Set(name, val)
		} else {
			body[name] = val
		}
	}

	fullURL := a.host + urlPath
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	curlArgs := []string{"-s", "-w", "\n%{http_code}", "-X", a.method, fullURL}

	if a.token != "" {
		curlArgs = append(curlArgs, "-H", "Authorization: Bearer "+a.token)
	} else {
		curlArgs = append(curlArgs, "-u", a.username+":"+a.password)
	}

	if len(body) > 0 {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		curlArgs = append(curlArgs, "-H", "Content-Type: application/json", "-d", string(b))
	}

	c := exec.Command("curl", curlArgs...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func pathParamNames(path string) []string {
	matches := pathParamRe.FindAllStringSubmatch(path, -1)
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		names = append(names, m[1])
	}
	return names
}

func specQueryParamNames(parameters []any) map[string]bool {
	out := map[string]bool{}
	for _, p := range parameters {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if in, _ := pm["in"].(string); in == "query" {
			if name, _ := pm["name"].(string); name != "" {
				out[name] = true
			}
		}
	}
	return out
}
