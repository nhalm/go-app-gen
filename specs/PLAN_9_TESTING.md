# Plan 9: Testing Infrastructure

## Overview
Create testing templates using standard Go testing with testify assertions, including integration tests, repository tests, and API handler tests.

## Tasks

### 9.1 Test Helpers

Location: `internal/generator/templates/internal/testutil/helpers.go.tmpl`

```go
package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nhalm/dbutil"
	"github.com/stretchr/testify/require"

	"{{.ModuleName}}/internal/repository/sqlc"
)

// TestConfig holds test configuration
type TestConfig struct {
	DatabaseURL string
	Timeout     time.Duration
}

// GetTestConfig returns test configuration from environment
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: "", // dbutil will use environment variables
		Timeout:     30 * time.Second,
	}
}

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *dbutil.Connection[*sqlc.Queries] {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use dbutil's test database functionality
	conn := dbutil.RequireTestDB(t, ctx, sqlc.New)

	return conn
}

// CleanupTestData removes all test data from tables
func CleanupTestData(t *testing.T, conn *dbutil.Connection[*sqlc.Queries]) {
	t.Helper()

	ctx := context.Background()
	
	// Clean up in reverse order of foreign key dependencies
	queries := []string{
		"TRUNCATE TABLE {{.domain_plural}} CASCADE",
		// Add other tables as needed
	}

	for _, query := range queries {
		err := conn.Exec(ctx, query)
		require.NoError(t, err, "failed to clean up test data")
	}
}

// Create{{.DomainTitle}}TestData creates test {{.domain}} data
func Create{{.DomainTitle}}TestData(t *testing.T, conn *dbutil.Connection[*sqlc.Queries], name string) *sqlc.{{.DomainTitle}} {
	t.Helper()

	ctx := context.Background()
	queries := conn.Queries()

	params := sqlc.Create{{.DomainTitle}}Params{
		Name:        name,
		Description: dbutil.ToPgxText("Test " + name),
		Column3:     dbutil.ToPgxTimestamptz(&time.Time{}), // Use current time
		Column4:     dbutil.ToPgxTimestamptz(nil),          // Use default far future
	}

	{{.domain}}, err := queries.Create{{.DomainTitle}}(ctx, params)
	require.NoError(t, err, "failed to create test {{.domain}}")

	return {{.domain}}
}

// AssertTimestampsSet verifies that created_at and updated_at are set
func AssertTimestampsSet(t *testing.T, createdAt, updatedAt time.Time) {
	t.Helper()

	now := time.Now()
	require.True(t, createdAt.After(now.Add(-1*time.Minute)), "created_at should be recent")
	require.True(t, updatedAt.After(now.Add(-1*time.Minute)), "updated_at should be recent")
	require.True(t, updatedAt.Equal(createdAt) || updatedAt.After(createdAt), "updated_at should be >= created_at")
}

// GenerateTestUUID generates a deterministic UUID for testing
func GenerateTestUUID(seed int) uuid.UUID {
	// Create a deterministic UUID based on seed
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	return uuid.NewSHA1(namespace, []byte(fmt.Sprintf("test-%d", seed)))
}
```

### 9.2 Repository Tests

Location: `internal/generator/templates/internal/repository/repository_test.go.tmpl`

