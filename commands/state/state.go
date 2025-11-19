package state

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/javanhut/genesys/pkg/state"
	"github.com/spf13/cobra"
)

// NewStateCommand creates the state command
func NewStateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage Genesys state tracking",
		Long: `Manage Genesys state tracking for deployed cloud resources.

This command allows you to:
  • List all tracked resources
  • View state file details
  • Validate resources against cloud providers
  • Clean up orphaned entries
  • Export and import state data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newStateListCommand())
	cmd.AddCommand(newStateShowCommand())
	cmd.AddCommand(newStatePathCommand())
	cmd.AddCommand(newStateValidateCommand())
	cmd.AddCommand(newStateCleanCommand())
	cmd.AddCommand(newStateExportCommand())
	cmd.AddCommand(newStateImportCommand())

	return cmd
}

// newStateListCommand creates the state list subcommand
func newStateListCommand() *cobra.Command {
	var resourceType, provider, region string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tracked resources",
		Long: `List all resources tracked in the Genesys state file.

You can filter by:
  • Resource type (ec2, s3, lambda, etc.)
  • Cloud provider (aws, gcp, azure, tencent)
  • Region

Examples:
  genesys state list                        # List all resources
  genesys state list --type ec2             # List only EC2 instances
  genesys state list --provider aws         # List only AWS resources
  genesys state list --region us-east-1     # List resources in us-east-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			localState, err := state.LoadLocalState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			if len(localState.Resources) == 0 {
				fmt.Println("No resources tracked in state file.")
				fmt.Println("")
				fmt.Println("Resources will be automatically tracked when you:")
				fmt.Println("  • Deploy with 'genesys execute <config-file>'")
				fmt.Println("  • Create resources with 'genesys interact'")
				return nil
			}

			// Filter resources
			filtered := localState.Resources
			if resourceType != "" {
				var temp []state.ResourceRecord
				for _, r := range filtered {
					if strings.EqualFold(r.Type, resourceType) {
						temp = append(temp, r)
					}
				}
				filtered = temp
			}
			if provider != "" {
				var temp []state.ResourceRecord
				for _, r := range filtered {
					if strings.EqualFold(r.Provider, provider) {
						temp = append(temp, r)
					}
				}
				filtered = temp
			}
			if region != "" {
				var temp []state.ResourceRecord
				for _, r := range filtered {
					if strings.EqualFold(r.Region, region) {
						temp = append(temp, r)
					}
				}
				filtered = temp
			}

			if len(filtered) == 0 {
				fmt.Println("No resources match the specified filters.")
				return nil
			}

			fmt.Printf("Tracked Resources (%d):\n", len(filtered))
			fmt.Println(strings.Repeat("=", 80))
			fmt.Println("")

			for i, resource := range filtered {
				fmt.Printf("%d. [%s] %s\n", i+1, strings.ToUpper(resource.Type), resource.Name)
				fmt.Printf("   ID: %s\n", resource.ID)
				fmt.Printf("   Provider: %s\n", strings.ToUpper(resource.Provider))
				fmt.Printf("   Region: %s\n", resource.Region)
				fmt.Printf("   Created: %s\n", resource.CreatedAt.Format("2006-01-02 15:04:05"))
				if resource.ConfigFile != "" {
					fmt.Printf("   Config: %s\n", resource.ConfigFile)
				}
				if len(resource.Tags) > 0 {
					fmt.Printf("   Tags: ")
					tagStrs := []string{}
					for k, v := range resource.Tags {
						tagStrs = append(tagStrs, fmt.Sprintf("%s=%s", k, v))
					}
					fmt.Println(strings.Join(tagStrs, ", "))
				}
				fmt.Println("")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type (ec2, s3, lambda)")
	cmd.Flags().StringVar(&provider, "provider", "", "Filter by cloud provider (aws, gcp, azure)")
	cmd.Flags().StringVar(&region, "region", "", "Filter by region")

	return cmd
}

