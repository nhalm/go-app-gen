# Plan 5: Service Layer

## Overview
Create service layer templates that implement business logic, define repository interfaces, and handle model conversions between layers.

## Tasks

### 5.1 Service Models

Location: `internal/generator/templates/internal/service/models.go.tmpl`

```go
package service

import (
	"time"
	"github.com/google/uuid"
)

// {{.DomainTitle}} represents a {{.domain}} in the service layer
type {{.DomainTitle}} struct {
	ID             uuid.UUID
	Name           string
	Description    *string
	EffectiveStart time.Time
	EffectiveEnd   time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Create{{.DomainTitle}}Request contains data for creating a {{.domain}}
type Create{{.DomainTitle}}Request struct {
	Name           string
	Description    *string
	EffectiveStart *time.Time
	EffectiveEnd   *time.Time
}

// Update{{.DomainTitle}}Request contains data for updating a {{.domain}}
type Update{{.DomainTitle}}Request struct {
	Name        *string
	Description *string
}

// ListParams contains parameters for listing {{.domain_plural}}
type ListParams struct {
	Limit     int
	AfterTime *time.Time
	AfterID   *uuid.UUID
}

// {{.DomainTitle}}List contains a list of {{.domain_plural}} with pagination info
type {{.DomainTitle}}List struct {
	Items       []*{{.DomainTitle}}
	HasNext     bool
	StartCursor string
	EndCursor   string
}
```

### 5.2 Service Implementation

Location: `internal/generator/templates/internal/service/service.go.tmpl`

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"{{.ModuleName}}/internal/repository/sqlc"
	"{{.ModuleName}}/internal/utils"
)

var (
	// ErrNotFound is returned when a {{.domain}} is not found
	ErrNotFound = errors.New("{{.domain}} not found")
	
	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)

