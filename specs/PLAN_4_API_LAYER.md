# Plan 4: API Layer

## Overview
Create API handler templates that implement JustiFi response format with cursor-based pagination and proper error handling.

## Tasks

### 4.1 API Response Types

Location: `internal/generator/templates/internal/api/types.go.tmpl`

```go
package api

import (
	"time"
	"github.com/google/uuid"
)

// Standard JustiFi envelope response format
type Response struct {
	ID     string      `json:"id"`
	Type   string      `json:"type"`
	Data   interface{} `json:"data,omitempty"`
	Errors []Error     `json:"errors,omitempty"`
}

// Error represents a JustiFi API error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

// PageInfo contains cursor pagination metadata
type PageInfo struct {
	HasNext     bool   `json:"has_next"`
	HasPrevious bool   `json:"has_previous"`
	StartCursor string `json:"start_cursor,omitempty"`
	EndCursor   string `json:"end_cursor,omitempty"`
}

// ListResponse is the standard list response format
type ListResponse struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	PageInfo PageInfo    `json:"page_info"`
}

// {{.DomainTitle}}Response is the API representation of a {{.domain}}
type {{.DomainTitle}}Response struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	EffectiveStart  time.Time  `json:"effective_start"`
	EffectiveEnd    time.Time  `json:"effective_end"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// {{.DomainTitle}}CreateRequest represents a create request
type {{.DomainTitle}}CreateRequest struct {
	Name           string     `json:"name" validate:"required,min=1,max=255"`
	Description    *string    `json:"description,omitempty"`
	EffectiveStart *time.Time `json:"effective_start,omitempty"`
	EffectiveEnd   *time.Time `json:"effective_end,omitempty"`
}

// {{.DomainTitle}}UpdateRequest represents an update request
type {{.DomainTitle}}UpdateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string `json:"description,omitempty"`
}
```

### 4.2 Handler Implementation

Location: `internal/generator/templates/internal/api/handler.go.tmpl`

```go
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"{{.ModuleName}}/internal/service"
	"{{.ModuleName}}/internal/utils"
)

// ServiceInterface defines what the handler needs from the service layer
type ServiceInterface interface {
	Create{{.DomainTitle}}(ctx context.Context, req *service.Create{{.DomainTitle}}Request) (*service.{{.DomainTitle}}, error)
	Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*service.{{.DomainTitle}}, error)
	Update{{.DomainTitle}}(ctx context.Context, id uuid.UUID, req *service.Update{{.DomainTitle}}Request) (*service.{{.DomainTitle}}, error)
	Delete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error
	List{{.DomainTitle}}s(ctx context.Context, params *service.ListParams) (*service.{{.DomainTitle}}List, error)
}

// Handler handles HTTP requests for {{.domain_plural}}
type Handler struct {
	service   ServiceInterface
	validator *validator.Validate
}

// NewHandler creates a new API handler
func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

// Create{{.DomainTitle}} handles POST /{{.domain_plural}}
func (h *Handler) Create{{.DomainTitle}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := utils.GetRequestID(ctx)

	var req {{.DomainTitle}}CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.sendValidationError(w, err)
		return
	}

	serviceReq := &service.Create{{.DomainTitle}}Request{
		Name:           req.Name,
		Description:    req.Description,
		EffectiveStart: req.EffectiveStart,
		EffectiveEnd:   req.EffectiveEnd,
	}

	{{.domain}}, err := h.service.Create{{.DomainTitle}}(ctx, serviceReq)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create {{.domain}}",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to create {{.domain}}")
		return
	}

	response := Response{
		ID:   requestID,
		Type: "{{.domain}}",
		Data: h.toResponse({{.domain}}),
	}

	h.sendJSON(w, http.StatusCreated, response)
}