// newStateShowCommand creates the state show subcommand
func newStateShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show state file details and summary",
		Long: `Show detailed information about the Genesys state file.

This displays:
  • State file location
  • Total number of tracked resources
  • Resource breakdown by type and provider
  • State file size and last modified time`,
		RunE: func(cmd *cobra.Command, args []string) error {
			localState, err := state.LoadLocalState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			// Get state file info
			homeDir, _ := os.UserHomeDir()
			stateFilePath := fmt.Sprintf("%s/.genesys-state.json", homeDir)

			fmt.Println("Genesys State Information:")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println("")

			// File info
			if fileInfo, err := os.Stat(stateFilePath); err == nil {
				fmt.Printf("State File: %s\n", stateFilePath)
				fmt.Printf("Size: %d bytes\n", fileInfo.Size())
				fmt.Printf("Last Modified: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("State File: %s (not created yet)\n", stateFilePath)
			}
			fmt.Println("")

			// Resource summary
			fmt.Printf("Total Resources: %d\n", len(localState.Resources))
			fmt.Println("")

			if len(localState.Resources) > 0 {
				// By type
				byType := make(map[string]int)
				for _, r := range localState.Resources {
					byType[r.Type]++
				}
				fmt.Println("By Resource Type:")
				for t, count := range byType {
					fmt.Printf("  • %s: %d\n", strings.ToUpper(t), count)
				}
				fmt.Println("")

				// By provider
				byProvider := make(map[string]int)
				for _, r := range localState.Resources {
					byProvider[r.Provider]++
				}
				fmt.Println("By Cloud Provider:")
				for p, count := range byProvider {
					fmt.Printf("  • %s: %d\n", strings.ToUpper(p), count)
				}
				fmt.Println("")

				// By region
				byRegion := make(map[string]int)
				for _, r := range localState.Resources {
					byRegion[r.Region]++
				}
				fmt.Println("By Region:")
				for region, count := range byRegion {
					fmt.Printf("  • %s: %d\n", region, count)
				}
			}

			return nil
		},
	}

	return cmd
}

// newStatePathCommand creates the state path subcommand
func newStatePathCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show state file location",
		Long: `Show the path to the Genesys state file.

The state file tracks all resources created by Genesys and is stored
in your home directory as .genesys-state.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, _ := os.UserHomeDir()
			stateFilePath := fmt.Sprintf("%s/.genesys-state.json", homeDir)

			if _, err := os.Stat(stateFilePath); err == nil {
				fmt.Printf("State file: %s ✓\n", stateFilePath)
			} else {
				fmt.Printf("State file: %s (not created yet)\n", stateFilePath)
			}

			return nil
		},
	}

	return cmd
}

// newStateValidateCommand creates the state validate subcommand
func newStateValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate tracked resources against cloud providers",
		Long: `Validate that tracked resources still exist in their respective cloud providers.

This command will:
  • Check each resource against the cloud provider API
  • Report resources that no longer exist
  • Identify orphaned state entries
  
Note: This requires valid cloud provider credentials.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			localState, err := state.LoadLocalState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			if len(localState.Resources) == 0 {
				fmt.Println("No resources to validate.")
				return nil
			}

			fmt.Printf("Validating %d resources...\n", len(localState.Resources))
			fmt.Println("")

			// TODO: Implement actual validation against cloud providers
			// For now, just report what would be validated
			fmt.Println("[INFO] Resource validation requires provider integration")
			fmt.Println("[INFO] This feature will validate resources in a future update")
			fmt.Println("")

			for i, resource := range localState.Resources {
				fmt.Printf("%d. [%s] %s\n", i+1, strings.ToUpper(resource.Type), resource.Name)
				fmt.Printf("   ID: %s\n", resource.ID)
				fmt.Printf("   Provider: %s | Region: %s\n", strings.ToUpper(resource.Provider), resource.Region)
				fmt.Println("   Status: [PENDING] Validation not yet implemented")
				fmt.Println("")
			}

			return nil
		},
	}

	return cmd
}

