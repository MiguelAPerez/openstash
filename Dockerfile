FROM --platform=$BUILDPLATFORM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X github.com/MiguelAPerez/openstash/internal/cli.version=${VERSION}" -o /openstash ./cmd/openstash

FROM gcr.io/distroless/static-debian12
COPY --from=build /openstash /openstash
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/openstash", "serve", "--store", "/data", "--addr", ":8080"]
