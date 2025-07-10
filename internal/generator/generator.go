package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jinzhu/inflection"
)

//go:embed templates/*
var templatesFS embed.FS

// ProjectConfig holds the configuration for project generation
type ProjectConfig struct {
	AppName     string
	ModuleName  string
	Domain      string
	Description string
	Author      string
	Features    []string
}

// TemplateData holds the data passed to templates
type TemplateData struct {
	AppName           string
	ModuleName        string
	Domain            string
	DomainTitle       string
	DomainPlural      string
	DomainPluralLower string
	DomainLower       string
	Description       string
	Author            string
	PackageImportPath string
	GoVersion         string
	HasFeature        func(string) bool
}

// Generator handles project generation
type Generator struct {
	outputDir string
	verbose   bool
}

// New creates a new generator
func New(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
		verbose:   false,
	}
}

// NewWithVerbose creates a new generator with verbose logging
func NewWithVerbose(outputDir string, verbose bool) *Generator {
	return &Generator{
		outputDir: outputDir,
		verbose:   verbose,
	}
}

// Generate creates a new project based on the configuration
func (g *Generator) Generate(config *ProjectConfig) error {
	// Create template data
	data := &TemplateData{
		AppName:           config.AppName,
		ModuleName:        config.ModuleName,
		Domain:            config.Domain,
		DomainTitle:       titleCase(config.Domain),
		DomainPlural:      inflection.Plural(config.Domain),
		DomainPluralLower: strings.ToLower(inflection.Plural(config.Domain)),
		DomainLower:       strings.ToLower(config.Domain),
		Description:       config.Description,
		Author:            config.Author,
		PackageImportPath: config.ModuleName,
		GoVersion:         "1.23",
		HasFeature: func(feature string) bool {
			for _, f := range config.Features {
				if f == feature {
					return true
				}
			}
			return false
		},
	}

	// Create project directory
	projectDir := filepath.Join(g.outputDir, config.AppName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Process templates
	if err := g.processTemplates(data, projectDir); err != nil {
		return err
	}

	// Run post-processing
	if err := g.PostProcess(projectDir, data); err != nil {
		return fmt.Errorf("post-processing failed: %w", err)
	}

	return nil
}

// processTemplates walks through the embedded templates and processes them
func (g *Generator) processTemplates(data *TemplateData, projectDir string) error {
	return fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read template file
		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		// Process the template
		tmpl, err := template.New(path).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Execute template
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		// Determine output path
		outputPath := g.getOutputPath(path, data)
		outputPath = filepath.Join(projectDir, outputPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
		}

		// Write file
		if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		return nil
	})
}

// getOutputPath converts template path to output path with substitutions
func (g *Generator) getOutputPath(templatePath string, data *TemplateData) string {
	// Remove "templates/" prefix
	path := strings.TrimPrefix(templatePath, "templates/")

	// Remove .tmpl extension
	if strings.HasSuffix(path, ".tmpl") {
		path = strings.TrimSuffix(path, ".tmpl")
	}

	// Replace placeholders in path
	path = strings.ReplaceAll(path, "{{.AppName}}", data.AppName)
	path = strings.ReplaceAll(path, "{{.Domain}}", data.Domain)
	path = strings.ReplaceAll(path, "{{.domain}}", data.DomainLower)
	path = strings.ReplaceAll(path, "{{.domain_plural}}", data.DomainPlural)

	return path
}

// runCommand executes a command in the specified directory
func (g *Generator) runCommand(ctx context.Context, projectDir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = projectDir

	if g.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// PostProcess runs post-generation validation and setup tasks
func (g *Generator) PostProcess(projectDir string, data *TemplateData) error {
	ctx := context.Background()

	fmt.Println("🔄 Running post-generation tasks...")

	// Initialize go module
	if err := g.runCommand(ctx, projectDir, "go", "mod", "init", data.ModuleName); err != nil {
		return fmt.Errorf("failed to initialize go module: %w", err)
	}

	// Generate SQLc code first (before go mod tidy)
	if err := g.runCommand(ctx, projectDir, "sqlc", "generate"); err != nil {
		// SQLc might not be installed, so warn but don't fail
		fmt.Printf("⚠️  SQLc generation failed: %v\n", err)
		fmt.Println("   Consider installing sqlc: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest")
		fmt.Println("   Or run 'make sqlc' in the project directory after setup")
	} else {
		fmt.Println("✅ SQLc code generation successful")
	}

	// Run go mod tidy (after SQLc generation)
	if err := g.runCommand(ctx, projectDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %w", err)
	}

	// Format generated Go code
	if err := g.runCommand(ctx, projectDir, "go", "fmt", "./..."); err != nil {
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Fix imports (if goimports is available)
	if err := g.runCommand(ctx, projectDir, "goimports", "-w", "."); err != nil {
		// goimports might not be installed, so just warn instead of failing
		fmt.Printf("⚠️  goimports not available or failed: %v\n", err)
		fmt.Println("   Consider installing goimports: go install golang.org/x/tools/cmd/goimports@latest")
	}

	// Try to build to verify syntax (but allow failure)
	if err := g.runCommand(ctx, projectDir, "go", "build", "./..."); err != nil {
		fmt.Printf("⚠️  Build failed (this is expected if dependencies require database): %v\n", err)
		fmt.Println("   Run 'make up' in the project directory to start the database and complete setup")
	} else {
		fmt.Println("✅ Build successful")
	}

	fmt.Println("✅ Post-generation tasks completed")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  cd " + filepath.Base(projectDir))
	fmt.Println("  make up      # Start the development environment")
	fmt.Println("  make help    # See all available commands")
	return nil
}

// titleCase converts a string to title case (alternative to deprecated strings.Title)
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
