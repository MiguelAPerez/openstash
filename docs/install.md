# Install

## Homebrew (macOS and Linux)

```bash
brew tap MiguelAPerez/tap
brew install openstash
```

## Pre-built binary

macOS and Linux binaries are on the [releases page](https://github.com/MiguelAPerez/openstash/releases/latest). This one-liner picks the right one for your OS and CPU:

```bash
curl -fsSL https://github.com/MiguelAPerez/openstash/releases/latest/download/openstash_$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz | tar xz
sudo mv openstash /usr/local/bin/
```

Check it worked:

```bash
openstash --version
```

## Docker

Images are published to GitHub Container Registry with each release. The container runs `openstash serve` (the [web UI](web-ui.md) and [HTTP API](serve-api.md)) with the store at `/data`:

```bash
docker run -p 8080:8080 -v ~/.openstash:/data ghcr.io/miguelaperez/openstash:latest
```

Then open http://localhost:8080. Mount your existing `~/.openstash` to see the specs you've already added, or an empty directory to start fresh.

To build and run from a checkout instead:

```bash
docker compose up --build
```

## From source

See [CONTRIBUTING.md](../CONTRIBUTING.md#building). `go build` works without Node; the web UI needs a Node build first (`make build` does both).
