# Variables
BINARY_NAME := mac-cleanup
BINARY_DIR := bin
CMD_DIR := ./cmd/mac-cleanup
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO ?= go
GOFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
.DEFAULT_GOAL := help

##@ Build

.PHONY: build
build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-dev
build-dev: ## Build without version info (faster)
	$(GO) build -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)

.PHONY: run
run: ## Run the application
	$(GO) run $(CMD_DIR)

##@ Development

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...

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
test: ## Run tests
	$(GO) test ./...

.PHONY: test-v
test-v: ## Run tests with verbose output
	$(GO) test -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	$(GO) test -cover ./...

.PHONY: test-cover-html
test-cover-html: ## Generate HTML coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

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
