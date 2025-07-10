# Plan 7: Container Setup

## Overview
Create Docker and container configuration templates for development environment with reflex hot reload, PostgreSQL, and all necessary tools.

## Tasks

### 7.1 Development Dockerfile

Location: `internal/generator/templates/Dockerfile.dev.tmpl`

```dockerfile
FROM golang:1.23-alpine AS dev

# Install system dependencies
RUN apk add --no-cache \
    git \
    make \
    curl \
    postgresql-client \
    bash

# Install development tools
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/cespare/reflex@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Default command uses reflex for hot reload
CMD ["reflex", "-c", ".reflex.conf"]
```

### 7.2 Docker Compose Configuration

Location: `internal/generator/templates/docker-compose.yml.tmpl`

```yaml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: {{.AppName}}_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  dev:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: dev
    volumes:
      - .:/app
      - go_cache:/go/pkg/mod
    ports:
      - "8080:8080"
    environment:
      # Database configuration
      DB_HOST: db
      DB_PORT: 5432
      DB_NAME: {{.AppName}}_dev
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_SSLMODE: disable
      
      # Application configuration
      PORT: 8080
      HOST: 0.0.0.0
      LOG_LEVEL: debug
      LOG_FORMAT: text
      
      # Development mode
      GO_ENV: development
    depends_on:
      db:
        condition: service_healthy
    command: ["reflex", "-c", ".reflex.conf"]

  # Migration runner service
  migrate:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: dev
    volumes:
      - .:/app
    environment:
      DB_HOST: db
      DB_PORT: 5432
      DB_NAME: {{.AppName}}_dev
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_SSLMODE: disable
    depends_on:
      db:
        condition: service_healthy
    profiles:
      - tools
    command: ["./cmd/{{.AppName}}/{{.AppName}}", "migrate", "up"]

  # SQLc code generator service
  sqlc:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: dev
    volumes:
      - .:/app
    profiles:
      - tools
    command: ["sqlc", "generate"]

  # Test runner service
  test:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: dev
    volumes:
      - .:/app
      - go_cache:/go/pkg/mod
    environment:
      # Test database configuration
      DB_HOST: db
      DB_PORT: 5432
      DB_NAME: {{.AppName}}_test
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_SSLMODE: disable
      
      # Test environment
      GO_ENV: test
    depends_on:
      db:
        condition: service_healthy
    profiles:
      - test
    command: ["go", "test", "-v", "./..."]

volumes:
  postgres_data:
  go_cache:
```

### 7.3 Reflex Configuration

Location: `internal/generator/templates/.reflex.conf.tmpl`

```conf
# Reflex configuration for hot reload

# Watch for changes in Go files
-r '\.go$' -s -- sh -c 'go build -o ./cmd/{{.AppName}}/{{.AppName}} ./cmd/{{.AppName}} && ./cmd/{{.AppName}}/{{.AppName}} serve'

# Exclude vendor and test files
-R '^vendor/' -R '_test\.go$'

# Also watch for changes in SQL files (rebuild SQLc)
-r '\.sql$' -s -- sh -c 'sqlc generate && go build -o ./cmd/{{.AppName}}/{{.AppName}} ./cmd/{{.AppName}} && ./cmd/{{.AppName}}/{{.AppName}} serve'
```

### 7.4 Environment Example

Location: `internal/generator/templates/.env.example.tmpl`

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME={{.AppName}}_dev
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable

# Server Configuration
PORT=8080
HOST=0.0.0.0

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Environment
GO_ENV=production

# Optional: External Services
# REDIS_URL=redis://localhost:6379
# AWS_REGION=us-east-1
# AWS_ACCESS_KEY_ID=
# AWS_SECRET_ACCESS_KEY=

# Optional: Feature Flags
# FEATURE_NEW_UI=false
# FEATURE_BETA_API=false

# Optional: Rate Limiting
# RATE_LIMIT_ENABLED=true
# RATE_LIMIT_REQUESTS_PER_MINUTE=60

# Optional: CORS
# CORS_ALLOWED_ORIGINS=https://example.com,https://app.example.com
# CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
# CORS_ALLOWED_HEADERS=Content-Type,Authorization
```

### 7.5 Docker Ignore

Location: `internal/generator/templates/.dockerignore.tmpl`

```
# Git
.git
.gitignore

# Go
vendor/
*.test
*.out
coverage.html
coverage.txt

# Build artifacts
cmd/{{.AppName}}/{{.AppName}}
dist/
build/

# Development
.env
.env.local
*.log
.DS_Store

# IDE
.idea/
.vscode/
*.swp
*.swo

# Documentation
*.md
docs/
```

### 7.6 Production Dockerfile

Location: `internal/generator/templates/Dockerfile.tmpl`

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o {{.AppName}} ./cmd/{{.AppName}}

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/{{.AppName}} .
COPY --from=builder /app/internal/database/migrations ./internal/database/migrations

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./{{.AppName}}", "serve"]
```

## Dependencies
- Docker and Docker Compose
- Go 1.23
- PostgreSQL 16
- reflex for hot reload
- golang-migrate
- sqlc

## Success Criteria
- Container starts with single `docker-compose up`
- Hot reload works for Go and SQL file changes
- All development tools available in container
- Health checks ensure proper startup order
- Production image is minimal and secure
- Environment variables properly configured