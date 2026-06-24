FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /openstash ./cmd/openstash

FROM gcr.io/distroless/static-debian12
COPY --from=build /openstash /openstash
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/openstash", "serve", "--store", "/data", "--addr", ":8080"]
