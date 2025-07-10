# go-app-gen Makefile

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Build
.PHONY: build
build: ## Build the go-app-gen binary
	go build -v -o bin/go-app-gen ./cmd/go-app-gen

.PHONY: install
install: ## Install go-app-gen to $GOPATH/bin
	go install ./cmd/go-app-gen

## Test & Quality
.PHONY: test
test: ## Run all tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and show coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt vet lint test ## Run all checks (format, vet, lint, test)
	@echo "All checks passed!"

## E2E Testing
.PHONY: e2e
e2e: build ## Run end-to-end tests (generate app and test it)
	@echo "Running e2e tests..."
	./scripts/e2e-test.sh

## Utilities
.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html test-output/

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	go mod tidy

.PHONY: mod-download
mod-download: ## Download dependencies
	go mod download

.DEFAULT_GOAL := help