# Plan 3: Database Layer

## Overview
Create database migration templates, SQLc configuration, and repository implementation using dbutil directly.

## Tasks

### 3.1 Initial Migration Templates

Location: `internal/generator/templates/internal/database/migrations/`

#### 001_initial_schema.up.sql.tmpl
```sql
-- Create {{.domain}} table with soft delete and temporal fields
CREATE TABLE IF NOT EXISTS {{.domain_plural}} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Temporal fields for versioning
    effective_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_end TIMESTAMPTZ NOT NULL DEFAULT '9999-12-31 23:59:59Z',
    
    -- Standard timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ -- NULL when not deleted (soft delete)
);

-- Create indexes for performance
CREATE INDEX idx_{{.domain_plural}}_deleted_at ON {{.domain_plural}}(deleted_at);
CREATE INDEX idx_{{.domain_plural}}_effective ON {{.domain_plural}}(effective_start, effective_end);
CREATE INDEX idx_{{.domain_plural}}_created_at ON {{.domain_plural}}(created_at);

-- Add trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_{{.domain_plural}}_updated_at 
    BEFORE UPDATE ON {{.domain_plural}} 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Enhance schema_migrations table if it exists
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'schema_migrations'
    ) THEN
        ALTER TABLE schema_migrations 
        ADD COLUMN IF NOT EXISTS applied_time TIMESTAMPTZ DEFAULT NOW();
    END IF;
END $$;
```

#### 001_initial_schema.down.sql.tmpl
```sql
-- Drop triggers first
DROP TRIGGER IF EXISTS update_{{.domain_plural}}_updated_at ON {{.domain_plural}};

-- Drop the table
DROP TABLE IF EXISTS {{.domain_plural}};

-- Note: We don't remove the applied_time column from schema_migrations
-- as it might affect other migrations
```

### 3.2 SQLc Configuration

Location: `internal/generator/templates/sqlc.yaml.tmpl`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/repository/queries/"
    schema: "internal/database/schema.sql"
    gen:
      go:
        package: "sqlc"
        out: "internal/repository/sqlc"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
        emit_empty_slices: true
```

### 3.3 SQLc Queries

Location: `internal/generator/templates/internal/repository/queries/{{.domain}}.sql.tmpl`

```sql
-- name: Create{{.DomainTitle}} :one
INSERT INTO {{.domain_plural}} (
    name, 
    description,
    effective_start,
    effective_end
) VALUES (
    $1, 
    $2,
    COALESCE($3, NOW()),
    COALESCE($4, '9999-12-31 23:59:59Z')
)
RETURNING *;

-- name: Get{{.DomainTitle}} :one
SELECT * FROM {{.domain_plural}}
WHERE id = $1 
  AND deleted_at IS NULL
  AND NOW() BETWEEN effective_start AND effective_end;

-- name: Get{{.DomainTitle}}ByID :one
SELECT * FROM {{.domain_plural}}
WHERE id = $1 AND deleted_at IS NULL;

-- name: Update{{.DomainTitle}} :one
UPDATE {{.domain_plural}}
SET 
    name = COALESCE($2, name),
    description = COALESCE($3, description),
    updated_at = NOW()
WHERE id = $1 
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDelete{{.DomainTitle}} :exec
UPDATE {{.domain_plural}}
SET 
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 
  AND deleted_at IS NULL;

-- name: List{{.DomainTitle}}s :many
SELECT * FROM {{.domain_plural}}
WHERE deleted_at IS NULL
  AND ($1::timestamptz IS NULL OR created_at < $1)
  AND ($2::uuid IS NULL OR id < $2)
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: List{{.DomainTitle}}sReverse :many
SELECT * FROM {{.domain_plural}}
WHERE deleted_at IS NULL
  AND ($1::timestamptz IS NULL OR created_at > $1)
  AND ($2::uuid IS NULL OR id > $2)
ORDER BY created_at ASC, id ASC
LIMIT $3;

-- name: Count{{.DomainTitle}}s :one
SELECT COUNT(*) FROM {{.domain_plural}}
WHERE deleted_at IS NULL;
```

### 3.4 Repository Implementation

Location: `internal/generator/templates/internal/repository/repository.go.tmpl`

```go
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nhalm/dbutil"
	
	"{{.ModuleName}}/internal/repository/sqlc"
)

