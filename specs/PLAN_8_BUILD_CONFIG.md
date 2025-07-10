# Plan 8: Build Configuration

## Overview
Create build configuration templates including Makefile with container-based commands, go.mod template, and project configuration files.

## Tasks

### 8.1 Makefile

Location: `internal/generator/templates/Makefile.tmpl`

```makefile
# {{.AppName}} Makefile
# All commands run in containers for consistency

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: dev
dev: ## Start development server with hot reload
	docker-compose up dev

.PHONY: up
up: ## Start all services in background
	docker-compose up -d

.PHONY: down
down: ## Stop all services
	docker-compose down

.PHONY: logs
logs: ## Show logs from all services
	docker-compose logs -f

.PHONY: ps
ps: ## Show running services
	docker-compose ps

.PHONY: build
build: ## Build the application
	docker-compose run --rm dev go build -v ./cmd/{{.AppName}}

.PHONY: test
test: ## Run all tests
	docker-compose run --rm -e GO_ENV=test dev go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and show coverage report
	docker-compose run --rm dev go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: ## Run linter
	docker-compose run --rm dev golangci-lint run

.PHONY: fmt
fmt: ## Format code
	docker-compose run --rm dev go fmt ./...

.PHONY: vet
vet: ## Run go vet
	docker-compose run --rm dev go vet ./...

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	docker-compose run --rm dev go mod tidy

.PHONY: mod-download
mod-download: ## Download go modules
	docker-compose run --rm dev go mod download

.PHONY: sqlc
sqlc: ## Generate SQLc code
	docker-compose run --rm --profile tools sqlc

.PHONY: migrate-create
migrate-create: ## Create a new migration (usage: make migrate-create name=create_users_table)
	@if [ -z "$(name)" ]; then echo "Error: name is required. Usage: make migrate-create name=migration_name"; exit 1; fi
	docker-compose run --rm dev migrate create -ext sql -dir internal/database/migrations -seq $(name)

.PHONY: migrate-up
migrate-up: ## Run all pending migrations
	docker-compose run --rm --profile tools migrate

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	docker-compose run --rm dev ./cmd/{{.AppName}}/{{.AppName}} migrate down 1

.PHONY: migrate-status
migrate-status: ## Show migration status
	docker-compose run --rm dev ./cmd/{{.AppName}}/{{.AppName}} migrate status

.PHONY: db-reset
db-reset: ## Reset database (drop, create, migrate)
	docker-compose down -v
	docker-compose up -d db
	@echo "Waiting for database to be ready..."
	@sleep 5
	$(MAKE) migrate-up

.PHONY: db-seed
db-seed: ## Seed database with test data
	docker-compose run --rm dev go run ./cmd/seed

.PHONY: shell
shell: ## Open a shell in the dev container
	docker-compose run --rm dev /bin/bash

.PHONY: psql
psql: ## Open PostgreSQL shell
	docker-compose exec db psql -U postgres -d {{.AppName}}_dev

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf ./cmd/{{.AppName}}/{{.AppName}} coverage.out coverage.html

.PHONY: docker-build
docker-build: ## Build production Docker image
	docker build -t {{.AppName}}:latest .

.PHONY: docker-run
docker-run: ## Run production Docker image
	docker run -p 8080:8080 \
		-e DB_HOST=host.docker.internal \
		-e DB_PORT=5432 \
		-e DB_NAME={{.AppName}}_dev \
		-e DB_USER=postgres \
		-e DB_PASSWORD=postgres \
		-e DB_SSLMODE=disable \
		{{.AppName}}:latest

.PHONY: install-tools
install-tools: ## Install development tools locally
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/cespare/reflex@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: verify
verify: fmt vet lint test ## Run all checks (format, vet, lint, test)
	@echo "All checks passed!"

# Database tasks for CI/CD
.PHONY: ci-test
ci-test: ## Run tests in CI environment
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: ci-lint
ci-lint: ## Run linter in CI environment
	golangci-lint run

.DEFAULT_GOAL := help
```

### 8.2 README Template

Location: `internal/generator/templates/README.md.tmpl`