```go
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"{{.ModuleName}}/internal/repository"
	"{{.ModuleName}}/internal/repository/sqlc"
	"{{.ModuleName}}/internal/testutil"
)

func TestRepository_Create{{.DomainTitle}}(t *testing.T) {
	conn := testutil.SetupTestDB(t)
	defer conn.Close()
	testutil.CleanupTestData(t, conn)

	repo := repository.New(conn)
	ctx := context.Background()

	tests := []struct {
		name    string
		params  *sqlc.Create{{.DomainTitle}}Params
		wantErr bool
	}{
		{
			name: "valid {{.domain}}",
			params: &sqlc.Create{{.DomainTitle}}Params{
				Name:        "Test {{.DomainTitle}}",
				Description: dbutil.ToPgxText("Test description"),
			},
			wantErr: false,
		},
		{
			name: "empty name",
			params: &sqlc.Create{{.DomainTitle}}Params{
				Name:        "",
				Description: dbutil.ToPgxText("Test description"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			{{.domain}}, err := repo.Create{{.DomainTitle}}(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, {{.domain}}.ID)
			assert.Equal(t, tt.params.Name, {{.domain}}.Name)
			testutil.AssertTimestampsSet(t, {{.domain}}.CreatedAt.Time, {{.domain}}.UpdatedAt.Time)
		})
	}
}

func TestRepository_Get{{.DomainTitle}}(t *testing.T) {
	conn := testutil.SetupTestDB(t)
	defer conn.Close()
	testutil.CleanupTestData(t, conn)

	repo := repository.New(conn)
	ctx := context.Background()

	// Create test data
	test{{.Domain}} := testutil.Create{{.DomainTitle}}TestData(t, conn, "Test Get")

	t.Run("existing {{.domain}}", func(t *testing.T) {
		{{.domain}}, err := repo.Get{{.DomainTitle}}(ctx, test{{.Domain}}.ID)
		
		require.NoError(t, err)
		assert.Equal(t, test{{.Domain}}.ID, {{.domain}}.ID)
		assert.Equal(t, test{{.Domain}}.Name, {{.domain}}.Name)
	})

	t.Run("non-existent {{.domain}}", func(t *testing.T) {
		{{.domain}}, err := repo.Get{{.DomainTitle}}(ctx, uuid.New())
		
		assert.Error(t, err)
		assert.Nil(t, {{.domain}})
		// Should return a specific not found error
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRepository_Update{{.DomainTitle}}(t *testing.T) {
	conn := testutil.SetupTestDB(t)
	defer conn.Close()
	testutil.CleanupTestData(t, conn)

	repo := repository.New(conn)
	ctx := context.Background()

	// Create test data
	test{{.Domain}} := testutil.Create{{.DomainTitle}}TestData(t, conn, "Test Update")
	originalUpdatedAt := test{{.Domain}}.UpdatedAt.Time

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	t.Run("update name", func(t *testing.T) {
		newName := "Updated Name"
		params := &sqlc.Update{{.DomainTitle}}Params{
			ID:      test{{.Domain}}.ID,
			Column2: dbutil.ToPgxText(&newName),
		}

		updated, err := repo.Update{{.DomainTitle}}(ctx, params)
		
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.True(t, updated.UpdatedAt.Time.After(originalUpdatedAt))
	})

	t.Run("update non-existent", func(t *testing.T) {
		params := &sqlc.Update{{.DomainTitle}}Params{
			ID:      uuid.New(),
			Column2: dbutil.ToPgxText(ptr("New Name")),
		}

		updated, err := repo.Update{{.DomainTitle}}(ctx, params)
		
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRepository_SoftDelete{{.DomainTitle}}(t *testing.T) {
	conn := testutil.SetupTestDB(t)
	defer conn.Close()
	testutil.CleanupTestData(t, conn)

	repo := repository.New(conn)
	ctx := context.Background()

	// Create test data
	test{{.Domain}} := testutil.Create{{.DomainTitle}}TestData(t, conn, "Test Delete")

	t.Run("soft delete existing", func(t *testing.T) {
		err := repo.SoftDelete{{.DomainTitle}}(ctx, test{{.Domain}}.ID)
		require.NoError(t, err)

		// Verify it's soft deleted (not returned by Get)
		{{.domain}}, err := repo.Get{{.DomainTitle}}(ctx, test{{.Domain}}.ID)
		assert.Error(t, err)
		assert.Nil(t, {{.domain}})
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := repo.SoftDelete{{.DomainTitle}}(ctx, uuid.New())
		// Should not error on deleting non-existent
		assert.NoError(t, err)
	})
}

func TestRepository_List{{.DomainTitle}}s(t *testing.T) {
	conn := testutil.SetupTestDB(t)
	defer conn.Close()
	testutil.CleanupTestData(t, conn)

	repo := repository.New(conn)
	ctx := context.Background()

	// Create test data
	for i := 0; i < 5; i++ {
		testutil.Create{{.DomainTitle}}TestData(t, conn, fmt.Sprintf("Test %d", i))
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	t.Run("list all", func(t *testing.T) {
		items, err := repo.List{{.DomainTitle}}s(ctx, 10, nil, nil)
		
		require.NoError(t, err)
		assert.Len(t, items, 5)
		
		// Verify order (newest first)
		for i := 1; i < len(items); i++ {
			assert.True(t, items[i-1].CreatedAt.Time.After(items[i].CreatedAt.Time))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		items, err := repo.List{{.DomainTitle}}s(ctx, 3, nil, nil)
		
		require.NoError(t, err)
		assert.Len(t, items, 3)
	})

	t.Run("list with cursor", func(t *testing.T) {
		// Get first page
		firstPage, err := repo.List{{.DomainTitle}}s(ctx, 2, nil, nil)
		require.NoError(t, err)
		require.Len(t, firstPage, 2)

		// Get second page using cursor
		lastItem := firstPage[len(firstPage)-1]
		secondPage, err := repo.List{{.DomainTitle}}s(ctx, 2, &lastItem.CreatedAt.Time, &lastItem.ID)
		
		require.NoError(t, err)
		assert.Len(t, secondPage, 2)
		
		// Ensure no overlap
		for _, item := range secondPage {
			assert.NotEqual(t, firstPage[0].ID, item.ID)
			assert.NotEqual(t, firstPage[1].ID, item.ID)
		}
	})
}

// Helper function
func ptr[T any](v T) *T {
	return &v
}
```

