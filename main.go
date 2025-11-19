package main

import (
	"fmt"
	"os"

	"github.com/javanhut/genesys/commands/interact"
	"github.com/javanhut/genesys/commands/discover"
	"github.com/javanhut/genesys/commands/execute"
	"github.com/javanhut/genesys/commands/config"
	"github.com/javanhut/genesys/commands/state"
	"github.com/javanhut/genesys/commands/version"
	"github.com/javanhut/genesys/commands/cache"

	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "genesys",
		Short: "Interactive cloud resource management tool",
		Long: `Genesys is a simplicity-first Infrastructure as a Service tool that 
focuses on outcomes rather than resources. It provides an interactive approach 
to cloud resource management with TOML-based configuration and dry-run capabilities.

Key features:
  • Interactive workflows for resource creation
  • Multi-cloud support (AWS, GCP, Azure, Tencent)
  • Configuration-driven resource lifecycle
  • Dry-run capability for safe previews
  • Direct API integration for fast performance

Get started:
  1. Configure provider: genesys config setup
  2. Create resources:    genesys interact
  3. Deploy safely:       genesys execute config.toml --dry-run
  4. Deploy for real:     genesys execute config.toml`,
		Version: fmt.Sprintf("%s (%s)", Version, commit),
	}

	// Add commands
	rootCmd.AddCommand(execute.NewExecuteCommand())
	rootCmd.AddCommand(interact.NewInteractCommand())
	rootCmd.AddCommand(discover.NewDiscoverCommand())
	rootCmd.AddCommand(config.NewConfigCommand())
	rootCmd.AddCommand(cache.NewCacheCommand())
	rootCmd.AddCommand(state.NewStateCommand())
	rootCmd.AddCommand(version.NewVersionCommand(Version, commit))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
