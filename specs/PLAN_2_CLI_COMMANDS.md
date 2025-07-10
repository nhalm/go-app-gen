# Plan 2: CLI Commands Templates

## Overview
Create/update Cobra CLI command templates with environment-only configuration, slog logging, and golang-migrate support.

## Tasks

### 2.1 Update root.go.tmpl

Location: `internal/generator/templates/cmd/{{.AppName}}/cmd/root.go.tmpl`

Key changes:
- Remove all viper config file support
- Use only `os.Getenv()` for configuration
- Remove initConfig complexity

```go
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
	goVersion = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "{{.AppName}}",
	Short: "{{.Description}}",
	Long: `{{.Description}}
		
A sophisticated API for {{.Domain}} management with comprehensive features
including database integration, migration support, and modern tooling.`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func SetVersionInfo(ver, gitCommit, buildTime, goVer string) {
	version = ver
	commit = gitCommit
	buildDate = buildTime
	goVersion = goVer
	rootCmd.Version = version
}

func init() {
	// Version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("{{.AppName}} version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built on: %s\n", buildDate)
			fmt.Printf("built with: %s\n", goVersion)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Register subcommands
	RegisterServeCommand(rootCmd)
	RegisterMigrateCommand(rootCmd)
}

// Helper function for environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

### 2.2 Create/Update serve.go.tmpl

Location: `internal/generator/templates/cmd/{{.AppName}}/cmd/serve.go.tmpl`

Key features:
- slog for structured logging
- Request logging middleware integration
- dbutil for database connections
- Graceful shutdown

```go
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
	"github.com/nhalm/dbutil"

	"{{.ModuleName}}/internal/api"
	"{{.ModuleName}}/internal/service"
	"{{.ModuleName}}/internal/repository"
	"{{.ModuleName}}/internal/repository/sqlc"
	"{{.ModuleName}}/internal/utils"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the {{.AppName}} API server",
	Long:  `Start the {{.AppName}} API server with HTTP endpoints for {{.Domain}} management.`,
	RunE:  runServe,
}

func RegisterServeCommand(rootCmd *cobra.Command) {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	// Setup logging
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "text")
	setupLogger(logLevel, logFormat)
	
	// Get configuration from environment
	port := getEnv("PORT", "8080")
	host := getEnv("HOST", "0.0.0.0")
	
	slog.Info("Starting {{.AppName}} server", 
		slog.String("host", host),
		slog.String("port", port),
		slog.String("version", version))
	
	// Initialize database connection
	conn, err := dbutil.NewConnection(ctx, "", sqlc.New)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()
	
	// Check database connectivity
	if !conn.IsReady(ctx) {
		return fmt.Errorf("database is not ready")
	}
	
	// Initialize layers
	repo := repository.New(conn)
	svc := service.New(repo)
	handler := api.NewHandler(svc)
	
	// Setup router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(utils.RequestLoggerMiddleware())
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	
	// Register routes
	api.RegisterRoutes(r, handler)
	
	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: r,
	}
	
	// Start server in goroutine
	go func() {
		slog.Info("Server listening", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", slog.String("error", err.Error()))
		}
	}()
	
	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	// Graceful shutdown
	slog.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	
	slog.Info("Server stopped")
	return nil
}

func setupLogger(level, format string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	
	opts := &slog.HandlerOptions{Level: logLevel}
	
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
```

### 2.3 Create/Update migrate.go.tmpl

Location: `internal/generator/templates/cmd/{{.AppName}}/cmd/migrate.go.tmpl`

Key features:
- golang-migrate integration
- Uses dbutil.GetDSN()
- Subcommands: up, down, version, status, create

```go
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/nhalm/dbutil"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  `Run database migrations for the {{.AppName}} application.`,
}

func RegisterMigrateCommand(rootCmd *cobra.Command) {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateVersionCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateCreateCmd)
	rootCmd.AddCommand(migrateCmd)
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	RunE:  runMigrateUp,
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [n]",
	Short: "Rollback migrations",
	Long:  "Rollback n migrations (default 1)",
	RunE:  runMigrateDown,
}

var migrateVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current migration version",
	RunE:  runMigrateVersion,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE:  runMigrateStatus,
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration file",
	Args:  cobra.ExactArgs(1),
	RunE:  runMigrateCreate,
}

func createMigrator() (*migrate.Migrate, error) {
	dsn := dbutil.GetDSN()
	m, err := migrate.New(
		"file://internal/database/migrations",
		dsn,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}
	return m, nil
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	slog.Info("Running database migrations...")
	
	m, err := createMigrator()
	if err != nil {
		return err
	}
	defer m.Close()
	
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No migrations to run")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}
	
	slog.Info("Migrations completed successfully")
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	steps := 1
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &steps)
	}
	
	slog.Info("Rolling back migrations", slog.Int("steps", steps))
	
	m, err := createMigrator()
	if err != nil {
		return err
	}
	defer m.Close()
	
	if err := m.Steps(-steps); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No migrations to rollback")
			return nil
		}
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	slog.Info("Rollback completed successfully")
	return nil
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	m, err := createMigrator()
	if err != nil {
		return err
	}
	defer m.Close()
	
	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("No migrations have been applied")
			return nil
		}
		return err
	}
	
	if dirty {
		fmt.Printf("Version: %d (dirty)\n", version)
	} else {
		fmt.Printf("Version: %d\n", version)
	}
	
	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	// Similar to version but with more detail
	return runMigrateVersion(cmd, args)
}

func runMigrateCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	timestamp := time.Now().Unix()
	
	upFile := filepath.Join("internal/database/migrations", 
		fmt.Sprintf("%d_%s.up.sql", timestamp, name))
	downFile := filepath.Join("internal/database/migrations", 
		fmt.Sprintf("%d_%s.down.sql", timestamp, name))
	
	// Create up migration
	if err := os.WriteFile(upFile, []byte("-- "+name+" up\n"), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}
	
	// Create down migration
	if err := os.WriteFile(downFile, []byte("-- "+name+" down\n"), 0644); err != nil {
		return fmt.Errorf("failed to create down migration: %w", err)
	}
	
	fmt.Printf("Created migrations:\n  %s\n  %s\n", upFile, downFile)
	return nil
}
```

## Dependencies
- github.com/spf13/cobra
- github.com/go-chi/chi/v5
- github.com/golang-migrate/migrate/v4
- github.com/nhalm/dbutil
- Standard library slog

## Success Criteria
- Environment-only configuration (no config files)
- Structured logging with slog
- Database migrations work correctly
- Graceful shutdown implemented
- Request logging middleware integrated