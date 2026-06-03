# Contributing to openstash

Thanks for your interest in improving openstash! This guide covers how to set up
your environment, build, run, test, and submit changes.

## Prerequisites

- **Go 1.22+** (the toolchain pin lives in [`go.mod`](go.mod); CI builds against
  the same version)
- **git**
- [**golangci-lint**](https://golangci-lint.run/) — optional locally, but CI
  runs it on every PR (pinned to `v1.62.2`)

No other system dependencies are required. openstash is a single Go binary with a
small dependency set (`cobra` for the CLI, `yaml.v3` for spec parsing).

## Getting the source

```bash
git clone https://github.com/MiguelAPerez/openstash.git
cd openstash
go mod download
```

## Project layout

```
cmd/openstash/      # main entry point (thin: just calls cli.Execute)
internal/
  cli/              # cobra commands: add, list, search, show, gather, refresh, schema, has
  spec/             # OpenAPI parsing, indexing, schema/$ref resolution
  search/           # endpoint search/ranking
  store/            # on-disk store (key@version layout, versioning)
  out/              # output formatting helpers
.github/workflows/  # ci.yml (lint + test), release.yml (tagged release builds)
```

All packages live under `internal/`, so they're private to this module.

## Building

Build a local binary into `bin/`:

```bash
go build -o bin/openstash ./cmd/openstash
```

To mirror a release build (strips symbols and stamps the version):

```bash
go build -ldflags="-s -w -X github.com/MiguelAPerez/openstash/internal/cli.version=$(cat VERSION)" \
  -o bin/openstash ./cmd/openstash
```

Without the `-X` ldflag, `openstash --version` reports `dev`.

## Running

Run straight from source without building:

```bash
go run ./cmd/openstash --help
go run ./cmd/openstash add gitea --from ./testdata/swagger.json
go run ./cmd/openstash search gitea "user repos"
```

Or use the binary you built:

```bash
./bin/openstash list
```

### Using a throwaway store

By default specs are cached in `~/.openstash`. To avoid touching your real store
while developing, point `--store` at a temp directory:

```bash
go run ./cmd/openstash --store /tmp/openstash-dev add gitea --from ./swagger.json
go run ./cmd/openstash --store /tmp/openstash-dev list
```

## Testing

Run the full suite the same way CI does:

```bash
go test -race -count=1 ./...
```

Useful variations:

```bash
go test ./internal/spec/...              # one package
go test -run TestResolveRef ./...        # a single test by name
go test -cover ./...                     # with coverage
go test -v ./internal/search/...         # verbose
```

Tests live next to the code they cover (`*_test.go`). When you add or change
behavior, add or update tests in the relevant package.

## Linting

CI runs `golangci-lint` with the config in [`.golangci.yml`](.golangci.yml)
(`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`). Run it locally
before pushing:

```bash
golangci-lint run
```

Also keep the tree gofmt-clean:

```bash
gofmt -l .          # lists files needing formatting (should print nothing)
go vet ./...
```

## Commit messages

This project follows [Conventional Commits](https://www.conventionalcommits.org/)
(enforced via [commitizen](https://commitizen-tools.github.io/commitizen/), see
[`.cz.toml`](.cz.toml)). Versioning is semver, and `CHANGELOG.md` / `VERSION` are
bumped from commit history.

Format:

```
<type>: <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. Examples from
the history:

```
feat: show --expand/--depth to inline $ref schemas
fix: expand $ref schemas in Swagger 2.0 operations
docs: simplify README installation to curl install only
```

## Submitting a pull request

1. Branch off `main` (e.g. `feat/my-change` or `fix/my-bug`).
2. Make your change with accompanying tests.
3. Make sure everything passes locally:
   ```bash
   gofmt -l .
   go vet ./...
   golangci-lint run
   go test -race -count=1 ./...
   ```
4. Push and open a PR against `main`. CI (`lint` + `test`) must pass.

## Releases

Releases are automated by [`release.yml`](.github/workflows/release.yml): pushing
a `v*` tag cross-compiles binaries for linux/darwin on amd64/arm64, packages them
as `.tar.gz`, and publishes a GitHub release with generated notes. The tag's
version (minus the `v`) is stamped into the binary via ldflags. Maintainers
typically bump with `cz bump`, which updates `VERSION` and `CHANGELOG.md`.
