## v0.5.0 (2026-07-09)

### Feat

- publish multi-arch Docker images with embedded version (#11)
- add serve HTTP API and Docker image (#10)

## v0.4.0 (2026-06-14)

### Feat

- smarter curl — operation IDs, auto host, pretty-print (#9)

## v0.3.0 (2026-06-12)

### Feat

- add dump and compare commands (#8)

## v0.2.0 (2026-06-03)

### Feat

- scoped search with --in (paths, schemas) (#4)
- explain command + per-spec cheat sheet in list (#6)
- show --expand/--depth to inline $ref schemas (#5)
- schema resolution engine (schema + has commands) (#3)

## v0.1.2 (2026-05-31)

## v0.1.1 (2026-05-31)

First public release.

### Features

- CLI to cache OpenAPI specs locally by `key@version`
- `search`, `show`, and `gather` commands for slim or detailed endpoint lookup
- `refresh` to check upstream spec versions
- Default to the latest stored version when a ref omits `@version`

### Infrastructure

- CI lint and test workflow
- Tag-triggered GitHub release with cross-platform binaries
- Commitizen versioning via `cz bump`

## v0.1.0 (2026-05-31)

Initial development preview.
