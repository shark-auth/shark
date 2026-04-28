.PHONY: build test lint vet run clean docker verify frontend-build

BINARY := shark
PKG    := ./cmd/shark

frontend-build:
	@cd admin && NODE_OPTIONS=--max-old-space-size=4096 pnpm build

build: frontend-build
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
	./$(BINARY) serve --dev --proxy-upstream http://localhost:3000

clean:
	rm -f $(BINARY)
	rm -rf internal/admin/dist

docker:
	docker build -t sharkauth .
