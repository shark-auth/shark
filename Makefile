.PHONY: build test lint vet run clean docker

BINARY := sharkauth
PKG    := ./cmd/shark

build:
	go build -ldflags="-s -w" -o $(BINARY) $(PKG)

test:
	go test -race -count=1 ./...

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