// RepositoryInterface defines what the service needs from the repository
type RepositoryInterface interface {
	Create{{.DomainTitle}}(ctx context.Context, params *sqlc.Create{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error)
	Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*sqlc.{{.DomainTitle}}, error)
	Update{{.DomainTitle}}(ctx context.Context, params *sqlc.Update{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error)
	SoftDelete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error
	List{{.DomainTitle}}s(ctx context.Context, limit int32, afterTime *time.Time, afterID *uuid.UUID) ([]*sqlc.{{.DomainTitle}}, error)
}

// Service implements business logic for {{.domain_plural}}
type Service struct {
	repo RepositoryInterface
}

// New creates a new service instance
func New(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// Create{{.DomainTitle}} creates a new {{.domain}}
func (s *Service) Create{{.DomainTitle}}(ctx context.Context, req *Create{{.DomainTitle}}Request) (*{{.DomainTitle}}, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	// Set defaults for temporal fields
	effectiveStart := time.Now()
	if req.EffectiveStart != nil {
		effectiveStart = *req.EffectiveStart
	}

	effectiveEnd := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if req.EffectiveEnd != nil {
		effectiveEnd = *req.EffectiveEnd
	}

	params := &sqlc.Create{{.DomainTitle}}Params{
		Name:        req.Name,
		Description: convertStringPtr(req.Description),
		Column3:     pgtype.Timestamptz{Time: effectiveStart, Valid: true},
		Column4:     pgtype.Timestamptz{Time: effectiveEnd, Valid: true},
	}

	dbModel, err := s.repo.Create{{.DomainTitle}}(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create {{.domain}}: %w", err)
	}

	return s.toServiceModel(dbModel), nil
}

// Get{{.DomainTitle}} retrieves a {{.domain}} by ID
func (s *Service) Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*{{.DomainTitle}}, error) {
	dbModel, err := s.repo.Get{{.DomainTitle}}(ctx, id)
	if err != nil {
		// Repository should return a specific error we can check
		if errors.Is(err, sqlc.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get {{.domain}}: %w", err)
	}

	return s.toServiceModel(dbModel), nil
}

// Update{{.DomainTitle}} updates an existing {{.domain}}
func (s *Service) Update{{.DomainTitle}}(ctx context.Context, id uuid.UUID, req *Update{{.DomainTitle}}Request) (*{{.DomainTitle}}, error) {
	// Check if at least one field is being updated
	if req.Name == nil && req.Description == nil {
		return nil, fmt.Errorf("%w: no fields to update", ErrInvalidInput)
	}

	params := &sqlc.Update{{.DomainTitle}}Params{
		ID:          id,
		Column2:     convertStringPtrToPgText(req.Name),
		Column3:     convertStringPtrToPgText(req.Description),
	}

	dbModel, err := s.repo.Update{{.DomainTitle}}(ctx, params)
	if err != nil {
		if errors.Is(err, sqlc.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to update {{.domain}}: %w", err)
	}

	return s.toServiceModel(dbModel), nil
}

// Delete{{.DomainTitle}} soft deletes a {{.domain}}
func (s *Service) Delete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error {
	err := s.repo.SoftDelete{{.DomainTitle}}(ctx, id)
	if err != nil {
		if errors.Is(err, sqlc.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete {{.domain}}: %w", err)
	}

	return nil
}

// List{{.DomainTitle}}s retrieves a paginated list of {{.domain_plural}}
func (s *Service) List{{.DomainTitle}}s(ctx context.Context, params *ListParams) (*{{.DomainTitle}}List, error) {
	// Fetch one extra item to determine if there are more results
	limit := int32(params.Limit + 1)

	items, err := s.repo.List{{.DomainTitle}}s(ctx, limit, params.AfterTime, params.AfterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list {{.domain_plural}}: %w", err)
	}

	// Check if we have more results
	hasNext := len(items) > params.Limit
	if hasNext {
		// Remove the extra item
		items = items[:params.Limit]
	}

	// Convert to service models
	serviceItems := make([]*{{.DomainTitle}}, len(items))
	for i, item := range items {
		serviceItems[i] = s.toServiceModel(item)
	}

	// Generate cursors
	var startCursor, endCursor string
	if len(serviceItems) > 0 {
		startCursor = utils.EncodeCursor(serviceItems[0].CreatedAt, serviceItems[0].ID)
		endCursor = utils.EncodeCursor(serviceItems[len(serviceItems)-1].CreatedAt, serviceItems[len(serviceItems)-1].ID)
	}

	return &{{.DomainTitle}}List{
		Items:       serviceItems,
		HasNext:     hasNext,
		StartCursor: startCursor,
		EndCursor:   endCursor,
	}, nil
}

// Model conversion helpers

func (s *Service) toServiceModel(db *sqlc.{{.DomainTitle}}) *{{.DomainTitle}} {
	return &{{.DomainTitle}}{
		ID:             db.ID,
		Name:           db.Name,
		Description:    convertPgTextToStringPtr(db.Description),
		EffectiveStart: db.EffectiveStart.Time,
		EffectiveEnd:   db.EffectiveEnd.Time,
		CreatedAt:      db.CreatedAt.Time,
		UpdatedAt:      db.UpdatedAt.Time,
	}
}

func convertStringPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func convertStringPtrToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func convertPgTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
```

### 5.3 Service Error Handling

Location: `internal/generator/templates/internal/service/errors.go.tmpl`

```go
package service

import (
	"errors"
	"fmt"
)

// BusinessError represents a business logic error
type BusinessError struct {
	Code    string
	Message string
	Err     error
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

// NewBusinessError creates a new business error
func NewBusinessError(code, message string, err error) error {
	return &BusinessError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common business errors
var (
	ErrDuplicateName = NewBusinessError("duplicate_name", "A {{.domain}} with this name already exists", nil)
	ErrInvalidDateRange = NewBusinessError("invalid_date_range", "Effective start date must be before end date", nil)
	ErrExpired = NewBusinessError("expired", "Cannot modify an expired {{.domain}}", nil)
)
```

## Dependencies
- github.com/google/uuid
- github.com/jackc/pgx/v5

## Success Criteria
- Service layer contains all business logic
- Repository interface defined by service (consumer)
- Clean model conversions between layers
- Proper error handling and wrapping
- Pagination logic implemented correctly
- Temporal field validation