# Frontend build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /src
RUN npm install -g pnpm

# Only copy what's needed for workspace dependency resolution
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY admin/package.json ./admin/
RUN pnpm install

# Build frontend
COPY admin/ ./admin/
RUN pnpm -C admin build

# Go build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
# Copy production assets from frontend-builder
COPY --from=frontend-builder /src/internal/admin/dist ./internal/admin/dist

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o sharkauth ./cmd/shark

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S shark && adduser -S shark -G shark

WORKDIR /app

COPY --from=builder /build/sharkauth .

RUN mkdir -p /app/data && chown -R shark:shark /app

USER shark

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["./sharkauth"]
