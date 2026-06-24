# Containerized serve server

> **Status: implemented.** `openstash serve` (`internal/server`, `internal/cli/serve.go`),
> the `Dockerfile`, `docker-compose.yml`, the CI smoke-test job, and the GHCR publish
> step in `release.yml` all ship in this repo. The design notes below are kept for
> rationale; where they say "proposed"/"sketch"/"later", read them as shipped.

## Goal

Give developers a long-running **HTTP API** over the existing openstash store so agents and tools can query cached OpenAPI specs without shelling out to the CLI. Ship it as a **Docker image** with a volume-mounted data directory.

## Current state

- openstash is a **Go CLI** (`cmd/openstash`, Cobra commands in `internal/cli/`).
- Specs live on disk under `--store` (default `~/.openstash/specs/<key>/<version>/`).
- Core logic is already library code: `internal/store`, `internal/search`, `internal/spec`.
- Commands emit **JSON to stdout** via `internal/out` — good shape to mirror in HTTP responses.
- The HTTP server, `Dockerfile`, and `docker-compose.yml` now live in `internal/server` and the repo root.
- `internal/spec/server.go` parses OpenAPI `servers[]` URLs for `openstash curl`; it is not an HTTP server.

## Proposed shape

### CLI entrypoint

Add `openstash serve`:

```bash
openstash serve --addr :8080 --store /data
```

Honor `OPENSTASH_STORE` when `--store` is unset (useful in containers).

### HTTP API (stdlib `net/http`, Go 1.22 routing)

Mirror CLI behavior; return the same JSON structures the CLI already prints. See **Decisions** below for path and payload conventions.

| Method | Path | CLI equivalent |
|--------|------|----------------|
| `GET` | `/health` | liveness |
| `GET` | `/v1/specs` | `openstash list` |
| `POST` | `/v1/specs` | `openstash add` |
| `GET` | `/v1/specs/{specKey}` | `openstash dump` (latest) |
| `GET` | `/v1/specs/{specKey}/versions/{version}` | `openstash dump` (pinned) |
| `GET` | `/v1/specs/{specKey}/versions/{version}/operations` | `openstash search` / `show` / `gather` |

Errors: `{"error":"..."}` with appropriate status codes (400 client, 404 not found, 409 conflict, 422 validation, 500 internal).

### Package layout

```
internal/server/   # HTTP handlers, wired to store/search/spec
internal/cli/serve.go
```

Keep handlers thin — reuse packages, don't fork business logic.

### Container

Multi-stage build from repo root; distroless or alpine runtime.

```dockerfile
# shipped — see ./Dockerfile (kept here for reference)
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /openstash ./cmd/openstash

FROM gcr.io/distroless/static-debian12
COPY --from=build /openstash /openstash
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/openstash", "serve", "--store", "/data", "--addr", ":8080"]
```

**docker-compose** (dev):

- build from repo
- `8080:8080`
- volume `./dev-store:/data` (or mount host `~/.openstash`)

### CI smoke test

Add to `.github/workflows/ci.yml`:

1. `docker build`
2. `curl -f localhost:8080/health`

Publishing the image on release alongside the tar.gz binaries is wired up in `release.yml` (GHCR, `:latest` + the release tag).

## Non-goals (v1)

- Auth / TLS — local dev only; bind to localhost or trusted network.
- MCP or gRPC — HTTP first; same handlers can back MCP later.
- Static UI or nginx — the value is the Go store/search API.
- New dependencies — stdlib HTTP is enough unless routing grows unwieldy.

## Trust model

The serve API has **no authentication**. It is meant to run on localhost (or a
trusted network) for a single developer/agent. Treat anyone who can reach the
port as fully trusted:

- The CLI defaults `--addr` to `127.0.0.1:8080`. Binding it to `0.0.0.0`/`:8080`
  (as the Dockerfile/compose do for port mapping) is an explicit opt-in — only
  expose the port to networks you trust.
- `POST /v1/specs` loads `from` via `spec.LoadFrom`, which reads local files **and**
  fetches `http(s)` URLs by design (it mirrors `openstash add`). On an exposed,
  unauthenticated port this is an SSRF / local-file-read primitive — don't expose it.

Defense in depth that *is* enforced regardless of bind address:

