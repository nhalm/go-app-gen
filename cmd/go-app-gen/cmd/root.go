package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Build information
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

var (
	rootCmd = &cobra.Command{
		Use:   "go-app-gen",
		Short: "Generate Go applications based on proven architecture patterns",
		Long: `go-app-gen is a CLI tool for generating Go applications with clean architecture,
database integration, and modern tooling. It creates projects based on proven patterns
including Cobra CLI, Viper configuration, clean architecture layers, and comprehensive
testing setups.`,
		Version: version,
	}
)

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Initialize viper
	viper.SetConfigName("go-app-gen")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config")
	viper.AutomaticEnv()

	// Set up version template
	rootCmd.SetVersionTemplate(`{{printf "%s version %s\n" .Name .Version}}` +
		fmt.Sprintf("commit: %s\n", commit) +
		fmt.Sprintf("built on: %s\n", buildDate))

	// Add version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  "Display detailed version information about go-app-gen",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("go-app-gen version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built on: %s\n", buildDate)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Register subcommands
	rootCmd.AddCommand(createCmd)
}