# openstash

**openstash** caches OpenAPI specs locally so you (or an agent) can look up endpoints without re-downloading or parsing huge `swagger.json` files every time.

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

Or download manually for your platform:

| Platform | Download |
|---|---|
| macOS (Apple Silicon) | `openstash_darwin_arm64.tar.gz` |
| macOS (Intel) | `openstash_darwin_amd64.tar.gz` |
| Linux (x86-64) | `openstash_linux_amd64.tar.gz` |
| Linux (ARM64) | `openstash_linux_arm64.tar.gz` |

Extract and move the `openstash` binary somewhere on your `PATH` (e.g. `/usr/local/bin`).

Verify:

```bash
openstash --version
```

### Go install

If you already have Go 1.22+ installed:

```bash
go install github.com/MiguelAPerez/openstash/cmd/openstash@latest
```

Go installs binaries to `~/go/bin`. If your shell can't find `openstash` after this, add that directory to your PATH:

```bash
# add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

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
```

### gather — search plus expanded details

```bash
openstash gather gitea "subscription" --expand 3
openstash gather gitea@1.0.0 --path /user/repos --method GET
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

YAML and JSON sources are normalized to JSON on disk.
