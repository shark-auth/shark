# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath \
    -ldflags="-s -w -X github.com/shark-auth/shark/internal/version.Version=${VERSION} -X github.com/shark-auth/shark/internal/version.Commit=${COMMIT} -X github.com/shark-auth/shark/internal/version.BuildTime=${BUILD_TIME}" \
    -o /out/sharkauth ./cmd/shark

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1000 shark && \
    useradd -r -u 1000 -g shark shark

WORKDIR /app

COPY --from=build /out/sharkauth /app/sharkauth
RUN mkdir -p /app/data && chown -R shark:shark /app && chmod +x /app/sharkauth

USER shark

EXPOSE 8080

ENTRYPOINT ["./sharkauth", "serve"]
