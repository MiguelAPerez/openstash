# Using the HTTP API

openstash can run as a long-lived HTTP server (`openstash serve`) so agents and tools can query cached OpenAPI specs without shelling out to the CLI. Responses mirror the JSON the CLI prints.

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

`POST /v1/specs` accepts a small JSON body (`key`, `from`, `version`, `endpoint`). The default 64 KiB cap is plenty when `from` is a URL or file path. Raise it only if you post large specs inline.

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

- `openstash compare`
- `openstash refresh`
- `openstash curl`

## Security notes

The serve API has **no authentication**. It is meant for localhost or a trusted network.

- The CLI defaults to `127.0.0.1:8080`. Binding to `0.0.0.0` (as Docker does for port mapping) exposes the API to anyone who can reach the port.
- `POST /v1/specs` fetches `from` URLs and reads local files on the server — do not expose an unauthenticated instance to untrusted networks.
