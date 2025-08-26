package commands

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/javanhut/genesys/pkg/config"
	"github.com/spf13/cobra"
)

// NewInteractCommand creates the interactive command
func NewInteractCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interact",
		Short: "Interactive resource creation wizard",
		Long: `Start an interactive wizard for cloud resource creation.

This guided workflow will:
  1. Select your cloud provider (AWS, GCP, Azure, Tencent)
  2. Choose resource type (S3 Storage Bucket, Compute Instance, etc.)
  3. Configure resource settings through step-by-step prompts
  4. Generate YAML configuration file for deployment

Example workflow:
  genesys interact
  # Follow prompts to create configuration
  # Saves: s3-mybucket-1234567890.yaml
  
Next steps after interactive mode:
  genesys execute s3-mybucket-*.yaml --dry-run   # Preview
  genesys execute s3-mybucket-*.yaml             # Deploy`,
		RunE: runInteract,
	}
}

func runInteract(cmd *cobra.Command, args []string) error {
	fmt.Println("Welcome to Genesys Interactive Mode!")
	fmt.Println("====================================")
	fmt.Println()

	// First, select cloud provider
	var provider string
	providerPrompt := &survey.Select{
		Message: "Select cloud provider:",
		Options: []string{
			"aws",
			"gcp",
			"azure",
			"tencent",
		},
		Description: func(value string, index int) string {
			switch value {
			case "aws":
				return "Amazon Web Services"
			case "gcp":
				return "Google Cloud Platform"
			case "azure":
				return "Microsoft Azure"
			case "tencent":
				return "Tencent Cloud"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(providerPrompt, &provider); err != nil {
		return err
	}

	// Then, select resource type
	var resourceType string
	resourcePrompt := &survey.Select{
		Message: "What type of resource would you like to create?",
		Options: []string{
			"S3 Storage Bucket",
			"Compute Instance", 
			"Database",
			"Function",
			"Network",
		},
	}
	if err := survey.AskOne(resourcePrompt, &resourceType); err != nil {
		return err
	}

	// Based on provider and resource type, start the specific workflow
	switch resourceType {
	case "S3 Storage Bucket":
		return interactS3Bucket(provider)
	case "Compute Instance":
		return interactCompute(provider)
	case "Database":
		return interactDatabase(provider)
	case "Function":
		return interactFunction(provider)
	case "Network":
		return interactNetwork(provider)
	default:
		fmt.Printf("Resource type '%s' not yet implemented for provider '%s'\n", resourceType, provider)
	}

	return nil
}


// interactS3Bucket handles S3 bucket creation workflow
func interactS3Bucket(provider string) error {
	fmt.Printf("\n🪣 Creating S3 Bucket Configuration for %s\n", provider)
	fmt.Println("==========================================")
	fmt.Println()

	// Check if provider is configured
	if err := checkProviderConfig(provider); err != nil {
		fmt.Printf("❌ Provider '%s' not configured. Run 'genesys config setup' first.\n", provider)
		return err
	}

	// Use the existing S3 interactive configuration
	s3Config, err := config.NewInteractiveS3Config()
	if err != nil {
		return fmt.Errorf("failed to initialize S3 configuration: %w", err)
	}

	// Generate configuration interactively
	bucketConfig, fileName, err := s3Config.CreateBucketConfig()
	if err != nil {
		return fmt.Errorf("failed to create bucket configuration: %w", err)
	}

	// Save configuration
	filePath, err := s3Config.SaveConfig(bucketConfig, fileName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✅ Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", fileName)
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", fileName)
	fmt.Printf("  • Deploy the bucket: genesys execute %s\n", fileName)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", fileName)

	return nil
}

func checkProviderConfig(provider string) error {
	interactiveConfig, err := config.NewInteractiveConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	_, err = interactiveConfig.LoadProviderConfig(provider)
	if err != nil {
		return fmt.Errorf("provider '%s' not configured", provider)
	}

	return nil
}

// Placeholder functions for other resource types
func interactCompute(provider string) error {
	fmt.Printf("Compute instance creation for %s not yet implemented\n", provider)
	return nil
}

func interactDatabase(provider string) error {
	fmt.Printf("Database creation for %s not yet implemented\n", provider)
	return nil
}

func interactFunction(provider string) error {
	fmt.Printf("Function creation for %s not yet implemented\n", provider)
	return nil
}

func interactNetwork(provider string) error {
	fmt.Printf("Network creation for %s not yet implemented\n", provider)
	return nil
}