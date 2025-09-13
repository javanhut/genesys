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

	// Then, select resource type based on provider
	var resourceType string
	var options []string
	
	// Base options available for all providers
	baseOptions := []string{
		"Storage Object",
		"Compute Instance",
		"Database",
		"Function",
		"Network",
	}
	
	// Add AWS-specific options
	if provider == "aws" {
		options = append(baseOptions, []string{
			"IAM Roles & Users",
			"Backup Plans",
			"DynamoDB Tables",
		}...)
	} else {
		options = baseOptions
	}
	
	resourcePrompt := &survey.Select{
		Message: "What type of resource would you like to create?",
		Options: options,
	}
	if err := survey.AskOne(resourcePrompt, &resourceType); err != nil {
		return err
	}

	// Based on provider and resource type, start the specific workflow
	switch resourceType {
	case "Storage Object":
		return interactStorageObject(provider)
	case "Compute Instance":
		return interactCompute(provider)
	case "Database":
		return interactDatabase(provider)
	case "Function":
		return interactFunction(provider)
	case "Network":
		return interactNetwork(provider)
	case "IAM Roles & Users":
		return interactIAM(provider)
	case "Backup Plans":
		return interactBackup(provider)
	case "DynamoDB Tables":
		return interactDynamoDB(provider)
	default:
		fmt.Printf("Resource type '%s' not yet implemented for provider '%s'\n", resourceType, provider)
	}

	return nil
}

// interactStorageObject handles S3 bucket creation workflow
func interactStorageObject(provider string) error {

	var storageObjectName string
	switch provider {
	case "aws":
		storageObjectName = "S3 Bucket"
	case "azure":
		storageObjectName = "Blob Storage"
	case "gcp":
		storageObjectName = "GCS"
	case "tencent":
		storageObjectName = "COS"
	}
	storageTypeTitle := fmt.Sprintf("Creating %s Configuration for %s", storageObjectName, provider)
	fmt.Println(storageTypeTitle)
	fmt.Println("==========================================")
	fmt.Println()
	switch storageObjectName {
	case "S3 Bucket":
		err := s3InteractiveCreation()
		if err != nil {
			return err
		}
	case "Blob Storage":
		err := blobInteractiveCreation()
		if err != nil {
			return err
		}
	case "GCS":
		err := gcsInteractiveCreation()
		if err != nil {
			return err
		}
	case "COS":
		err := cosInteractiveCreation()
		if err != nil {
			return err
		}
	}
	// Note: No credential check needed for interactive mode - only for execution

	return nil
}

func s3InteractiveCreation() error {

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
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the bucket: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)
	return nil
}
func blobInteractiveCreation() error {
	// Use the existing Azure Blob Storage interactive configuration
	blobConfig, err := config.NewInteractiveBlobConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize Blob Storage configuration: %w", err)
	}

	// Generate configuration interactively
	storageConfig, accountName, err := blobConfig.CreateStorageConfig()
	if err != nil {
		return fmt.Errorf("failed to create storage configuration: %w", err)
	}

	// Save configuration
	filePath, err := blobConfig.SaveConfig(storageConfig, accountName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the storage account: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)
	return nil
}
func gcsInteractiveCreation() error {
	// Use the existing GCS interactive configuration
	gcsConfig, err := config.NewInteractiveGCSConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize GCS configuration: %w", err)
	}

	// Generate configuration interactively
	bucketConfig, bucketName, err := gcsConfig.CreateBucketConfig()
	if err != nil {
		return fmt.Errorf("failed to create bucket configuration: %w", err)
	}

	// Save configuration
	filePath, err := gcsConfig.SaveConfig(bucketConfig, bucketName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the bucket: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)
	return nil
}
func cosInteractiveCreation() error {
	// Use the existing COS interactive configuration
	cosConfig, err := config.NewInteractiveCOSConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize COS configuration: %w", err)
	}

	// Generate configuration interactively
	bucketConfig, bucketName, err := cosConfig.CreateBucketConfig()
	if err != nil {
		return fmt.Errorf("failed to create bucket configuration: %w", err)
	}

	// Save configuration
	filePath, err := cosConfig.SaveConfig(bucketConfig, bucketName)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[OK] Configuration saved to: %s\n", filePath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  • Review the configuration: cat %s\n", filePath)
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the bucket: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)
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
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the instance: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)

	return nil
}

func interactDatabase(provider string) error {
	fmt.Printf("Database creation for %s not yet implemented\n", provider)
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
	fmt.Printf("  • Preview deployment: genesys execute %s --dry-run\n", filePath)
	fmt.Printf("  • Deploy the function: genesys execute %s\n", filePath)
	fmt.Printf("  • Delete when done: genesys execute deletion %s\n", filePath)

	return nil
}

func interactNetwork(provider string) error {
	fmt.Printf("Network creation for %s not yet implemented\n", provider)
	return nil
}

