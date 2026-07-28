BINARY := bin/dockly
GOFLAGS ?=

.PHONY: build test check run clean docker-build

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build $(GOFLAGS) -trimpath -o $(BINARY) ./cmd/dockly

test:
	go test ./...

check:
	@test -z "$$(gofmt -l .)" || (echo "Run gofmt on:"; gofmt -l .; exit 1)
	go vet ./...
	go test -race ./...
	CGO_ENABLED=0 go build -trimpath -o /tmp/dockly-check ./cmd/dockly
	@rm -f /tmp/dockly-check

run:
	DOCKLY_DATA_DIR="$${DOCKLY_DATA_DIR:-$$(pwd)/.dockly}" go run ./cmd/dockly

docker-build:
	docker build -t dockly:dev .

clean:
	rm -rf bin .dockly
