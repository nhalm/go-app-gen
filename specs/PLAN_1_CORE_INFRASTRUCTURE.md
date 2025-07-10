# Plan 1: Core Infrastructure

## Overview
Update the generator.go to support post-generation validation and container-based tooling.

## Tasks

### 1.1 Update generator.go

Add the following to the Generator struct and methods:

```go
// Add to imports
import (
    "os/exec"
    "context"
)

// Add to Generator struct
type Generator struct {
    outputDir string
    verbose   bool  // Add for debugging
}

// Add new method
func (g *Generator) runCommand(ctx context.Context, projectDir string, name string, args ...string) error {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Dir = projectDir
    
    if g.verbose {
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
    }
    
    return cmd.Run()
}

// Add PostProcess method
func (g *Generator) PostProcess(projectDir string) error {
    ctx := context.Background()
    
    fmt.Println("🔄 Running post-generation tasks...")
    
    // Start database container
    if err := g.runCommand(ctx, projectDir, "docker-compose", "up", "-d", "db"); err != nil {
        return fmt.Errorf("failed to start database: %w", err)
    }
    
    // Initialize go module
    if err := g.runCommand(ctx, projectDir, "go", "mod", "init", data.ModuleName); err != nil {
        return fmt.Errorf("failed to initialize go module: %w", err)
    }
    
    // Run go mod tidy in container
    if err := g.runCommand(ctx, projectDir, "docker-compose", "run", "--rm", "dev", "go", "mod", "tidy"); err != nil {
        return fmt.Errorf("failed to run go mod tidy: %w", err)
    }
    
    // Run migrations
    if err := g.runCommand(ctx, projectDir, "docker-compose", "run", "--rm", "dev", "make", "migrate-up"); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    // Generate sqlc
    if err := g.runCommand(ctx, projectDir, "docker-compose", "run", "--rm", "dev", "make", "sqlc"); err != nil {
        return fmt.Errorf("failed to generate sqlc: %w", err)
    }
    
    // Build to verify
    if err := g.runCommand(ctx, projectDir, "docker-compose", "run", "--rm", "dev", "go", "build", "./..."); err != nil {
        return fmt.Errorf("failed to build: %w", err)
    }
    
    fmt.Println("✅ Post-generation tasks completed")
    return nil
}
```

### 1.2 Update Generate method

Modify the Generate method to call PostProcess:

```go
func (g *Generator) Generate(config *ProjectConfig) error {
    // ... existing code ...
    
    // Process templates
    if err := g.processTemplates(data, projectDir); err != nil {
        return err
    }
    
    // Run post-processing
    if err := g.PostProcess(projectDir); err != nil {
        return fmt.Errorf("post-processing failed: %w", err)
    }
    
    return nil
}
```

### 1.3 Update TemplateData

Add new fields to TemplateData for enhanced functionality:

```go
type TemplateData struct {
    AppName           string
    ModuleName        string
    Domain            string
    DomainTitle       string
    DomainPlural      string
    DomainLower       string  // Add lowercase domain
    Description       string
    Author            string
    PackageImportPath string
    GoVersion         string  // Add Go version
    HasFeature        func(string) bool
}
```

Update data creation:
```go
data := &TemplateData{
    // ... existing fields ...
    DomainLower: strings.ToLower(config.Domain),
    GoVersion:   "1.23",
}
```

### 1.4 Add placeholder handling

Update getOutputPath to handle more placeholders:

```go
func (g *Generator) getOutputPath(templatePath string, data *TemplateData) string {
    // ... existing code ...
    
    // Replace placeholders in path
    path = strings.ReplaceAll(path, "{{.AppName}}", data.AppName)
    path = strings.ReplaceAll(path, "{{.Domain}}", data.Domain)
    path = strings.ReplaceAll(path, "{{.domain}}", data.DomainLower)  // Add lowercase
    path = strings.ReplaceAll(path, "{{.domain_plural}}", data.DomainPlural)  // Add plural
    
    return path
}
```

## Dependencies
- Go 1.23
- Docker and docker-compose installed on host
- Container commands will handle all other dependencies

## Success Criteria
- Generator can run container commands
- Post-processing validates the generated app
- All placeholders work correctly in paths and templates