# Plan 6: Utilities

## Overview
Create utility templates for request logging with slog, cursor pagination helpers, and common utilities.

## Tasks

### 6.1 Request Logger Middleware

Location: `internal/generator/templates/internal/utils/logger.go.tmpl`

```go
package utils

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// contextKey is a type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
)

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	// Fall back to chi's request ID
	return middleware.GetReqID(ctx)
}

// RequestLoggerMiddleware creates a middleware for single-line request logging
func RequestLoggerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			
			// Add request ID to context
			ctx := r.Context()
			requestID := middleware.GetReqID(ctx)
			ctx = context.WithValue(ctx, RequestIDKey, requestID)
			r = r.WithContext(ctx)
			
			// Process request
			next.ServeHTTP(ww, r)
			
			// Log request (single line)
			duration := time.Since(start)
			slog.InfoContext(ctx, "request",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", duration),
				slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
			)
		})
	}
}

// ErrorLoggerMiddleware logs errors with context
func ErrorLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				ctx := r.Context()
				requestID := GetRequestID(ctx)
				
				slog.ErrorContext(ctx, "panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
				
				// Re-panic to let the Recoverer middleware handle it
				panic(err)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}
```

### 6.2 Cursor Pagination Utilities

Location: `internal/generator/templates/internal/utils/cursor.go.tmpl`

```go
package utils

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EncodeCursor creates a cursor from timestamp and ID
func EncodeCursor(timestamp time.Time, id uuid.UUID) string {
	// Format: timestamp_id
	cursor := fmt.Sprintf("%d_%s", timestamp.UnixNano(), id.String())
	return base64.URLEncoding.EncodeToString([]byte(cursor))
}

// DecodeCursor extracts timestamp and ID from cursor
func DecodeCursor(cursor string) (time.Time, uuid.UUID, error) {
	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	parts := strings.Split(string(decoded), "_")
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor structure")
	}

	// Parse timestamp
	var nanos int64
	if _, err := fmt.Sscanf(parts[0], "%d", &nanos); err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid timestamp in cursor: %w", err)
	}
	timestamp := time.Unix(0, nanos)

	// Parse UUID
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid ID in cursor: %w", err)
	}

	return timestamp, id, nil
}

// PaginationParams holds common pagination parameters
type PaginationParams struct {
	Limit       int
	Cursor      string
	Direction   string // "forward" or "backward"
}

// ParsePaginationParams extracts pagination parameters from query string
func ParsePaginationParams(r *http.Request) (*PaginationParams, error) {
	params := &PaginationParams{
		Limit:     20, // default
		Direction: "forward",
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
			if limit > 0 && limit <= 100 {
				params.Limit = limit
			}
		}
	}

	// Parse cursor
	params.Cursor = r.URL.Query().Get("cursor")

	// Parse direction
	if dir := r.URL.Query().Get("direction"); dir == "backward" {
		params.Direction = "backward"
	}

	return params, nil
}
```

### 6.3 Common Utilities

Location: `internal/generator/templates/internal/utils/helpers.go.tmpl`

```go
package utils

import (
	"context"
	"strings"
	"unicode"
)

// StringPtr returns a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to a bool
func BoolPtr(b bool) *bool {
	return &b
}

// NormalizeString normalizes a string for consistent storage
func NormalizeString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)
	
	// Normalize internal whitespace
	var result strings.Builder
	lastWasSpace := false
	
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			result.WriteRune(r)
			lastWasSpace = false
		}
	}
	
	return result.String()
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	// Find last space before maxLen
	truncated := s[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	
	if lastSpace > 0 && lastSpace > maxLen-20 {
		return s[:lastSpace] + "..."
	}
	
	return truncated + "..."
}

// ContextWithRequestID adds a request ID to the context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// SafeDeref safely dereferences a pointer
func SafeDeref[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// Filter filters a slice based on a predicate
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms a slice using a mapping function
func Map[T, U any](slice []T, mapper func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = mapper(item)
	}
	return result
}
```

### 6.4 Validation Utilities

Location: `internal/generator/templates/internal/utils/validation.go.tmpl`

```go
package utils

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

var (
	// Common regex patterns
	alphanumericRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	slugRegex         = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// ValidateEmail validates an email address
func ValidateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateSlug validates a URL-friendly slug
func ValidateSlug(slug string) error {
	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}
	return nil
}

// ValidateAlphanumeric validates that a string contains only letters and numbers
func ValidateAlphanumeric(s string) error {
	if !alphanumericRegex.MatchString(s) {
		return fmt.Errorf("must contain only letters and numbers")
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(s string, min, max int) error {
	length := len(s)
	if length < min {
		return fmt.Errorf("must be at least %d characters", min)
	}
	if max > 0 && length > max {
		return fmt.Errorf("must be at most %d characters", max)
	}
	return nil
}

// SanitizeInput removes potentially dangerous characters
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Normalize whitespace
	input = NormalizeString(input)
	
	return input
}
```

## Dependencies
- Standard library (log/slog, encoding/base64, etc.)
- github.com/go-chi/chi/v5
- github.com/google/uuid

## Success Criteria
- Single-line request logging with all relevant fields
- Cursor encoding/decoding works correctly
- Common utility functions are generic and reusable
- Request ID propagation through context
- Proper error recovery logging