// Repository implements database operations for {{.domain_plural}}
type Repository struct {
	conn *dbutil.Connection[*sqlc.Queries]
}

// New creates a new repository instance
func New(conn *dbutil.Connection[*sqlc.Queries]) *Repository {
	return &Repository{conn: conn}
}

// Create{{.DomainTitle}} creates a new {{.domain}}
func (r *Repository) Create{{.DomainTitle}}(ctx context.Context, params *sqlc.Create{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error) {
	queries := r.conn.Queries()
	
	{{.domain}}, err := queries.Create{{.DomainTitle}}(ctx, params)
	if err != nil {
		return nil, dbutil.NewDatabaseError("{{.DomainTitle}}", "create", err)
	}
	
	return {{.domain}}, nil
}

// Get{{.DomainTitle}} retrieves a {{.domain}} by ID (current version)
func (r *Repository) Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*sqlc.{{.DomainTitle}}, error) {
	queries := r.conn.Queries()
	
	{{.domain}}, err := queries.Get{{.DomainTitle}}(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, dbutil.NewNotFoundError("{{.DomainTitle}}", id)
		}
		return nil, dbutil.NewDatabaseError("{{.DomainTitle}}", "get", err)
	}
	
	return {{.domain}}, nil
}

// Update{{.DomainTitle}} updates an existing {{.domain}}
func (r *Repository) Update{{.DomainTitle}}(ctx context.Context, params *sqlc.Update{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error) {
	var result *sqlc.{{.DomainTitle}}
	
	err := r.conn.WithTransaction(ctx, func(ctx context.Context, tx *sqlc.Queries) error {
		{{.domain}}, err := tx.Update{{.DomainTitle}}(ctx, params)
		if err != nil {
			if err == sql.ErrNoRows {
				return dbutil.NewNotFoundError("{{.DomainTitle}}", params.ID)
			}
			return dbutil.NewDatabaseError("{{.DomainTitle}}", "update", err)
		}
		
		result = {{.domain}}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// SoftDelete{{.DomainTitle}} soft deletes a {{.domain}}
func (r *Repository) SoftDelete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error {
	queries := r.conn.Queries()
	
	err := queries.SoftDelete{{.DomainTitle}}(ctx, id)
	if err != nil {
		return dbutil.NewDatabaseError("{{.DomainTitle}}", "delete", err)
	}
	
	return nil
}

// List{{.DomainTitle}}s retrieves a paginated list of {{.domain_plural}}
func (r *Repository) List{{.DomainTitle}}s(ctx context.Context, limit int32, afterTime *time.Time, afterID *uuid.UUID) ([]*sqlc.{{.DomainTitle}}, error) {
	queries := r.conn.Queries()
	
	// Convert to pgx types
	var timeParam pgtype.Timestamptz
	if afterTime != nil {
		timeParam = dbutil.ToPgxTimestamptz(afterTime)
	}
	
	var idParam pgtype.UUID
	if afterID != nil {
		idParam = dbutil.ToPgxUUID(*afterID)
	}
	
	items, err := queries.List{{.DomainTitle}}s(ctx, sqlc.List{{.DomainTitle}}sParams{
		Column1: timeParam,
		Column2: idParam,
		Limit:   limit,
	})
	if err != nil {
		return nil, dbutil.NewDatabaseError("{{.DomainTitle}}", "list", err)
	}
	
	return items, nil
}

// GetConnection returns the underlying database connection for testing
func (r *Repository) GetConnection() *dbutil.Connection[*sqlc.Queries] {
	return r.conn
}
```

### 3.5 Directory Structure Template

Create empty directories:
- `internal/generator/templates/internal/database/migrations/` (for user migrations)
- `internal/generator/templates/internal/repository/queries/` (for user queries)

## Dependencies
- github.com/nhalm/dbutil
- github.com/golang-migrate/migrate/v4
- github.com/google/uuid
- SQLc for code generation

## Success Criteria
- Migrations create tables with soft delete and temporal fields
- Updated_at trigger works automatically
- SQLc generates type-safe queries
- Repository uses dbutil directly (no wrapper)
- Cursor pagination queries support forward and backward
- All errors use dbutil error types