### 9.3 Service Tests

Location: `internal/generator/templates/internal/service/service_test.go.tmpl`

```go
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"{{.ModuleName}}/internal/repository/sqlc"
	"{{.ModuleName}}/internal/service"
)

// MockRepository is a mock implementation of RepositoryInterface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create{{.DomainTitle}}(ctx context.Context, params *sqlc.Create{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlc.{{.DomainTitle}}), args.Error(1)
}

func (m *MockRepository) Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*sqlc.{{.DomainTitle}}, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlc.{{.DomainTitle}}), args.Error(1)
}

func (m *MockRepository) Update{{.DomainTitle}}(ctx context.Context, params *sqlc.Update{{.DomainTitle}}Params) (*sqlc.{{.DomainTitle}}, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlc.{{.DomainTitle}}), args.Error(1)
}

func (m *MockRepository) SoftDelete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List{{.DomainTitle}}s(ctx context.Context, limit int32, afterTime *time.Time, afterID *uuid.UUID) ([]*sqlc.{{.DomainTitle}}, error) {
	args := m.Called(ctx, limit, afterTime, afterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sqlc.{{.DomainTitle}}), args.Error(1)
}

func TestService_Create{{.DomainTitle}}(t *testing.T) {
	ctx := context.Background()

	t.Run("valid creation", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.New(mockRepo)

		expected := &sqlc.{{.DomainTitle}}{
			ID:        uuid.New(),
			Name:      "Test {{.DomainTitle}}",
			CreatedAt: dbutil.ToPgxTimestamptz(&time.Now()),
			UpdatedAt: dbutil.ToPgxTimestamptz(&time.Now()),
		}

		mockRepo.On("Create{{.DomainTitle}}", ctx, mock.MatchedBy(func(params *sqlc.Create{{.DomainTitle}}Params) bool {
			return params.Name == "Test {{.DomainTitle}}"
		})).Return(expected, nil)

		req := &service.Create{{.DomainTitle}}Request{
			Name: "Test {{.DomainTitle}}",
		}

		result, err := svc.Create{{.DomainTitle}}(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty name", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.New(mockRepo)

		req := &service.Create{{.DomainTitle}}Request{
			Name: "",
		}

		result, err := svc.Create{{.DomainTitle}}(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "name is required")
		mockRepo.AssertNotCalled(t, "Create{{.DomainTitle}}")
	})
}

func TestService_List{{.DomainTitle}}s(t *testing.T) {
	ctx := context.Background()

	t.Run("pagination", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := service.New(mockRepo)

		// Create test data
		items := make([]*sqlc.{{.DomainTitle}}, 3)
		for i := 0; i < 3; i++ {
			items[i] = &sqlc.{{.DomainTitle}}{
				ID:        uuid.New(),
				Name:      fmt.Sprintf("Test %d", i),
				CreatedAt: dbutil.ToPgxTimestamptz(&time.Now()),
			}
		}

		// Mock returns 3 items (limit + 1)
		mockRepo.On("List{{.DomainTitle}}s", ctx, int32(3), (*time.Time)(nil), (*uuid.UUID)(nil)).
			Return(items, nil)

		params := &service.ListParams{
			Limit: 2,
		}

		result, err := svc.List{{.DomainTitle}}s(ctx, params)

		require.NoError(t, err)
		assert.Len(t, result.Items, 2) // Should trim to requested limit
		assert.True(t, result.HasNext)  // Has more items
		assert.NotEmpty(t, result.StartCursor)
		assert.NotEmpty(t, result.EndCursor)
		mockRepo.AssertExpectations(t)
	})
}
```

