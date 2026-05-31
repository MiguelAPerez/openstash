# openstash

> **Early preview** — experimental CLI, not production-ready. APIs and behavior may change.

**openstash** caches OpenAPI specs locally so you (or an agent) can look up endpoints without re-downloading or parsing huge `swagger.json` files every time.

## Why use it?

- **Less noise** — Slim search results or one operation at a time instead of an entire spec in context.
- **Stable references** — Same `key@version` every session.
- **Three levels of detail** — `search` (discover), `show` (one op), `gather` (search + details).

## Install

Install the latest [release](https://github.com/MiguelAPerez/openstash/releases/latest) with Go:

```bash
go install github.com/MiguelAPerez/openstash/cmd/openstash@latest
```

Pin a specific version:

```bash
go install github.com/MiguelAPerez/openstash/cmd/openstash@v0.1.0
```

> Pre-built binaries (darwin/linux, amd64/arm64) are attached on the [releases](https://github.com/MiguelAPerez/openstash/releases) page. Download, extract, and put `openstash` on your `PATH`.

### Build from source

```bash
git clone git@github.com:MiguelAPerez/openstash.git
cd openstash
go build -o bin/openstash ./cmd/openstash
./bin/openstash --help
```

Requires [Go](https://go.dev/dl/) 1.22+.

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

Every stored spec is referenced as `**key@version**`.

### search — find endpoints (slim)

```bash
openstash search gitea@1.0.0 "user repos"
openstash search gitea@1.0.0 "repos" --path-prefix /user --method GET
```

### show — one operation (full detail)

```bash
openstash show gitea@1.0.0 --path /user/repos --method GET
```

### gather — search plus expanded details

```bash
openstash gather gitea@1.0.0 "subscription" --expand 3
openstash gather gitea@1.0.0 --path /user/repos --method GET
```

### refresh — check for updates

Re-fetches the source and reports whether `info.version` changed (does not overwrite stored specs).

```bash
openstash refresh gitea@1.0.0
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

YAML and JSON sources are normalized to JSON on disk.
