.PHONY: build test lint vet run clean docker verify

BINARY := sharkauth
PKG    := ./cmd/shark

build:
	go build -ldflags="-s -w" -o $(BINARY) $(PKG)

test:
	go test -race -count=1 ./...

# verify: runs vet + unit + integration + e2e (strict superset of test).
# CI and pre-release gate. Use `make test` for fast iteration.
verify:
	go vet ./...
	go test -race -count=1 ./...
	go test -race -count=1 -tags=integration ./...
	go test -race -count=1 -tags=e2e ./internal/testutil/e2e/...

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

docker:
	docker build -t sharkauth .
