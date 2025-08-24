package commands

import (
	"context"
	"fmt"

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
		Args: cobra.ArbitraryArgs,
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

	// Load configuration if provided
	var cfg *config.Config
	if configFile != "" {
		var err error
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

	// If we have a config file but no command line intent, execute based on config
	if configFile != "" && len(args) == 0 {
		return executeFromConfig(ctx, cfg)
	}

	// Parse the intent from command line arguments
	if len(args) == 0 {
		return fmt.Errorf("no intent specified")
	}

	parser := intent.NewParser()
	parsedIntent, err := parser.Parse(args)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
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

// executeFromConfig executes based on configuration file content
func executeFromConfig(ctx context.Context, cfg *config.Config) error {
	fmt.Println("Executing from configuration file...")
	
	// Get the provider
	p, err := provider.Get(cfg.Provider, map[string]string{
		"region": cfg.Region,
	})
	if err != nil {
		// For now, create a mock provider for testing
		fmt.Printf("Note: Using mock provider (real provider not yet implemented)\n\n")
		p = provider.NewMockProvider(cfg.Provider, cfg.Region)
	}

	// Create planner (for future use)
	_ = planner.New(p)

	// Process outcomes if they exist
	if len(cfg.Outcomes) > 0 {
		for name, outcome := range cfg.Outcomes {
			fmt.Printf("Processing outcome: %s\n", name)
			// For now, just show what would be planned
			fmt.Printf("- Type: %s\n", name)
			if outcome.Domain != "" {
				fmt.Printf("- Domain: %s\n", outcome.Domain)
			}
			if outcome.Runtime != "" {
				fmt.Printf("- Runtime: %s\n", outcome.Runtime)
			}
			fmt.Println()
		}
	}

	// Process resources if they exist
	if len(cfg.Resources.Compute) > 0 || len(cfg.Resources.Storage) > 0 || 
	   len(cfg.Resources.Database) > 0 || len(cfg.Resources.Serverless) > 0 {
		
		fmt.Println("Processing resources from configuration:")
		
		// Process compute resources
		for _, compute := range cfg.Resources.Compute {
			fmt.Printf("- Compute: %s (%s, count: %d)\n", compute.Name, compute.Type, compute.Count)
		}
		
		// Process storage resources  
		for _, storage := range cfg.Resources.Storage {
			fmt.Printf("- Storage: %s (%s)\n", storage.Name, storage.Type)
		}
		
		// Process database resources
		for _, database := range cfg.Resources.Database {
			fmt.Printf("- Database: %s (%s %s, %dGB)\n", database.Name, database.Engine, database.Size, database.Storage)
		}
		
		// Process serverless resources
		for _, serverless := range cfg.Resources.Serverless {
			fmt.Printf("- Function: %s (%s, %dMB)\n", serverless.Name, serverless.Runtime, serverless.Memory)
		}
		
		fmt.Println()
		fmt.Println("Note: Full resource execution from config not yet implemented")
	}

	return nil
}