### 9.4 API Handler Tests

Location: `internal/generator/templates/internal/api/handler_test.go.tmpl`

```go
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"{{.ModuleName}}/internal/api"
	"{{.ModuleName}}/internal/service"
)

// MockService is a mock implementation of ServiceInterface
type MockService struct {
	mock.Mock
}

func (m *MockService) Create{{.DomainTitle}}(ctx context.Context, req *service.Create{{.DomainTitle}}Request) (*service.{{.DomainTitle}}, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.{{.DomainTitle}}), args.Error(1)
}

func (m *MockService) Get{{.DomainTitle}}(ctx context.Context, id uuid.UUID) (*service.{{.DomainTitle}}, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.{{.DomainTitle}}), args.Error(1)
}

func (m *MockService) Update{{.DomainTitle}}(ctx context.Context, id uuid.UUID, req *service.Update{{.DomainTitle}}Request) (*service.{{.DomainTitle}}, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.{{.DomainTitle}}), args.Error(1)
}

func (m *MockService) Delete{{.DomainTitle}}(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) List{{.DomainTitle}}s(ctx context.Context, params *service.ListParams) (*service.{{.DomainTitle}}List, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.{{.DomainTitle}}List), args.Error(1)
}

func TestHandler_Create{{.DomainTitle}}(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		mockService := new(MockService)
		handler := api.NewHandler(mockService)

		expected := &service.{{.DomainTitle}}{
			ID:   uuid.New(),
			Name: "Test {{.DomainTitle}}",
		}

		mockService.On("Create{{.DomainTitle}}", mock.Anything, mock.MatchedBy(func(req *service.Create{{.DomainTitle}}Request) bool {
			return req.Name == "Test {{.DomainTitle}}"
		})).Return(expected, nil)

		body := api.{{.DomainTitle}}CreateRequest{
			Name: "Test {{.DomainTitle}}",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/{{.domain_plural}}", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Create{{.DomainTitle}}(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response api.Response
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "{{.domain}}", response.Type)
		assert.NotNil(t, response.Data)
		assert.Empty(t, response.Errors)

		mockService.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := new(MockService)
		handler := api.NewHandler(mockService)

		req := httptest.NewRequest("POST", "/api/v1/{{.domain_plural}}", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Create{{.DomainTitle}}(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response api.Response
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "error", response.Type)
		assert.NotEmpty(t, response.Errors)

		mockService.AssertNotCalled(t, "Create{{.DomainTitle}}")
	})
}

func TestHandler_Get{{.DomainTitle}}(t *testing.T) {
	t.Run("existing {{.domain}}", func(t *testing.T) {
		mockService := new(MockService)
		handler := api.NewHandler(mockService)

		id := uuid.New()
		expected := &service.{{.DomainTitle}}{
			ID:   id,
			Name: "Test {{.DomainTitle}}",
		}

		mockService.On("Get{{.DomainTitle}}", mock.Anything, id).Return(expected, nil)

		req := httptest.NewRequest("GET", "/api/v1/{{.domain_plural}}/"+id.String(), nil)
		w := httptest.NewRecorder()

		// Setup chi context with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.Get{{.DomainTitle}}(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.Response
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "{{.domain}}", response.Type)
		assert.NotNil(t, response.Data)

		mockService.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockService := new(MockService)
		handler := api.NewHandler(mockService)

		id := uuid.New()
		mockService.On("Get{{.DomainTitle}}", mock.Anything, id).Return(nil, service.ErrNotFound)

		req := httptest.NewRequest("GET", "/api/v1/{{.domain_plural}}/"+id.String(), nil)
		w := httptest.NewRecorder()

		// Setup chi context with URL param
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.Get{{.DomainTitle}}(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		mockService.AssertExpectations(t)
	})
}

func TestHandler_List{{.DomainTitle}}s(t *testing.T) {
	t.Run("list with pagination", func(t *testing.T) {
		mockService := new(MockService)
		handler := api.NewHandler(mockService)

		items := []*service.{{.DomainTitle}}{
			{ID: uuid.New(), Name: "Test 1"},
			{ID: uuid.New(), Name: "Test 2"},
		}

		expected := &service.{{.DomainTitle}}List{
			Items:       items,
			HasNext:     true,
			StartCursor: "cursor1",
			EndCursor:   "cursor2",
		}

		mockService.On("List{{.DomainTitle}}s", mock.Anything, mock.MatchedBy(func(params *service.ListParams) bool {
			return params.Limit == 10
		})).Return(expected, nil)

		req := httptest.NewRequest("GET", "/api/v1/{{.domain_plural}}?limit=10", nil)
		w := httptest.NewRecorder()

		handler.List{{.DomainTitle}}s(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.ListResponse
		err := json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "list", response.Type)
		assert.True(t, response.PageInfo.HasNext)
		assert.NotEmpty(t, response.PageInfo.StartCursor)
		assert.NotEmpty(t, response.PageInfo.EndCursor)

		mockService.AssertExpectations(t)
	})
}
```

