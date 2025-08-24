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
		Short: "Simplicity-first IaaS tool for cloud resource management",
		Long: `Genesys is a simplicity-first Infrastructure as a Service tool that 
focuses on outcomes rather than resources. It provides a discovery-first 
approach to cloud resource management with human-readable plans.`,
		Version: fmt.Sprintf("%s (%s)", version, commit),
	}

	// Add commands
	rootCmd.AddCommand(commands.NewExecuteCommand())
	rootCmd.AddCommand(commands.NewInteractCommand())
	rootCmd.AddCommand(commands.NewDiscoverCommand())
	rootCmd.AddCommand(commands.NewVersionCommand(version, commit))

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}