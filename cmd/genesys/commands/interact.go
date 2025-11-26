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
  4. Generate TOML configuration file for deployment

Example workflow:
  genesys interact
  # Follow prompts to create configuration
  # Saves: s3-mybucket-1234567890.toml
  
Next steps after interactive mode:
  genesys execute s3-mybucket-*.toml             # Preview changes
  genesys execute s3-mybucket-*.toml --apply     # Deploy`,
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
	fmt.Printf("\nCreating S3 Bucket Configuration for %s\n", provider)
	fmt.Println("==========================================")
	fmt.Println()

	// Note: No credential check needed for interactive mode - only for execution

	// Use the existing S3 interactive configuration
	s3Config, err := config.NewInteractiveS3Config()
	if err != nil {
		return fmt.Errorf("failed to initialize S3 configuration: %w", err)
	}

	// Generate configuration interactively
	bucketConfig, bucketName, err := s3Config.CreateBucketConfig()
	if err != nil {
		return fmt.Errorf("failed to create bucket configuration: %w", err)
	}

	// Save configuration
	filePath, err := s3Config.SaveConfig(bucketConfig, bucketName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s\n", filePath)
	fmt.Printf("  • Deploy the bucket: genesys execute %s --apply\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute %s --delete\n", filePath)

	return nil
}

// interactCompute handles EC2 instance creation workflow
func interactCompute(provider string) error {
	fmt.Printf("\nCreating EC2 Instance Configuration for %s\n", provider)
	fmt.Println("==========================================")
	fmt.Println()

	// Note: No credential check needed for interactive mode - only for execution

	// Use the EC2 interactive configuration
	ec2Config, err := config.NewInteractiveEC2Config()
	if err != nil {
		return fmt.Errorf("failed to initialize EC2 configuration: %w", err)
	}

	// Generate configuration interactively
	instanceConfig, instanceName, err := ec2Config.CreateInstanceConfig()
	if err != nil {
		return fmt.Errorf("failed to create instance configuration: %w", err)
	}

	// Save configuration
	filePath, err := ec2Config.SaveConfig(instanceConfig, instanceName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s\n", filePath)
	fmt.Printf("  • Deploy the instance: genesys execute %s --apply\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute %s --delete\n", filePath)

	return nil
}

func interactDatabase(provider string) error {
	fmt.Printf("\nCreating Database Configuration for %s\n", provider)
	fmt.Println("==========================================")
	fmt.Println()

	// Only AWS DynamoDB is currently supported
	if provider != "aws" {
		fmt.Printf("Database creation is currently only supported for AWS provider\n")
		return nil
	}

	// Select database type
	var dbType string
	dbPrompt := &survey.Select{
		Message: "Select database type:",
		Options: []string{
			"DynamoDB (NoSQL)",
			"RDS (Relational)",
		},
		Description: func(value string, index int) string {
			switch value {
			case "DynamoDB (NoSQL)":
				return "Serverless NoSQL key-value and document database"
			case "RDS (Relational)":
				return "Managed relational database (PostgreSQL, MySQL, etc.)"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(dbPrompt, &dbType); err != nil {
		return err
	}

	switch dbType {
	case "DynamoDB (NoSQL)":
		return interactDynamoDB(provider)
	case "RDS (Relational)":
		fmt.Println("RDS configuration is not yet implemented")
		return nil
	default:
		fmt.Printf("Database type '%s' not yet implemented\n", dbType)
		return nil
	}
}

func interactDynamoDB(provider string) error {
	fmt.Printf("\nCreating DynamoDB Table Configuration\n")
	fmt.Println("======================================")
	fmt.Println()

	// Use the DynamoDB interactive configuration
	dynamoConfig, err := config.NewInteractiveDynamoDBConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize DynamoDB configuration: %w", err)
	}

	// Generate configuration interactively
	tableConfig, tableName, err := dynamoConfig.CreateTableConfig()
	if err != nil {
		return fmt.Errorf("failed to create table configuration: %w", err)
	}

	// Get region
	region, err := dynamoConfig.GetRegion()
	if err != nil {
		return fmt.Errorf("failed to get region: %w", err)
	}

	// Save configuration
	filePath, err := dynamoConfig.SaveConfig(tableConfig, tableName, region)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("\n[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  - Review the configuration: cat %s\n", filePath)
	fmt.Printf("  - Preview deployment: genesys execute %s\n", filePath)
	fmt.Printf("  - Deploy the table: genesys execute %s --apply\n", filePath)
	fmt.Printf("  - Delete when done: genesys execute %s --delete\n", filePath)

	return nil
}

func interactFunction(provider string) error {
	fmt.Printf("\nCreating Lambda Function Configuration for %s\n", provider)
	fmt.Println("==========================================")
	fmt.Println()

	// Note: No credential check needed for interactive mode - only for execution

	// Only AWS Lambda is currently supported
	if provider != "aws" {
		fmt.Printf("Lambda functions are currently only supported for AWS provider\n")
		return nil
	}

	// Use the Lambda interactive configuration
	lambdaConfig, err := config.NewInteractiveLambdaConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize Lambda configuration: %w", err)
	}

	// Generate configuration interactively
	functionConfig, functionName, err := lambdaConfig.CreateLambdaConfig()
	if err != nil {
		return fmt.Errorf("failed to create Lambda configuration: %w", err)
	}

	// Save configuration
	filePath, err := lambdaConfig.SaveConfig(functionConfig, functionName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Build and test locally: genesys lambda build %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s\n", filePath)
	fmt.Printf("  • Deploy the function: genesys execute %s --apply\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute %s --delete\n", filePath)

	return nil
}

func interactNetwork(provider string) error {
	fmt.Printf("Network creation for %s not yet implemented\n", provider)
	return nil
}
