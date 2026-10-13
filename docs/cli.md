# CLI reference

Every command prints JSON, so you can read it, pipe it to `jq`, or hand it to an agent as-is. Run `openstash <command> --help` for the full flag list.

- [Specs and versions](#specs-and-versions)
- [Add and list specs](#add-and-list-specs): `add`, `list`, `refresh`
- [Find endpoints](#find-endpoints): `explain`, `search`, `show`, `gather`
- [Inspect schemas](#inspect-schemas): `schema`, `has`
- [Whole specs and diffs](#whole-specs-and-diffs): `dump`, `compare`
- [Call an endpoint](#call-an-endpoint): `curl`
- [Serve the web UI and HTTP API](#serve-the-web-ui-and-http-api): `serve`
- [Storage](#storage)

## Specs and versions

Every stored spec has a **key** (a name you choose, like `gitea`) and a **version**. Refer to one as `key@version`, or just `key` for the latest stored version:

```bash
openstash search gitea "user repos"          # latest gitea
openstash search gitea@1.22.0 "user repos"   # a specific version
```

Versions sort by semver when they parse, so `1.10.0` is newer than `1.9.0`.

## Add and list specs

### add

Fetch a spec from a URL or read it from a file. JSON and YAML both work.

```bash
openstash add petstore --from https://petstore3.swagger.io/api/v3/openapi.json
openstash add gitea --from ./swagger.json --version 1.22.0 --endpoint https://gitea.example/api/v1
```

- `--version` defaults to the spec's `info.version`.
- `--endpoint` records the API's base URL, which `curl` uses.

Adding a new version under an existing key keeps the old ones. That's what builds the [timeline](web-ui.md#timeline) in the web UI.

### list

```bash
openstash list
```

Shows every stored `key@version` with its source, endpoint, fetch time and a few example commands to try.

### refresh

```bash
openstash refresh gitea
```

Re-fetches the source and reports whether `info.version` changed. It doesn't overwrite anything; run `add` to store the new version.

## Find endpoints

### explain

```bash
openstash explain gitea
```

A quick start for one spec: operation and schema counts, the most common tags, and ready-to-run commands built from what's actually in it. A good first call for an agent meeting a new API.

### search

A short, ranked list of matching endpoints.

```bash
openstash search gitea "user repos"
openstash search gitea "repos" --path-prefix /user --method GET
openstash search gitea "Repository" --in schemas
```

- `--in` picks what to search: `paths` (default), `schemas`, `descriptions`. Repeat it to search several.
- `--limit` sets the number of results (default 5).

### show

Full detail for one operation.

```bash
openstash show gitea --method GET --path /user/repos
openstash show gitea --method GET --path /user/repos --expand    # inline $ref schemas one level
openstash show gitea --method GET --path /user/repos --depth 2   # or more levels
```

### gather

Search plus full detail for each hit, in one call.

```bash
openstash gather gitea "subscription" --expand
openstash gather gitea@1.22.0 --method GET --path /user/repos
```

Takes the same filters as `search` and the same `--depth`/`--expand` as `show`.

## Inspect schemas

### schema

```bash
openstash schema gitea Repository             # $refs inlined one level
openstash schema gitea Repository --depth 0   # keep $refs as short names
openstash schema gitea Repository --fields    # flat field list
```

### has

Check that a schema or a field path exists before relying on it.

```bash
openstash has gitea Repository.owner.login
```

## Whole specs and diffs

### dump

The complete stored document.

```bash
openstash dump gitea
openstash dump forgejo@15.0.0 | jq '.info'
```

### compare

Diff operations and schemas between two stored specs. The first argument is the **baseline**, the second the **target**:

- **added**: only in the target
- **removed**: only in the baseline
- **changed**: in both, with differences

```bash
openstash compare gitea@1.21.0 gitea@1.22.0
openstash compare forgejo gitea --brief
openstash compare forgejo gitea --section operations --limit 10
```

The two specs can have different keys. To compare versions of one key in more depth (parameters, request bodies and responses, including changes behind shared `$ref`s), use the web UI's [timeline](web-ui.md#timeline) or the [HTTP API's compare endpoint](serve-api.md#7-version-history).

## Call an endpoint

### curl

Call an operation by its `operationId`. Parameters go in the right place (path, query or body) based on the spec.

```bash
openstash curl gitea --operation issueListIssues \
  --token "$GITEA_TOKEN" \
  --param owner=alice --param repo=myrepo
```

- The host defaults to the `--endpoint` stored with the spec. Override it with `--host`.
- Auth: `--token` (Bearer), or `--username` and `--password` (basic).
- `--pretty-print` formats the JSON response.

## Serve the web UI and HTTP API

```bash
openstash serve                  # http://127.0.0.1:8080
openstash serve --addr :9000
openstash serve --no-ui          # API only
```

See [Web UI](web-ui.md) and [HTTP API](serve-api.md).

## Storage

Specs live in `~/.openstash` by default. Use another directory for one command with `--store`, or for everything with `OPENSTASH_STORE`:

```bash
openstash --store /path/to/store list
export OPENSTASH_STORE=/path/to/store
```

Each version is a folder under `specs/<key>/<version>/` holding the spec (YAML is converted to JSON), a search index and a metadata file.
