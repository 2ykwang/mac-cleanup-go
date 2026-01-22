# Variables
BINARY_NAME := mac-cleanup
BINARY_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO ?= go
GOFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
.DEFAULT_GOAL := help

##@ Build

.PHONY: build
build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) .

.PHONY: build-dev
build-dev: ## Build without version info (faster)
	$(GO) build -o $(BINARY_DIR)/$(BINARY_NAME) .

.PHONY: run
run: ## Run the application (with DEBUG logging)
	DEBUG=true $(GO) run .

.PHONY: run-release
run-release: ## Run without debug logging
	$(GO) run .

##@ Development

.PHONY: fmt
fmt: ## Format code
	golangci-lint run --fix

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: tidy
tidy: ## Run go mod tidy
	$(GO) mod tidy

##@ Testing

.PHONY: test
test: ## Run tests (pretty output with gotestsum)
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format pkgname; \
	else \
		$(GO) test ./...; \
	fi

.PHONY: test-watch
test-watch: ## Run tests in watch mode
	gotestsum --watch --format pkgname

.PHONY: test-v
test-v: ## Run tests with verbose output
	@if command -v gotestsum >/dev/null 2>&1; then \
		gotestsum --format standard-verbose; \
	else \
		$(GO) test -v ./...; \
	fi

.PHONY: test-cover
test-cover: ## Run tests with coverage
	$(GO) test -cover ./...

.PHONY: patch-cover
patch-cover: ## Report patch coverage against origin/main (set BASE=...)
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) run ./scripts/patch_cover.go --base=$${BASE:-origin/main} --profile=coverage.out

.PHONY: patch-cover-worktree
patch-cover-worktree: ## Report patch coverage including worktree changes (set BASE=...)
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) run ./scripts/patch_cover.go --base=$${BASE:-origin/main} --profile=coverage.out --worktree

.PHONY: test-cover-html
test-cover-html: ## Generate HTML coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

PERF_COUNT ?= 1
ifneq ($(filter test-perf,$(MAKECMDGOALS)),)
PERF_ARGS := $(filter-out test-perf,$(MAKECMDGOALS))
ifneq ($(strip $(PERF_ARGS)),)
PERF_COUNT := $(firstword $(PERF_ARGS))
endif
$(PERF_ARGS):
	@:
endif

.PHONY: test-perf
test-perf: ## Run performance benchmarks (requires -tags=perf, set count via `make test-perf 3`)
	@set -e; \
	tmp=$$(mktemp -t bench.XXXXXX); \
	trap 'rm -f $$tmp' EXIT; \
	$(GO) test -tags=perf -bench=. -benchmem -run=^$$ -count=$(PERF_COUNT) ./internal/target/... | tee $$tmp; \
	if command -v benchstat >/dev/null 2>&1; then \
		echo ""; \
		benchstat $$tmp; \
	else \
		echo ""; \
		echo "benchstat not found; install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

##@ Cleanup

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BINARY_DIR)/$(BINARY_NAME)
	rm -f coverage.out coverage.html

##@ Helpers

.PHONY: deps
deps: ## Download dependencies
	$(GO) mod download

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