// newStateCleanCommand creates the state clean subcommand
func newStateCleanCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove orphaned or invalid entries from state",
		Long: `Clean the state file by removing orphaned or invalid resource entries.

This command will:
  • Identify resources that no longer exist
  • Remove invalid or corrupted entries
  • Compact the state file

Use --dry-run to preview changes without modifying the state.

Examples:
  genesys state clean --dry-run    # Preview what would be cleaned
  genesys state clean              # Actually clean the state`,
		RunE: func(cmd *cobra.Command, args []string) error {
			localState, err := state.LoadLocalState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			if len(localState.Resources) == 0 {
				fmt.Println("No resources in state file.")
				return nil
			}

			if dryRun {
				fmt.Println("[DRY RUN] Preview of state cleaning:")
				fmt.Println("")
			} else {
				fmt.Println("Cleaning state file...")
				fmt.Println("")
			}

			// TODO: Implement actual validation and cleaning logic
			// For now, just validate basic data integrity
			invalidCount := 0
			validResources := []state.ResourceRecord{}

			for _, resource := range localState.Resources {
				// Check for required fields
				if resource.ID == "" || resource.Name == "" || resource.Type == "" {
					invalidCount++
					if dryRun {
						fmt.Printf("[WOULD REMOVE] Invalid resource (missing required fields): %s\n", resource.Name)
					} else {
						fmt.Printf("[REMOVED] Invalid resource: %s\n", resource.Name)
					}
					continue
				}
				validResources = append(validResources, resource)
			}

			if invalidCount == 0 {
				fmt.Println("[OK] No invalid entries found")
				fmt.Printf("All %d resources are valid\n", len(localState.Resources))
				return nil
			}

			if !dryRun {
				localState.Resources = validResources
				if err := localState.SaveLocalState(); err != nil {
					return fmt.Errorf("failed to save cleaned state: %w", err)
				}
				fmt.Println("")
				fmt.Printf("[OK] Removed %d invalid entries\n", invalidCount)
				fmt.Printf("Remaining resources: %d\n", len(validResources))
			} else {
				fmt.Println("")
				fmt.Printf("[DRY RUN] Would remove %d invalid entries\n", invalidCount)
				fmt.Printf("Would keep %d valid resources\n", len(validResources))
				fmt.Println("")
				fmt.Println("Run without --dry-run to apply changes")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying state")

	return cmd
}

// newStateExportCommand creates the state export subcommand
func newStateExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <file>",
		Short: "Export state to a JSON file",
		Long: `Export the current Genesys state to a JSON file.

This is useful for:
  • Backing up your state
  • Sharing state with team members
  • Migrating state to another system
  • Inspecting state in external tools

Examples:
  genesys state export backup.json         # Export to backup.json
  genesys state export state-backup.json   # Export with custom name`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFile := args[0]

			localState, err := state.LoadLocalState()
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}

			// Export to JSON
			data, err := json.MarshalIndent(localState, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal state: %w", err)
			}

			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write export file: %w", err)
			}

			fmt.Printf("[OK] State exported to: %s\n", outputFile)
			fmt.Printf("Resources exported: %d\n", len(localState.Resources))

			return nil
		},
	}

	return cmd
}

// newStateImportCommand creates the state import subcommand
func newStateImportCommand() *cobra.Command {
	var merge bool

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import state from a JSON file",
		Long: `Import Genesys state from a JSON file.

By default, this will replace the current state. Use --merge to
merge the imported resources with existing state.

WARNING: Importing state will overwrite your current state unless
you use the --merge flag. Make sure to backup first!

Examples:
  genesys state import backup.json           # Replace current state
  genesys state import backup.json --merge   # Merge with current state`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			importFile := args[0]

			// Read import file
			data, err := os.ReadFile(importFile)
			if err != nil {
				return fmt.Errorf("failed to read import file: %w", err)
			}

			var importState state.LocalState
			if err := json.Unmarshal(data, &importState); err != nil {
				return fmt.Errorf("failed to parse import file: %w", err)
			}

			if merge {
				// Load current state and merge
				currentState, err := state.LoadLocalState()
				if err != nil {
					return fmt.Errorf("failed to load current state: %w", err)
				}

				// Merge resources (avoiding duplicates by ID)
				existingIDs := make(map[string]bool)
				for _, r := range currentState.Resources {
					existingIDs[r.ID] = true
				}

				mergedCount := 0
				for _, r := range importState.Resources {
					if !existingIDs[r.ID] {
						currentState.Resources = append(currentState.Resources, r)
						mergedCount++
					}
				}

				if err := currentState.SaveLocalState(); err != nil {
					return fmt.Errorf("failed to save merged state: %w", err)
				}

				fmt.Printf("[OK] State merged successfully\n")
				fmt.Printf("Added resources: %d\n", mergedCount)
				fmt.Printf("Total resources: %d\n", len(currentState.Resources))
			} else {
				// Replace current state
				if err := importState.SaveLocalState(); err != nil {
					return fmt.Errorf("failed to save imported state: %w", err)
				}

				fmt.Printf("[OK] State imported successfully\n")
				fmt.Printf("Total resources: %d\n", len(importState.Resources))
				fmt.Println("")
				fmt.Println("WARNING: Previous state has been replaced")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&merge, "merge", false, "Merge with existing state instead of replacing")

	return cmd
}