- `key` and `version` are used as on-disk path segments, so handlers reject any
  value containing `..`, `/`, `\`, or `@` (`validatePathSegment` in
  `internal/server/handlers.go`) to prevent traversal out of the store root.
- `POST /v1/specs` caps the request body (default 64 KiB, override via
  `OPENSTASH_MAX_BODY_BYTES` or `--max-body-bytes`) and rejects unknown JSON fields.

## Decisions (OpenAPI 3.1 aligned)

OpenAPI 3.1 distinguishes three version concepts — keep them separate in the serve API:

| Concept | Example | Where it lives |
|---------|---------|----------------|
| **OpenAPI format** | `openapi: 3.1.0` | Field on the stored document; returned in dump/list metadata |
| **Document version** | `info.version` (`1.2.1`, semver) | Stored-spec identity; maps to `key` + `version` in openstash |
| **HTTP API version** | `/v1/…` | openstash serve implementation; bump only on breaking handler changes |

Zalando / OpenAPI Initiative guidance applies to how we *expose* stored specs over HTTP, not to rewriting the specs themselves.

### Path style → use nested sub-resources

**Recommendation:** `/v1/specs/{specKey}/versions/{version}/…`

- Treat `specKey` and `version` as **resource identifiers in path segments**, not query params. Query params are for search/filter/pagination on a resource, not for naming the resource (Zalando: sub-resources via path segments).
- Use **kebab-case** literal segments (`specs`, `versions`, `operations`).
- Keep nesting shallow (≤2 levels after `/v1`): `specs → versions → operations`.
- `specKey` must be URL-safe (`[a-zA-Z0-9:._-]*`); encode when needed. Do **not** put `key@version` in paths — `@` and compound refs belong in CLI ergonomics, not URLs.
- **Latest version:** `GET /v1/specs/{specKey}` resolves server-side to latest stored `info.version` (same semantics as bare `key` in CLI). Optionally add `GET /v1/specs/{specKey}/versions` to list stored versions.
- **Search/filter** stays on query params (lowerCamelCase): `q`, `limit`, `pathPrefix`, `method`, `in` — these mirror CLI flags and match OpenAPI parameter style.

Revised route table:

| Method | Path | CLI equivalent |
|--------|------|----------------|
| `GET` | `/health` | liveness |
| `GET` | `/v1/specs` | `openstash list` |
| `POST` | `/v1/specs` | `openstash add` |
| `GET` | `/v1/specs/{specKey}` | `openstash dump` (latest) |
| `GET` | `/v1/specs/{specKey}/versions/{version}` | `openstash dump` (pinned) |
| `GET` | `/v1/specs/{specKey}/versions/{version}/operations` | `openstash search` / `show` / `gather` |

Use a `detail` query param (`search` \| `show` \| `gather`, default `search`) or separate sub-paths later if the single endpoint feels too RPC-ish. Prefer **one operations collection** with filters over verb paths like `/search` (nouns in URLs, HTTP method is the verb).

### Add via HTTP → design-first request/response schema

**Recommendation:** `POST /v1/specs` with JSON body (lowerCamelCase):

```json
{
  "key": "gitea",
  "from": "https://gitea.example/swagger.v1.json",
  "version": "1.0.0",
  "endpoint": "https://gitea.example/api/v1"
}
```

- `key` and `from` required; `version` optional (default: `info.version` from fetched doc, same as CLI).
- `from`: prefer **URI** (`format: uri` in the serve OpenAPI doc). File paths work inside containers but URLs are the primary case.
- `endpoint`: optional override for `servers[0].url` when calling described APIs.
- **201 Created** + body `{ status, meta, indexed, schemas }` (same shape as CLI `add` output).
- **409 Conflict** if `key` + `version` already exists.
- **422 Unprocessable Entity** if spec is not OpenAPI/Swagger or lacks `info.version` when `version` omitted.

Ship an `openapi.yaml` for the serve API itself (OpenAPI 3.1, design-first) — dogfood the tool and give agents a stable contract.

### Image name/registry

**Recommendation:** `ghcr.io/miguelaperez/openstash:<tag>` alongside existing GitHub Release tarballs.

- Tag with release version (`0.4.0`) and `latest` on stable releases.
- Document in README; wire publish step in `release.yml` when implementation lands.
