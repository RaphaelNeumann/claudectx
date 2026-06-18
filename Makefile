BINARY  := claudectx
PKG     := github.com/raphaelneumann/claudectx
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/cmd.version=$(VERSION)

.PHONY: build test vet fmt fmt-check install clean snapshot release-check tidy

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format the code
	gofmt -w .

fmt-check: ## Fail if code is not gofmt-clean
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

tidy: ## Tidy modules
	go mod tidy

install: ## Install to GOBIN
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" .

snapshot: ## Build a local release snapshot (no publish)
	goreleaser release --snapshot --clean

release-check: ## Validate the goreleaser config
	goreleaser check

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist
