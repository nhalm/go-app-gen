# Go App Generator Enhancement - Task Overview

This document provides an overview of all the enhancement tasks, broken down into separate plan files for parallel work.

## Task Breakdown

### 1. Core Infrastructure (`PLAN_1_CORE_INFRASTRUCTURE.md`)
- Update generator.go with runCommand helper and PostProcess method
- Add container-based validation
- Go version: 1.23 (latest stable)

### 2. CLI Commands (`PLAN_2_CLI_COMMANDS.md`)
- root.go.tmpl - Environment-only configuration (no config files)
- serve.go.tmpl - HTTP server with slog and request logging
- migrate.go.tmpl - golang-migrate integration

### 3. Database Layer (`PLAN_3_DATABASE_LAYER.md`)
- Initial migration templates with soft delete and temporal fields
- SQLc configuration and query templates
- Repository implementation using dbutil directly

### 4. API Layer (`PLAN_4_API_LAYER.md`)
- JustiFi response format (envelope pattern)
- Cursor-based pagination
- Handler templates with CRUD operations
- Service interface definition

### 5. Service Layer (`PLAN_5_SERVICE_LAYER.md`)
- Business logic implementation
- Repository interface definition
- Model conversions between layers

### 6. Utilities (`PLAN_6_UTILITIES.md`)
- Request logger (single-line slog)
- Cursor pagination utilities
- Model conversion helpers

### 7. Container Setup (`PLAN_7_CONTAINER_SETUP.md`)
- Dockerfile.dev with Go 1.23 and tools
- docker-compose.yml with reflex hot reload
- .reflex.conf configuration
- .env.example template

### 8. Build Configuration (`PLAN_8_BUILD_CONFIG.md`)
- Makefile with container-based commands
- go.mod minimal template
- .gitignore template

### 9. Testing Infrastructure (`PLAN_9_TESTING.md`)
- Integration test templates using testify
- Repository tests with dbutil.RequireTestDB
- API handler tests
- No testify/suite - standard Go testing

## Key Principles
- 12-factor methodology (environment-only config)
- Container-based development
- Soft delete pattern (deleted_at timestamps)
- Temporal fields (effective_start/end)
- Layered models with conversions
- Consumer defines interfaces
- Direct dbutil usage (no wrappers)
- JustiFi API compliance

## Success Criteria
- Generated app runs immediately with `make dev`
- All CRUD operations work with soft delete
- Cursor pagination functions correctly
- Tests pass in containers
- Single-line request logging works
- Health checks verify connectivity