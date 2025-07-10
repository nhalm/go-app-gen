# Agent Implementation Prompts

Here are the prompts you should use for each agent to implement the plans:

## Agent 1: Core Infrastructure
```
Please implement PLAN_1_CORE_INFRASTRUCTURE.md for the go-app-gen project. 

Read the existing generator.go file and update it according to the plan:
1. Add the runCommand helper method to execute shell commands
2. Add the PostProcess method that runs after template generation
3. Update the TemplateData struct with new fields (DomainLower, GoVersion)
4. Update the Generate method to call PostProcess
5. Update getOutputPath to handle new placeholders

Make sure to preserve all existing functionality while adding these enhancements. The goal is to enable container-based validation after generating templates.
```

## Agent 2: CLI Commands
```
Please implement PLAN_2_CLI_COMMANDS.md for the go-app-gen project.

Create the following template files in internal/generator/templates/cmd/{{.AppName}}/cmd/:
1. root.go.tmpl - Remove all viper config file support, use only environment variables
2. serve.go.tmpl - HTTP server with slog logging and dbutil integration
3. migrate.go.tmpl - golang-migrate commands (up, down, status, create)

Follow the specifications in the plan exactly. Key requirements:
- No config files, only environment variables
- Use slog for structured logging
- Graceful shutdown in serve command
- All migrate subcommands working with dbutil.GetDSN()
```

## Agent 3: Database Layer
```
Please implement PLAN_3_DATABASE_LAYER.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/internal/database/migrations/001_initial_schema.up.sql.tmpl
2. internal/generator/templates/internal/database/migrations/001_initial_schema.down.sql.tmpl
3. internal/generator/templates/sqlc.yaml.tmpl
4. internal/generator/templates/internal/repository/queries/{{.domain}}.sql.tmpl
5. internal/generator/templates/internal/repository/repository.go.tmpl

Key requirements:
- All tables must have soft delete (deleted_at) and temporal fields (effective_start/end)
- Repository uses dbutil directly without wrappers
- SQL queries handle cursor pagination
- Proper error handling with dbutil error types
```

## Agent 4: API Layer
```
Please implement PLAN_4_API_LAYER.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/internal/api/types.go.tmpl - JustiFi response types
2. internal/generator/templates/internal/api/handler.go.tmpl - HTTP handlers
3. internal/generator/templates/internal/api/routes.go.tmpl - Route registration

Requirements:
- Follow JustiFi API envelope format exactly
- Implement cursor-based pagination
- Handler defines ServiceInterface (consumer defines interface pattern)
- Proper error responses with request IDs
- Use go-playground/validator for input validation
```

## Agent 5: Service Layer
```
Please implement PLAN_5_SERVICE_LAYER.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/internal/service/models.go.tmpl
2. internal/generator/templates/internal/service/service.go.tmpl
3. internal/generator/templates/internal/service/errors.go.tmpl

Requirements:
- Service defines RepositoryInterface (consumer defines interface)
- Handle model conversions between repository and service layers
- Business logic validation (empty names, date ranges, etc.)
- Proper error wrapping and custom business errors
- Pagination logic with cursor generation
```

## Agent 6: Utilities
```
Please implement PLAN_6_UTILITIES.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/internal/utils/logger.go.tmpl - Request logging middleware
2. internal/generator/templates/internal/utils/cursor.go.tmpl - Cursor pagination helpers
3. internal/generator/templates/internal/utils/helpers.go.tmpl - Common utilities
4. internal/generator/templates/internal/utils/validation.go.tmpl - Validation helpers

Requirements:
- Single-line request logging with slog
- Cursor encoding/decoding for pagination
- Generic helper functions using Go generics where appropriate
- Request ID propagation through context
```

## Agent 7: Container Setup
```
Please implement PLAN_7_CONTAINER_SETUP.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/Dockerfile.dev.tmpl - Development container
2. internal/generator/templates/docker-compose.yml.tmpl - Full stack setup
3. internal/generator/templates/.reflex.conf.tmpl - Hot reload configuration
4. internal/generator/templates/.env.example.tmpl - Environment variables
5. internal/generator/templates/.dockerignore.tmpl
6. internal/generator/templates/Dockerfile.tmpl - Production container

Requirements:
- Go 1.23 base image
- PostgreSQL 16 with health checks
- Reflex hot reload for .go and .sql files
- All tools installed in dev container
- Minimal production image
```

## Agent 8: Build Configuration
```
Please implement PLAN_8_BUILD_CONFIG.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/Makefile.tmpl - Container-based commands
2. internal/generator/templates/README.md.tmpl - Project documentation
3. internal/generator/templates/.gitignore.tmpl
4. internal/generator/templates/.editorconfig.tmpl
5. internal/generator/templates/.golangci.yml.tmpl - Linter configuration

Requirements:
- All Makefile commands run in containers
- Help documentation for all commands
- Comprehensive gitignore
- Strict linting rules
```

## Agent 9: Testing Infrastructure
```
Please implement PLAN_9_TESTING.md for the go-app-gen project.

Create the following template files:
1. internal/generator/templates/internal/testutil/helpers.go.tmpl - Test utilities
2. internal/generator/templates/internal/repository/repository_test.go.tmpl
3. internal/generator/templates/internal/service/service_test.go.tmpl
4. internal/generator/templates/internal/api/handler_test.go.tmpl
5. internal/generator/templates/internal/integration/setup_test.go.tmpl

Requirements:
- Standard Go testing (no testify/suite)
- Use testify for assertions only
- Repository tests use dbutil.RequireTestDB
- Service tests use mocks
- API tests verify HTTP behavior
- Integration test setup available
```

## General Instructions for All Agents
```
Additional context:
- The "domain" placeholder represents the main entity (e.g., "product", "user")
- All templates should use Go 1.23 features where appropriate
- Follow the exact specifications in your assigned plan document
- Create directories as needed for the templates
- Ensure all placeholders ({{.AppName}}, {{.domain}}, etc.) are used correctly
- Test that the generated code would compile (syntax-wise)
```