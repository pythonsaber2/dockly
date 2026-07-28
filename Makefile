BINARY := bin/dockly
SITE_BINARY := bin/dockly-site
GOFLAGS ?=

.PHONY: build site-build test check run site-run clean docker-build

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build $(GOFLAGS) -trimpath -o $(BINARY) ./cmd/dockly

site-build:
	@mkdir -p bin
	CGO_ENABLED=0 go build $(GOFLAGS) -trimpath -o $(SITE_BINARY) ./cmd/dockly-site

test:
	go test ./...

check:
	@test -z "$$(gofmt -l .)" || (echo "Run gofmt on:"; gofmt -l .; exit 1)
	go vet ./...
	go test -race ./...
	python3 site/scripts/check-site.py
	node --check internal/server/web/app.js
	node --check site/script.js
	node --check site/site.js
	CGO_ENABLED=0 go build -trimpath -o /tmp/dockly-check ./cmd/dockly
	CGO_ENABLED=0 go build -trimpath -o /tmp/dockly-site-check ./cmd/dockly-site
	@rm -f /tmp/dockly-check
	@rm -f /tmp/dockly-site-check

run:
	DOCKLY_DATA_DIR="$${DOCKLY_DATA_DIR:-$$(pwd)/.dockly}" go run ./cmd/dockly

site-run:
	DOCKLY_SITE_ROOT="$${DOCKLY_SITE_ROOT:-$$(pwd)/site}" go run ./cmd/dockly-site

docker-build:
	docker build -t dockly:dev .

clean:
	rm -rf bin .dockly
