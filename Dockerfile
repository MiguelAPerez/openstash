FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# vite.config.ts writes to ../internal/server/web/dist
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /internal/server/web/dist ./internal/server/web/dist
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X github.com/MiguelAPerez/openstash/internal/cli.version=${VERSION}" -o /openstash ./cmd/openstash

FROM gcr.io/distroless/static-debian12
COPY --from=build /openstash /openstash
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/openstash", "serve", "--store", "/data", "--addr", ":8080"]
