# Common development tasks. The web UI lives in web/ (React + TypeScript) and is
# built into internal/server/web/dist, which the Go binary embeds.

VERSION ?= $(shell cat VERSION)
LDFLAGS := -s -w -X github.com/MiguelAPerez/openstash/internal/cli.version=$(VERSION)

.PHONY: all build build-go web web-deps web-dev test test-go test-web lint clean

all: build

## build: web UI + binary into bin/openstash
build: web build-go

## build-go: binary only (embeds whatever is in internal/server/web/dist)
build-go:
	go build -ldflags="$(LDFLAGS)" -o bin/openstash ./cmd/openstash

web-deps: web/node_modules

web/node_modules: web/package.json web/package-lock.json
	cd web && npm ci
	@touch $@

## web: typecheck and build the UI into internal/server/web/dist
web: web-deps
	cd web && npm run build

## web-dev: Vite dev server with hot reload, proxying /v1 to openstash serve on :8080
web-dev: web-deps
	cd web && npm run dev

## test: Go and web unit tests
test: test-go test-web

test-go:
	go test -race -count=1 ./...

test-web: web-deps
	cd web && npm test

lint:
	@out=$$(gofmt -l cmd internal test); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	go vet ./...
	golangci-lint run
	cd web && npm run typecheck

clean:
	rm -rf bin
	find internal/server/web/dist -mindepth 1 ! -name .gitkeep -delete
