# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o sharkauth ./cmd/shark

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S shark && adduser -S shark -G shark

WORKDIR /app

COPY --from=builder /build/sharkauth .

RUN mkdir -p /app/data && chown -R shark:shark /app

USER shark

EXPOSE 8080

ENTRYPOINT ["./sharkauth"]