```markdown
# {{.AppName}}

{{.Description}}

## Quick Start

```bash
# Start development environment
make dev

# Run migrations
make migrate-up

# Run tests
make test
```

## Development

This project uses container-based development. All commands should be run through the Makefile.

### Prerequisites

- Docker and Docker Compose
- Make

### Common Commands

- `make dev` - Start development server with hot reload
- `make test` - Run all tests
- `make lint` - Run linter
- `make migrate-create name=<migration_name>` - Create a new migration
- `make psql` - Open PostgreSQL shell

## API Documentation

The API follows the JustiFi API specification with envelope responses and cursor-based pagination.

### Endpoints

- `GET /api/v1/health` - Health check
- `GET /api/v1/{{.domain_plural}}` - List {{.domain_plural}}
- `POST /api/v1/{{.domain_plural}}` - Create {{.domain}}
- `GET /api/v1/{{.domain_plural}}/:id` - Get {{.domain}}
- `PATCH /api/v1/{{.domain_plural}}/:id` - Update {{.domain}}
- `DELETE /api/v1/{{.domain_plural}}/:id` - Delete {{.domain}}

## Configuration

Configuration is done through environment variables only (12-factor app methodology).

See `.env.example` for available configuration options.

## Testing

Tests use standard Go testing with testify assertions:

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

## Deployment

Build the production Docker image:

```bash
make docker-build
```
```

### 8.3 Gitignore

Location: `internal/generator/templates/.gitignore.tmpl`

```gitignore
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage.html
coverage.txt

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment files
.env
.env.local
.env.*.local

# IDE files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Build output
/cmd/{{.AppName}}/{{.AppName}}
/dist/
/build/

# Database
*.db
*.sqlite
*.sqlite3

# Logs
*.log
logs/

# Temporary files
tmp/
temp/

# Generated files
/internal/repository/sqlc/

# Docker volumes (local development)
.docker/

# Test data
testdata/output/

# Profiling data
*.prof
*.mem
*.cpu

# Certificates (for local development)
*.pem
*.key
*.crt

# Documentation build
docs/_build/
```

### 8.4 Editor Config

Location: `internal/generator/templates/.editorconfig.tmpl`

```ini
# EditorConfig is awesome: https://EditorConfig.org

# top-most EditorConfig file
root = true

# Unix-style newlines with a newline ending every file
[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8
indent_style = tab
indent_size = 4
trim_trailing_whitespace = true

# YAML files
[*.{yml,yaml}]
indent_style = space
indent_size = 2

# Markdown files
[*.md]
trim_trailing_whitespace = false

# Makefile
[Makefile]
indent_style = tab

# Go mod files
[go.{mod,sum}]
indent_style = tab

# SQL files
[*.sql]
indent_style = space
indent_size = 2

# JSON files
[*.json]
indent_style = space
indent_size = 2
```

### 8.5 GolangCI Lint Configuration

Location: `internal/generator/templates/.golangci.yml.tmpl`

```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - typecheck
    - unused
    - gosimple
    - gosec
    - unconvert
    - dupl
    - misspell
    - nakedret
    - prealloc
    - scopelint
    - gocritic
    - gochecknoinits
    - gocyclo
    - gocognit
    - godox
    - funlen
    - whitespace
    - wsl
    - goprintffuncname
    - gomnd
    - goerr113
    - gomodguard
    - nestif
    - exportloopref
    - exhaustive
    - sqlclosecheck
    - nolintlint

linters-settings:
  gocyclo:
    min-complexity: 15
  dupl:
    threshold: 100
  funlen:
    lines: 100
    statements: 50
  goconst:
    min-len: 2
    min-occurrences: 2
  misspell:
    locale: US
  lll:
    line-length: 140
  goimports:
    local-prefixes: {{.ModuleName}}
  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gomnd
        - funlen
        - dupl
    - path: cmd/
      linters:
        - gochecknoinits
```

## Dependencies
- Make
- Docker and Docker Compose
- Go 1.23
- golangci-lint

## Success Criteria
- All commands work through containers
- Build process is reproducible
- Linting configuration catches common issues
- Easy database management commands
- CI-friendly commands available
- Help documentation in Makefile