// Get{{.DomainTitle}} handles GET /{{.domain_plural}}/:id
func (h *Handler) Get{{.DomainTitle}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := utils.GetRequestID(ctx)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_id", "Invalid {{.domain}} ID")
		return
	}

	{{.domain}}, err := h.service.Get{{.DomainTitle}}(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "not_found", "{{.DomainTitle}} not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to get {{.domain}}",
			slog.String("request_id", requestID),
			slog.String("id", id.String()),
			slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to get {{.domain}}")
		return
	}

	response := Response{
		ID:   requestID,
		Type: "{{.domain}}",
		Data: h.toResponse({{.domain}}),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Update{{.DomainTitle}} handles PATCH /{{.domain_plural}}/:id
func (h *Handler) Update{{.DomainTitle}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := utils.GetRequestID(ctx)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_id", "Invalid {{.domain}} ID")
		return
	}

	var req {{.DomainTitle}}UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.sendValidationError(w, err)
		return
	}

	serviceReq := &service.Update{{.DomainTitle}}Request{
		Name:        req.Name,
		Description: req.Description,
	}

	{{.domain}}, err := h.service.Update{{.DomainTitle}}(ctx, id, serviceReq)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "not_found", "{{.DomainTitle}} not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to update {{.domain}}",
			slog.String("request_id", requestID),
			slog.String("id", id.String()),
			slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to update {{.domain}}")
		return
	}

	response := Response{
		ID:   requestID,
		Type: "{{.domain}}",
		Data: h.toResponse({{.domain}}),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Delete{{.DomainTitle}} handles DELETE /{{.domain_plural}}/:id
func (h *Handler) Delete{{.DomainTitle}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := utils.GetRequestID(ctx)

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid_id", "Invalid {{.domain}} ID")
		return
	}

	err = h.service.Delete{{.DomainTitle}}(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.sendError(w, http.StatusNotFound, "not_found", "{{.DomainTitle}} not found")
			return
		}
		slog.ErrorContext(ctx, "Failed to delete {{.domain}}",
			slog.String("request_id", requestID),
			slog.String("id", id.String()),
			slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to delete {{.domain}}")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List{{.DomainTitle}}s handles GET /{{.domain_plural}}
func (h *Handler) List{{.DomainTitle}}s(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := utils.GetRequestID(ctx)

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse cursor
	cursor := r.URL.Query().Get("cursor")
	var afterTime *time.Time
	var afterID *uuid.UUID
	if cursor != "" {
		t, id, err := utils.DecodeCursor(cursor)
		if err != nil {
			h.sendError(w, http.StatusBadRequest, "invalid_cursor", "Invalid cursor")
			return
		}
		afterTime = &t
		afterID = &id
	}

	params := &service.ListParams{
		Limit:     limit,
		AfterTime: afterTime,
		AfterID:   afterID,
	}

	result, err := h.service.List{{.DomainTitle}}s(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list {{.domain_plural}}",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, "internal_error", "Failed to list {{.domain_plural}}")
		return
	}

	// Convert to API responses
	items := make([]{{.DomainTitle}}Response, len(result.Items))
	for i, item := range result.Items {
		items[i] = *h.toResponse(item)
	}

	response := ListResponse{
		ID:   requestID,
		Type: "list",
		Data: items,
		PageInfo: PageInfo{
			HasNext:     result.HasNext,
			HasPrevious: cursor != "",
			StartCursor: result.StartCursor,
			EndCursor:   result.EndCursor,
		},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *Handler) toResponse({{.domain}} *service.{{.DomainTitle}}) *{{.DomainTitle}}Response {
	return &{{.DomainTitle}}Response{
		ID:             {{.domain}}.ID.String(),
		Name:           {{.domain}}.Name,
		Description:    {{.domain}}.Description,
		EffectiveStart: {{.domain}}.EffectiveStart,
		EffectiveEnd:   {{.domain}}.EffectiveEnd,
		CreatedAt:      {{.domain}}.CreatedAt,
		UpdatedAt:      {{.domain}}.UpdatedAt,
	}
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, code, message string) {
	response := Response{
		ID:   utils.GetRequestID(r.Context()),
		Type: "error",
		Errors: []Error{
			{Code: code, Message: message},
		},
	}
	h.sendJSON(w, status, response)
}

func (h *Handler) sendValidationError(w http.ResponseWriter, err error) {
	var errors []Error
	for _, e := range err.(validator.ValidationErrors) {
		errors = append(errors, Error{
			Code:    "validation_error",
			Message: "Validation failed",
			Field:   e.Field(),
		})
	}
	
	response := Response{
		ID:     utils.GetRequestID(r.Context()),
		Type:   "error",
		Errors: errors,
	}
	h.sendJSON(w, http.StatusBadRequest, response)
}
```

### 4.3 Routes Registration

Location: `internal/generator/templates/internal/api/routes.go.tmpl`

```go
package api

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(r chi.Router, handler *Handler) {
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", HealthCheck)
		
		// {{.DomainTitle}} routes
		r.Route("/{{.domain_plural}}", func(r chi.Router) {
			r.Get("/", handler.List{{.DomainTitle}}s)
			r.Post("/", handler.Create{{.DomainTitle}})
			
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", handler.Get{{.DomainTitle}})
				r.Patch("/", handler.Update{{.DomainTitle}})
				r.Delete("/", handler.Delete{{.DomainTitle}})
			})
		})
	})
}

// HealthCheck returns service health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

## Dependencies
- github.com/go-chi/chi/v5
- github.com/go-playground/validator/v10
- github.com/google/uuid

## Success Criteria
- JustiFi-compliant envelope format for all responses
- Proper error handling with structured errors
- Cursor-based pagination working correctly
- Request validation using validator
- Service interface defined by consumer (API layer)