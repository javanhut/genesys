package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/javanhut/genesys/pkg/config"
	"github.com/javanhut/genesys/pkg/intent"
	"github.com/javanhut/genesys/pkg/planner"
	"github.com/javanhut/genesys/pkg/provider"
	"github.com/spf13/cobra"
)

var (
	applyFlag    bool
	configFile   string
	providerName string
	region       string
	outputFormat string
)

// NewExecuteCommand creates the execute command
func NewExecuteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute [intent]",
		Short: "Execute an infrastructure intent",
		Long: `Execute an infrastructure intent. By default, this shows a preview of changes.
Use --apply to actually make the changes.

Examples:
  genesys execute bucket my-bucket           # Preview bucket creation
  genesys execute static-site --apply        # Deploy static site
  genesys execute network --apply            # Create network infrastructure`,
		Args: cobra.MinimumNArgs(1),
		RunE: runExecute,
	}

	cmd.Flags().BoolVar(&applyFlag, "apply", false, "Apply the changes (default is preview only)")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file (YAML or TOML)")
	cmd.Flags().StringVar(&providerName, "provider", "aws", "Cloud provider (aws|gcp|azure)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud region")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "human", "Output format (human|json)")

	return cmd
}

func runExecute(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse the intent from command line arguments
	parser := intent.NewParser()
	parsedIntent, err := parser.Parse(args)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
	}

	// Load configuration if provided
	var cfg *config.Config
	if configFile != "" {
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// Create default config from flags
		cfg = &config.Config{
			Provider: providerName,
			Region:   region,
		}
	}

	// Get the provider
	p, err := provider.Get(cfg.Provider, map[string]string{
		"region": cfg.Region,
	})
	if err != nil {
		// For now, create a mock provider for testing
		fmt.Printf("Note: Using mock provider (real provider not yet implemented)\n\n")
		p = provider.NewMockProvider(cfg.Provider, cfg.Region)
	}

	// Create planner
	plnr := planner.New(p)

	// Generate plan
	plan, err := plnr.PlanFromIntent(ctx, parsedIntent)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// Display plan
	if outputFormat == "json" {
		fmt.Println(plan.ToJSON())
	} else {
		fmt.Println(plan.ToHumanReadable())
	}

	// Apply if requested
	if applyFlag {
		fmt.Println("\nApplying changes...")
		// TODO: Implement executor
		fmt.Println("Executor not yet implemented - this is a preview of what would happen")
	} else {
		fmt.Println("\nNote: This is a preview. Use --apply to make these changes.")
	}

	return nil
}