### 9.5 Integration Test Setup

Location: `internal/generator/templates/internal/integration/setup_test.go.tmpl`

```go
package integration_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nhalm/dbutil"

	"{{.ModuleName}}/internal/api"
	"{{.ModuleName}}/internal/repository"
	"{{.ModuleName}}/internal/repository/sqlc"
	"{{.ModuleName}}/internal/service"
	"{{.ModuleName}}/internal/testutil"
)

// TestServer provides a test HTTP server with all dependencies
type TestServer struct {
	*httptest.Server
	DB      *dbutil.Connection[*sqlc.Queries]
	Handler *api.Handler
}

// SetupTestServer creates a fully configured test server
func SetupTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Setup database
	conn := testutil.SetupTestDB(t)

	// Create layers
	repo := repository.New(conn)
	svc := service.New(repo)
	handler := api.NewHandler(svc)

	// Setup router
	r := chi.NewRouter()
	api.RegisterRoutes(r, handler)

	// Create test server
	server := httptest.NewServer(r)

	return &TestServer{
		Server:  server,
		DB:      conn,
		Handler: handler,
	}
}

// Cleanup cleans up test resources
func (ts *TestServer) Cleanup(t *testing.T) {
	t.Helper()
	ts.Server.Close()
	ts.DB.Close()
}
```

## Dependencies
- github.com/stretchr/testify
- Standard testing package
- github.com/nhalm/dbutil (for test database)

## Success Criteria
- Tests use standard Go testing (no testify/suite)
- Repository tests verify database operations
- Service tests use mocks for isolation
- API handler tests verify HTTP behavior
- Integration tests available for end-to-end testing
- Test helpers reduce boilerplate
- dbutil.RequireTestDB used for database tests