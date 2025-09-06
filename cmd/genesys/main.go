package main

import (
	"fmt"
	"os"

	"github.com/javanhut/genesys/cmd/genesys/commands"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "genesys",
		Short: "Interactive cloud resource management tool",
		Long: `Genesys is a simplicity-first Infrastructure as a Service tool that 
focuses on outcomes rather than resources. It provides an interactive approach 
to cloud resource management with YAML-based configuration and dry-run capabilities.

Key features:
  • Interactive workflows for resource creation
  • Multi-cloud support (AWS, GCP, Azure, Tencent)
  • Configuration-driven resource lifecycle
  • Dry-run capability for safe previews
  • Direct API integration for fast performance

Get started:
  1. Configure provider: genesys config setup
  2. Create resources:    genesys interact
  3. Deploy safely:       genesys execute config.yaml --dry-run
  4. Deploy for real:     genesys execute config.yaml`,
		Version: fmt.Sprintf("%s (%s)", version, commit),
	}

	// Add commands
	rootCmd.AddCommand(commands.NewExecuteCommand())
	rootCmd.AddCommand(commands.NewInteractCommand())
	rootCmd.AddCommand(commands.NewDiscoverCommand())
	rootCmd.AddCommand(commands.NewConfigCommand())
	rootCmd.AddCommand(commands.NewCacheCommand())
	rootCmd.AddCommand(commands.NewStateCommand())
	rootCmd.AddCommand(commands.NewVersionCommand(version, commit))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
