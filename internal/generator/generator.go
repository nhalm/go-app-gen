package generator

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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
	Description       string
	Author            string
	PackageImportPath string
	HasFeature        func(string) bool
}

// Generator handles project generation
type Generator struct {
	outputDir string
}

// New creates a new generator
func New(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
	}
}

// Generate creates a new project based on the configuration
func (g *Generator) Generate(config *ProjectConfig) error {
	// Create template data
	data := &TemplateData{
		AppName:           config.AppName,
		ModuleName:        config.ModuleName,
		Domain:            config.Domain,
		DomainTitle:       strings.Title(config.Domain),
		DomainPlural:      pluralize(config.Domain),
		Description:       config.Description,
		Author:            config.Author,
		PackageImportPath: config.ModuleName,
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
	return g.processTemplates(data, projectDir)
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
	
	return path
}

// pluralize is a simple pluralizer (can be enhanced)
func pluralize(word string) string {
	if strings.HasSuffix(word, "y") {
		return strings.TrimSuffix(word, "y") + "ies"
	}
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "sh") || strings.HasSuffix(word, "ch") {
		return word + "es"
	}
	return word + "s"
}