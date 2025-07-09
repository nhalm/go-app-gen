package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nhalm/go-app-gen/internal/generator"
)

// Config holds the configuration for project generation
type Config struct {
	AppName     string
	ModuleName  string
	Domain      string
	Description string
	Author      string
	OutputDir   string
	Features    []string
}

var (
	config Config
	interactive bool
)

var createCmd = &cobra.Command{
	Use:   "create [project-name]",
	Short: "Create a new Go application",
	Long: `Create a new Go application with clean architecture patterns.
	
This command generates a complete Go application structure including:
- Cobra CLI with subcommands
- Clean architecture (api/service/repository layers)
- Database integration with migrations
- Configuration management with Viper
- Comprehensive testing setup
- Docker and development tooling

Examples:
  go-app-gen create myapp
  go-app-gen create myapp --module github.com/myorg/myapp --domain product
  go-app-gen create --interactive`,
	Args: func(cmd *cobra.Command, args []string) error {
		if interactive {
			return nil
		}
		if len(args) < 1 {
			return errors.New("project name is required when not using --interactive")
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&config.ModuleName, "module", "m", "", "Go module name (e.g., github.com/user/project)")
	createCmd.Flags().StringVarP(&config.Domain, "domain", "d", "", "Primary domain entity (e.g., user, product, order)")
	createCmd.Flags().StringVar(&config.Description, "description", "", "Project description")
	createCmd.Flags().StringVar(&config.Author, "author", "", "Author name")
	createCmd.Flags().StringVarP(&config.OutputDir, "output", "o", ".", "Output directory")
	createCmd.Flags().StringSliceVar(&config.Features, "features", []string{}, "Additional features to include")
	createCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode")
}

func runCreate(cmd *cobra.Command, args []string) error {
	var err error
	
	if interactive {
		err = runInteractiveMode()
	} else {
		config.AppName = args[0]
		err = runDirectMode()
	}
	
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Generate the project
	gen := generator.New(config.OutputDir)
	
	projectConfig := &generator.ProjectConfig{
		AppName:     config.AppName,
		ModuleName:  config.ModuleName,
		Domain:      config.Domain,
		Description: config.Description,
		Author:      config.Author,
		Features:    config.Features,
	}
	
	if err := gen.Generate(projectConfig); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	fmt.Printf("✅ Successfully created project '%s' in %s\n", config.AppName, filepath.Join(config.OutputDir, config.AppName))
	fmt.Printf("📁 Project structure generated with module: %s\n", config.ModuleName)
	fmt.Printf("🚀 To get started:\n")
	fmt.Printf("   cd %s\n", config.AppName)
	fmt.Printf("   go mod tidy\n")
	fmt.Printf("   make help\n")
	
	return nil
}

func runDirectMode() error {
	// Set defaults if not provided
	if config.ModuleName == "" {
		config.ModuleName = fmt.Sprintf("github.com/user/%s", config.AppName)
	}
	if config.Domain == "" {
		config.Domain = "item"
	}
	if config.Description == "" {
		config.Description = fmt.Sprintf("A %s management API", config.Domain)
	}
	if config.Author == "" {
		config.Author = "Developer"
	}
	
	return validateConfig()
}

func runInteractiveMode() error {
	fmt.Println("🚀 Welcome to go-app-gen!")
	fmt.Println("Let's create your Go application step by step.")
	fmt.Println()
	
	// Get project name
	config.AppName = promptString("Project name", "myapp")
	
	// Get module name
	defaultModule := fmt.Sprintf("github.com/user/%s", config.AppName)
	config.ModuleName = promptString("Go module name", defaultModule)
	
	// Get domain
	config.Domain = promptString("Primary domain entity (e.g., user, product, order)", "item")
	
	// Get description
	defaultDesc := fmt.Sprintf("A %s management API", config.Domain)
	config.Description = promptString("Project description", defaultDesc)
	
	// Get author
	config.Author = promptString("Author name", "Developer")
	
	// Get output directory
	config.OutputDir = promptString("Output directory", ".")
	
	return validateConfig()
}

func promptString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	var input string
	fmt.Scanln(&input)
	
	if input == "" {
		return defaultValue
	}
	return strings.TrimSpace(input)
}

func validateConfig() error {
	if config.AppName == "" {
		return errors.New("app name is required")
	}
	
	if config.ModuleName == "" {
		return errors.New("module name is required")
	}
	
	if config.Domain == "" {
		return errors.New("domain is required")
	}
	
	// Check if output directory exists
	if _, err := os.Stat(config.OutputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", config.OutputDir)
	}
	
	// Check if target directory already exists
	targetDir := filepath.Join(config.OutputDir, config.AppName)
	if _, err := os.Stat(targetDir); err == nil {
		// Directory exists, check if it's empty
		empty, err := isDirEmpty(targetDir)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		
		if !empty {
			// Directory has contents, ask user for confirmation
			fmt.Printf("Directory '%s' already exists and contains files.\n", targetDir)
			fmt.Print("Do you want to recreate it? [y/N]: ")
			
			var response string
			fmt.Scanln(&response)
			
			if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
				return errors.New("operation cancelled")
			}
			
			// Remove existing directory
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("failed to remove existing directory: %w", err)
			}
		}
	}
	
	return nil
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()
	
	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil // Directory has at least one entry
	}
	
	return true, nil // Directory is empty
}