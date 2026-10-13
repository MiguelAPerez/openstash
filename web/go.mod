// The web UI is not Go code. This file makes web/ its own module so that
// `go test ./...` and golangci-lint skip it (node_modules can contain .go files).
module github.com/MiguelAPerez/openstash/web

go 1.22.0
