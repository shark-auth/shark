# SharkAuth Unified Makefile
# Supports Windows (Git Bash/MSYS2) and Linux

BINARY_NAME := shark
ifeq ($(OS),Windows_NT)
    BINARY := $(BINARY_NAME).exe
    # Use standard shell commands available in Git Bash/MSYS2
    RM := rm -f
    RMDIR := rm -rf
else
    BINARY := $(BINARY_NAME)
    RM := rm -f
    RMDIR := rm -rf
endif

PKG     := ./cmd/shark
LDFLAGS := -s -w -X github.com/shark-auth/shark/internal/version.Version=0.1.0 -X "github.com/shark-auth/shark/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)" -X "github.com/shark-auth/shark/internal/version.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
GOFLAGS := -trimpath -ldflags="$(LDFLAGS)"

.PHONY: all build test verify verify-full lint run clean frontend-build binary-build docker

all: build

# Frontend installation
frontend-install:
	@echo ">> installing frontend dependencies"
	@cd admin && npm install

# Frontend build
frontend-build:
	@echo ">> building frontend (admin)"
	@cd admin && $(if $(filter Windows_NT,$(OS)),npm run build,NODE_OPTIONS=--max-old-space-size=4096 pnpm build)

# Binary build
# binary-build now depends on frontend-build to ensure assets are always up to date
binary-build: frontend-build
	@echo ">> building binary -> $(BINARY)"
	go build $(GOFLAGS) -o $(BINARY) $(PKG)

build: binary-build

test:
	go test -race -count=1 ./...

# Basic verification
verify:
	go vet ./...
	go test -count=1 ./...

# Full verification for release/CI
# Integration and E2E tests are often skipped on local Windows dev unless specifically requested
verify-full: verify
	go test -race -count=1 -tags=integration ./...
	$(if $(filter Windows_NT,$(OS)),,@go test -race -count=1 -tags=e2e ./internal/testutil/e2e/...)

lint:
	@which golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

run: build
	./$(BINARY) serve --dev $(if $(filter Windows_NT,$(OS)),,--proxy-upstream http://localhost:3000)

clean:
	@echo ">> cleaning"
	$(RM) $(BINARY)
	$(RMDIR) internal/admin/dist
	$(RM) sharkauth.exe $(BINARY_NAME)

docker:
	docker build -t sharkauth .