// interactIAM handles AWS IAM roles and users creation workflow
func interactIAM(provider string) error {
	fmt.Println("Creating AWS IAM Configuration")
	fmt.Println("==============================")
	fmt.Println()

	// Select IAM resource type
	var iamType string
	iamPrompt := &survey.Select{
		Message: "What type of IAM resource would you like to create?",
		Options: []string{
			"IAM Role",
			"IAM User",
			"IAM Policy",
			"IAM Group",
		},
		Description: func(value string, index int) string {
			switch value {
			case "IAM Role":
				return "Service roles for AWS resources (EC2, Lambda, etc.)"
			case "IAM User":
				return "Individual user accounts with programmatic access"
			case "IAM Policy":
				return "Custom permission policies"
			case "IAM Group":
				return "Groups to organize users and assign permissions"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(iamPrompt, &iamType); err != nil {
		return err
	}

	fmt.Printf("\n[INFO] %s creation workflow not yet implemented.\n", iamType)
	fmt.Println("This feature will be available in a future release.")
	fmt.Println()
	fmt.Println("For now, you can:")
	fmt.Printf("  • Use AWS CLI: aws iam create-role --role-name MyRole\n")
	fmt.Printf("  • Use AWS Console: https://console.aws.amazon.com/iam/\n")
	fmt.Printf("  • Use existing roles in Genesys resource configurations\n")

	return nil
}

// interactBackup handles AWS Backup plans creation workflow
func interactBackup(provider string) error {
	fmt.Println("Creating AWS Backup Plan Configuration")
	fmt.Println("======================================")
	fmt.Println()

	// Select backup plan type
	var backupType string
	backupPrompt := &survey.Select{
		Message: "What type of backup plan would you like to create?",
		Options: []string{
			"Daily Backup Plan",
			"Weekly Backup Plan",
			"Custom Backup Plan",
			"Cross-Region Backup Plan",
		},
		Description: func(value string, index int) string {
			switch value {
			case "Daily Backup Plan":
				return "Automated daily backups with 30-day retention"
			case "Weekly Backup Plan":
				return "Weekly backups with 12-week retention"
			case "Custom Backup Plan":
				return "Custom schedule and retention settings"
			case "Cross-Region Backup Plan":
				return "Backup to multiple AWS regions for disaster recovery"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(backupPrompt, &backupType); err != nil {
		return err
	}

	// Select resources to backup
	var resourceTypes []string
	resourcePrompt := &survey.MultiSelect{
		Message: "What types of resources do you want to backup?",
		Options: []string{
			"EC2 Instances",
			"EBS Volumes",
			"RDS Databases",
			"DynamoDB Tables",
			"EFS File Systems",
			"S3 Buckets",
		},
	}
	if err := survey.AskOne(resourcePrompt, &resourceTypes); err != nil {
		return err
	}

	fmt.Printf("\n[INFO] %s creation workflow not yet implemented.\n", backupType)
	fmt.Println("This feature will be available in a future release.")
	fmt.Println()
	fmt.Printf("Selected resource types: %v\n", resourceTypes)
	fmt.Println()
	fmt.Println("For now, you can:")
	fmt.Printf("  • Use AWS Backup Console: https://console.aws.amazon.com/backup/\n")
	fmt.Printf("  • Use AWS CLI: aws backup create-backup-plan\n")
	fmt.Printf("  • Enable automatic backups in individual services\n")

	return nil
}

// interactDynamoDB handles AWS DynamoDB tables creation workflow
func interactDynamoDB(provider string) error {
	fmt.Println("Creating AWS DynamoDB Table Configuration")
	fmt.Println("=========================================")
	fmt.Println()

	// Select table type
	var tableType string
	tablePrompt := &survey.Select{
		Message: "What type of DynamoDB table would you like to create?",
		Options: []string{
			"Simple Key-Value Table",
			"Table with Sort Key",
			"Global Secondary Index Table",
			"Time Series Data Table",
		},
		Description: func(value string, index int) string {
			switch value {
			case "Simple Key-Value Table":
				return "Basic table with only a partition key"
			case "Table with Sort Key":
				return "Table with partition key and sort key for complex queries"
			case "Global Secondary Index Table":
				return "Table with additional indexes for flexible querying"
			case "Time Series Data Table":
				return "Optimized for time-based data with TTL"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(tablePrompt, &tableType); err != nil {
		return err
	}

	// Select billing mode
	var billingMode string
	billingPrompt := &survey.Select{
		Message: "Select billing mode:",
		Options: []string{
			"On-Demand",
			"Provisioned",
		},
		Description: func(value string, index int) string {
			switch value {
			case "On-Demand":
				return "Pay per request - good for unpredictable traffic"
			case "Provisioned":
				return "Reserve capacity - cost-effective for steady traffic"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(billingPrompt, &billingMode); err != nil {
		return err
	}

	// Select additional features
	var features []string
	featuresPrompt := &survey.MultiSelect{
		Message: "Select additional features:",
		Options: []string{
			"Point-in-Time Recovery",
			"Encryption at Rest",
			"DynamoDB Streams",
			"Global Tables",
			"Auto Scaling",
			"Time to Live (TTL)",
		},
	}
	if err := survey.AskOne(featuresPrompt, &features); err != nil {
		return err
	}

	fmt.Printf("\n[INFO] %s creation workflow not yet implemented.\n", tableType)
	fmt.Println("This feature will be available in a future release.")
	fmt.Println()
	fmt.Printf("Selected billing mode: %s\n", billingMode)
	fmt.Printf("Selected features: %v\n", features)
	fmt.Println()
	fmt.Println("For now, you can:")
	fmt.Printf("  • Use AWS DynamoDB Console: https://console.aws.amazon.com/dynamodb/\n")
	fmt.Printf("  • Use AWS CLI: aws dynamodb create-table\n")
	fmt.Printf("  • Use AWS CloudFormation or Terraform\n")

	return nil
}
