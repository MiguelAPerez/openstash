# Using the HTTP API

openstash can run as a long-lived HTTP server (`openstash serve`) so agents and tools can query cached OpenAPI specs without shelling out to the CLI. Responses mirror the JSON the CLI prints.

The same server hosts a browser UI at `/` (explore, API reference, version timeline with diffs). Disable it with `--no-ui`.

**Base URL:** `http://localhost:8080` (default when running locally or via Docker on port 8080)

**Contract:** [`api/serve.openapi.yaml`](../api/serve.openapi.yaml)

## Start the server

### Docker

```bash
docker pull ghcr.io/miguelaperez/openstash:latest
docker run -p 8080:8080 -v ~/.openstash:/data ghcr.io/miguelaperez/openstash:latest
```

The image runs `openstash serve` with the store at `/data`. Mount your existing cache or an empty directory.

### Local binary

```bash
openstash serve --addr :8080
# or
export OPENSTASH_STORE=~/.openstash
openstash serve --addr 127.0.0.1:8080
```

### Configuration

| Env var | Flag | Default | Purpose |
|---------|------|---------|---------|
| `OPENSTASH_STORE` | `--store` | `~/.openstash` | Store directory (`/data` in the Docker image) |
| `OPENSTASH_MAX_BODY_BYTES` | `--max-body-bytes` | `65536` (64 KiB) | Max `POST /v1/specs` request body |
| — | `--no-ui` | off | Serve the API only (no browser UI at `/`) |
| `OPENSTASH_ALLOWED_HOSTS` | `--allowed-hosts` | none | Extra `Host` names to accept (comma-separated), or `*` for any. See [Security notes](#security-notes) |

`POST /v1/specs` accepts a small JSON body (`key`, `from`, `version`, `endpoint`) and requires `Content-Type: application/json` (anything else gets `415`). The default 64 KiB cap is plenty when `from` is a URL or file path. Raise it only if you post large specs inline.

## Quick check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

All endpoints return JSON. Errors look like `{"error":"..."}` with an appropriate HTTP status (400, 404, 409, 422, or 500).

## Endpoints

| Method | Path | CLI equivalent |
|--------|------|----------------|
| `GET` | `/health` | liveness |
| `GET` | `/v1/specs` | `openstash list` |
| `POST` | `/v1/specs` | `openstash add` |
| `GET` | `/v1/specs/{specKey}` | `openstash dump` (latest) |
| `GET` | `/v1/specs/{specKey}/versions` | list stored versions |
| `GET` | `/v1/specs/{specKey}/versions/{version}` | `openstash dump` (pinned) |
| `GET` | `/v1/specs/{specKey}/versions/{version}/operations` | `openstash search` / `show` / `gather` |
| `GET` | `/v1/specs/{specKey}/versions/{version}/schemas/{name}` | `openstash schema` |
| `GET` | `/v1/specs/{specKey}/timeline` | version history with per-step change counts |
| `GET` | `/v1/specs/{specKey}/compare?from=&to=` | `openstash compare` (same key, definition-aware) |
| `GET` | `/v1/specs/{specKey}/compare/definition` | operation (refs resolved) or raw schema from both versions |

`specKey` and `version` are path segments — use `gitea` and `1.0.0`, not the CLI-style `gitea@1.0.0`.

## Typical workflow

### 1. Add a spec

```bash
curl -X POST http://localhost:8080/v1/specs \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "gitea",
    "from": "https://gitea.example/swagger.v1.json",
    "endpoint": "https://gitea.example/api/v1"
  }'
```

- `key` and `from` are required.
- `from` can be an `http(s)` URL or a file path readable by the server (inside Docker, paths are under the mounted `/data` volume).
- `version` is optional — defaults to `info.version` from the fetched spec.
- Returns **201** with `{"status":"added", "meta":..., "indexed":..., "schemas":...}`.

### 2. List cached specs

```bash
curl http://localhost:8080/v1/specs
# {"entries":[{"key":"gitea","version":"1.0.0","source":"...","endpoint":"...","fetchedAt":"..."}]}
```

### 3. Search operations

Equivalent to `openstash search gitea@1.0.0 "user repos"`:

```bash
curl 'http://localhost:8080/v1/specs/gitea/versions/1.0.0/operations?q=user%20repos'
# {"hits":[...]}
```

Filter params (all optional):

| Param | CLI flag | Example |
|-------|----------|---------|
| `q` | search query | `user repos` |
| `limit` | `--limit` | `10` (default 5) |
| `pathPrefix` | `--path-prefix` | `/user` |
| `method` | `--method` | `GET` |

### 4. Show one operation

Equivalent to `openstash show gitea --path /user/repos --method GET`:

```bash
curl 'http://localhost:8080/v1/specs/gitea/versions/1.0.0/operations?detail=show&path=/user/repos&method=GET'
# {"operation":{...}}
```

Extra params: `depth`, `expand`, `in` (repeat for multiple values, e.g. `in=paths&in=schemas`).

### 5. Gather — search plus expanded details

Equivalent to `openstash gather gitea "subscription" --depth 2`:

```bash
curl 'http://localhost:8080/v1/specs/gitea/versions/1.0.0/operations?detail=gather&q=subscription&depth=2'
# {"operations":[...]}
```

`detail` defaults to `search` when omitted.

### 6. Dump the full spec

Latest version (like `openstash dump gitea`):

```bash
curl http://localhost:8080/v1/specs/gitea
```

Pinned version:

```bash
curl http://localhost:8080/v1/specs/gitea/versions/1.0.0
```

List versions for a key:

```bash
curl http://localhost:8080/v1/specs/gitea/versions
# {"key":"gitea","versions":["1.0.0"]}
```

### 7. Version history

Every version stored under a key forms a timeline. Versions are ordered by semver when they parse. `GET /v1/specs` and `/versions` list versions in the same order, so the last one is the latest:

```bash
curl http://localhost:8080/v1/specs/gitea/timeline
# {"key":"gitea","versions":[{"version":"1.21.0","operations":412,...},...],
#  "steps":[{"from":"1.21.0","to":"1.22.0","summary":{"operations":{"added":9,...},"schemas":{...}}}]}
```

Diff two versions:

```bash
curl 'http://localhost:8080/v1/specs/gitea/compare?from=1.21.0&to=1.22.0'
```

This is `openstash compare` plus definition-level detection. An operation whose `parameters`, `requestBody`, `responses`, `deprecated`, `security`, `servers`, `callbacks`, `consumes` or `produces` differ is listed under `operations.changed` with an empty object for each differing field (for example `"changes":{"parameters":{}}`). A schema whose definition differs gets `"definitionChanged":true`.

Operations are compared as clients see them. Path-level parameters are merged in, and `$ref`s to shared parameters, request bodies, responses and response headers are resolved. So editing `components.parameters.Page` flags every operation that uses it. Document-level `security` (and Swagger 2 `consumes`/`produces`) applies when an operation doesn't set its own.

A change confined to `x-*` vendor extensions (for example an `x-go-package` rename) also sets `"extensionsOnly":true`, so you can tell it apart from real changes. Changes to `info`, top-level `servers` or security scheme definitions on their own are not reported.

Fetch both sides of one change to diff it yourself:

```bash
curl 'http://localhost:8080/v1/specs/gitea/compare/definition?from=1.21.0&to=1.22.0&method=GET&path=/user/repos'
curl 'http://localhost:8080/v1/specs/gitea/compare/definition?from=1.21.0&to=1.22.0&schema=Repository'
# {"baseline":{...}|null,"target":{...}|null,...}
```

For operations, both sides are in the same resolved form the compare uses, so a change behind a `$ref` shows up in the diff.

### 8. One schema

Equivalent to `openstash schema gitea@1.0.0 Repository --depth 1`:

```bash
curl 'http://localhost:8080/v1/specs/gitea/versions/1.0.0/schemas/Repository?depth=1'
```

## From application code

### Python

```python
import requests

base = "http://localhost:8080"

# search
r = requests.get(
    f"{base}/v1/specs/gitea/versions/1.0.0/operations",
    params={"q": "user repos"},
)
hits = r.json()["hits"]

# show one operation
r = requests.get(
    f"{base}/v1/specs/gitea/versions/1.0.0/operations",
    params={"detail": "show", "path": "/user/repos", "method": "GET"},
)
op = r.json()["operation"]
```

### JavaScript

```javascript
const base = "http://localhost:8080";

const url = new URL(`${base}/v1/specs/gitea/versions/1.0.0/operations`);
url.searchParams.set("q", "user repos");
const { hits } = await fetch(url).then((r) => r.json());
```

## Not available over HTTP

These CLI commands have no HTTP equivalent yet:

- `openstash compare` across different keys (same-key compare is `/v1/specs/{specKey}/compare`)
- `openstash refresh`
- `openstash curl`

## Security notes

The serve API has **no authentication**. It is meant for localhost or a trusted network.

- The CLI defaults to `127.0.0.1:8080`. Binding to `0.0.0.0` (as Docker does for port mapping) exposes the API to anyone who can reach the port.
- `POST /v1/specs` fetches `from` URLs and reads local files on the server — do not expose an unauthenticated instance to untrusted networks.
- `POST /v1/specs` only accepts `Content-Type: application/json`. Browsers send a cross-origin JSON request only after a CORS preflight, which the server never approves, so another site can't post a spec through a plain form or `fetch`.
- A site could still use DNS rebinding: point its own domain at `127.0.0.1` so its pages count as same-origin with your server. The server blocks this by checking the `Host` header. When it's bound to a specific host (like the default `127.0.0.1:8080`), it only answers `localhost`, `*.localhost`, IP addresses, the bind host, and names you list in `--allowed-hosts` / `OPENSTASH_ALLOWED_HOSTS`. Anything else gets `403`.
- With a wildcard bind (`--addr :8080` or `0.0.0.0`, as in Docker), any `Host` is accepted by default, so compose service names and LAN hostnames keep working. If you also browse from the machine running the container, set `OPENSTASH_ALLOWED_HOSTS` to the names you use (for example `localhost,openstash`) to turn the check on.
- Schema expansion (`depth`) is capped per response. Refs past the cap come back as `{"$ref": ..., "$truncated": true}`, so a crafted spec can't make one request expand without bound.
