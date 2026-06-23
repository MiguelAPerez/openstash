# openstash

A way to cache OpenAPI specs locally so you (or an agent) can look up endpoints without re-downloading or parsing huge `swagger.json` files every time.

*Endpoints your agent actually finds. [See the benchmarks →](bench/README.md)*


| Spec     | Size   | Full spec | openstash | Saved    | Accuracy      |
| -------- | ------ | --------- | --------- | -------- | ------------- |
| petstore | 20 KB  | 4,953     | 1,585     | 3×       | 100% → 100%   |
| cursor   | 72 KB  | 12,548    | 1,833     | 7×       | 0% → 100%     |
| gitea    | 820 KB | 148,146   | 866       | 171×     | 0% → 100%     |
| stripe   | 7.5 MB | 1,054,928 | 5,106     | **207×** | 0% → **100%** |


## Why use it?

- **Less noise** — Slim search results or one operation at a time instead of an entire spec in context.
- **Stable references** — Same `key@version` every session.
- **Three levels of detail** — `search` (discover), `show` (one op), `gather` (search + details).

## Install

### Homebrew (macOS / Linux)

```bash
brew tap MiguelAPerez/tap
brew install openstash
```

### Pre-built binary

macOS and Linux binaries are published on the [releases page](https://github.com/MiguelAPerez/openstash/releases/latest). The one-liner below detects your OS and architecture automatically:

```bash
curl -fsSL https://github.com/MiguelAPerez/openstash/releases/latest/download/openstash_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz | tar xz
sudo mv openstash /usr/local/bin/
```

Verify:

```bash
openstash --version
```

### Docker (HTTP server)

Container images are published to GitHub Container Registry on each release:

```bash
docker pull ghcr.io/miguelaperez/openstash:latest
docker run -p 8080:8080 -v ~/.openstash:/data ghcr.io/miguelaperez/openstash:latest
```

The image runs `openstash serve` with the store at `/data`. Mount your local cache or an empty directory. API contract: [`api/serve.openapi.yaml`](api/serve.openapi.yaml).

For local development from source:

```bash
docker compose up --build
curl http://localhost:8080/health
```

## Getting started

### Add your first spec

```bash
openstash add gitea \
  --from https://gitea.example/swagger.v1.json
```

If the spec has `info.version`, that becomes the version tag automatically. Override it when needed:

```bash
openstash add gitea \
  --version 1.0.0 \
  --from ./swagger.json \
  --endpoint https://gitea.example/api/v1
```

List stored specs:

```bash
openstash list
```

## Usage

Every stored spec is referenced as `key@version`. Omit the version to use the latest stored version for that key.

```bash
openstash search gitea "user repos"          # latest gitea version
openstash search gitea@1.0.0 "user repos"    # specific version
```

### search — find endpoints (slim)

```bash
openstash search gitea "user repos"
openstash search gitea@1.0.0 "repos" --path-prefix /user --method GET
```

### show — one operation (full detail)

```bash
openstash show gitea --path /user/repos --method GET
openstash show gitea --path /user/repos --method GET --depth 2
openstash show gitea --path /user/repos --method GET --expand
```

### gather — search plus expanded details

```bash
openstash gather gitea "subscription" --depth 2
openstash gather gitea "subscription" --expand
openstash gather gitea@1.0.0 --path /user/repos --method GET
```

### dump — full stored spec

Print the complete OpenAPI document from the local store (pipe to `jq`, redirect to a file, etc.).

```bash
openstash dump gitea
openstash dump forgejo@15.0.0 | jq '.info'
```

### compare — diff two specs

Compare operations and schemas between two stored specs. The first argument is the **baseline**; the second is the **target**.

- **added** — present in target only
- **removed** — present in baseline only
- **changed** — present in both with differences

```bash
openstash compare forgejo@12 forgejo@15
openstash compare forgejo gitea --brief
openstash compare forgejo gitea --section operations --limit 10
```

### refresh — check for updates

Re-fetches the source and reports whether `info.version` changed (does not overwrite stored specs).

```bash
openstash refresh gitea
```

## Storage

Default location:

```
~/.openstash/
```

Override for a session:

```bash
openstash --store /path/to/store list
```

Or set `OPENSTASH_STORE` (used by `serve` in containers):

```bash
export OPENSTASH_STORE=/path/to/store
openstash serve --addr :8080
```

YAML and JSON sources are normalized to JSON on disk.