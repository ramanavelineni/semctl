BINARY   := semctl
MODULE   := github.com/ramanavelineni/semctl
VERSION  ?= dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS  := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: all build install clean test test-v fmt vet lint check generate help

all: check build ## Run checks and build (default)

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: ## Install to $GOPATH/bin
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" .

clean: ## Remove build artifacts
	rm -f $(BINARY)

test: ## Run all tests
	go test ./...

test-v: ## Run all tests with verbose output
	go test -v ./...

fmt: ## Format all Go source files
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (install: https://golangci-lint.run/welcome/install)
	golangci-lint run ./...

check: fmt vet ## Format, vet (run lint separately if golangci-lint is installed)

generate: ## Regenerate the Semaphore API client (requires Docker)
	./scripts/generate-api.sh

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
