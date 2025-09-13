package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/javanhut/genesys/pkg/validation"
)

// S3StorageResource represents a storage resource configuration
type S3StorageResource struct {
	Name         string             `toml:"name"`
	Type         string             `toml:"type"`
	Versioning   bool               `toml:"versioning"`
	Encryption   bool               `toml:"encryption"`
	PublicAccess bool               `toml:"public_access"`
	Tags         map[string]string  `toml:"tags,omitempty"`
	Lifecycle    *S3LifecycleConfig `toml:"lifecycle,omitempty"`
}

// S3LifecycleConfig represents lifecycle configuration
type S3LifecycleConfig struct {
	DeleteAfterDays  int `toml:"delete_after_days,omitempty"`
	ArchiveAfterDays int `toml:"archive_after_days,omitempty"`
}

// S3BucketConfig represents a simple S3 bucket configuration
type S3BucketConfig struct {
	Provider string `toml:"provider"`
	Region   string `toml:"region"`

	Resources struct {
		Storage []S3StorageResource `toml:"storage"`
	} `toml:"resources"`

	Policies struct {
		RequireEncryption bool     `toml:"require_encryption"`
		NoPublicBuckets   bool     `toml:"no_public_buckets"`
		RequireTags       []string `toml:"require_tags,omitempty"`
	} `toml:"policies"`

	IAM *UnifiedIAMConfig `toml:"iam,omitempty"`
}

// InteractiveS3Config manages interactive S3 bucket configuration
type InteractiveS3Config struct {
	configDir string
}

// NewInteractiveS3Config creates a new interactive S3 configuration manager
func NewInteractiveS3Config() (*InteractiveS3Config, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveS3Config{
		configDir: ic.configDir,
	}, nil
}

// CreateBucketConfig creates an interactive S3 bucket configuration
func (isc *InteractiveS3Config) CreateBucketConfig() (*S3BucketConfig, string, error) {
	fmt.Println("S3 Bucket Configuration Wizard")
	fmt.Println("Let's create a simple S3 bucket configuration!")
	fmt.Println("")

	config := &S3BucketConfig{
		Provider: "aws",
	}

	// Get bucket name
	bucketName, err := isc.getBucketName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket name: %w", err)
	}

	// Get region
	region, err := isc.getRegion()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get region: %w", err)
	}
	config.Region = region

	// Get bucket configuration
	bucketConfig, err := isc.getBucketConfiguration(bucketName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket configuration: %w", err)
	}

	// Add storage resource
	config.Resources.Storage = []S3StorageResource{bucketConfig}

	// Set policies
	config.Policies.RequireEncryption = bucketConfig.Encryption
	config.Policies.NoPublicBuckets = !bucketConfig.PublicAccess
	if len(bucketConfig.Tags) > 0 {
		for tag := range bucketConfig.Tags {
			config.Policies.RequireTags = append(config.Policies.RequireTags, tag)
		}
	}

	// Get IAM role configuration
	iamConfig, err := isc.getIAMConfiguration(bucketName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get IAM configuration: %w", err)
	}
	if iamConfig != nil {
		config.IAM = iamConfig
	}

	return config, bucketName, nil
}

func (isc *InteractiveS3Config) getBucketName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Bucket name:",
		Help:    "Enter any name - it will be validated for AWS S3 compliance (globally unique)",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Validate S3 bucket name
	suggestion, err := validation.ValidateAndSuggestS3BucketName(rawName)
	if err != nil {
		fmt.Printf("Error: Invalid bucket name: %v\n", err)
		
		// If we have a suggestion, offer it to the user
		if s3Err, ok := err.(*validation.S3BucketNameError); ok && s3Err.Suggestion != "" {
			var useSuggestion bool
			suggestionPrompt := &survey.Confirm{
				Message: fmt.Sprintf("Use suggested name '%s'?", s3Err.Suggestion),
				Default: true,
				Help:    "This name follows AWS S3 naming rules",
			}
			if surveyErr := survey.AskOne(suggestionPrompt, &useSuggestion); surveyErr != nil {
				return "", surveyErr
			}
			
			if useSuggestion {
				rawName = s3Err.Suggestion
			} else {
				fmt.Println("Please enter a different name...")
				return isc.getBucketName() // Ask again
			}
		} else {
			fmt.Println("Please enter a different name...")
			return isc.getBucketName() // Ask again
		}
	} else {
		rawName = suggestion
	}

	// Check if the name is likely to be available
	if !validation.IsS3BucketNameAvailable(rawName) {
		fmt.Printf("Warning: '%s' is a common name and likely already taken\n", rawName)
		
		// Suggest a unique alternative
		uniqueName := validation.SuggestUniqueBucketName(rawName)
		var useUnique bool
		uniquePrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use unique name '%s' instead?", uniqueName),
			Default: true,
			Help:    "This adds a timestamp to make the name globally unique",
		}
		if err := survey.AskOne(uniquePrompt, &useUnique); err != nil {
			return "", err
		}
		
		if useUnique {
			rawName = uniqueName
		} else {
			fmt.Printf("Warning: Proceeding with '%s' - deployment may fail if name is taken\n", rawName)
		}
	}

	fmt.Printf("✓ Bucket name validated: %s\n", rawName)
	return rawName, nil
}

func (isc *InteractiveS3Config) getRegion() (string, error) {
	commonRegions := []string{
		"us-east-1",      // US East (N. Virginia)
		"us-east-2",      // US East (Ohio)
		"us-west-1",      // US West (N. California)
		"us-west-2",      // US West (Oregon)
		"eu-west-1",      // Europe (Ireland)
		"eu-central-1",   // Europe (Frankfurt)
		"ap-southeast-1", // Asia Pacific (Singapore)
		"ap-northeast-1", // Asia Pacific (Tokyo)
	}

	regionDescriptions := map[string]string{
		"us-east-1":      "US East (N. Virginia) - Default region",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-central-1":   "Europe (Frankfurt)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message: "Select AWS region:",
		Options: commonRegions,
		Default: "us-east-1",
		Description: func(value string, index int) string {
			return regionDescriptions[value]
		},
	}

	if err := survey.AskOne(prompt, &selectedRegion); err != nil {
		return "", err
	}

	return selectedRegion, nil
}

func (isc *InteractiveS3Config) getBucketConfiguration(bucketName string) (S3StorageResource, error) {

	config := S3StorageResource{
		Name: bucketName,
		Type: "bucket",
	}

	// Versioning
	versioningPrompt := &survey.Confirm{
		Message: "Enable versioning?",
		Help:    "Keep multiple versions of objects in the bucket",
		Default: true,
	}
	if err := survey.AskOne(versioningPrompt, &config.Versioning); err != nil {
		return config, err
	}

	// Encryption
	encryptionPrompt := &survey.Confirm{
		Message: "Enable encryption?",
		Help:    "Encrypt objects at rest using AES256",
		Default: true,
	}
	if err := survey.AskOne(encryptionPrompt, &config.Encryption); err != nil {
		return config, err
	}

	// Public access
	publicPrompt := &survey.Confirm{
		Message: "Allow public access?",
		Help:    "WARNING: This makes your bucket publicly accessible",
		Default: false,
	}
	if err := survey.AskOne(publicPrompt, &config.PublicAccess); err != nil {
		return config, err
	}

	// Tags
	var addTags bool
	tagsPrompt := &survey.Confirm{
		Message: "Add tags to the bucket?",
		Help:    "Tags help organize and manage your resources",
		Default: true,
	}
	if err := survey.AskOne(tagsPrompt, &addTags); err != nil {
		return config, err
	}

	if addTags {
		tags, err := isc.getTags()
		if err != nil {
			return config, err
		}
		if len(tags) > 0 {
			config.Tags = tags
		}
	}

	// Lifecycle policies
	var addLifecycle bool
	lifecyclePrompt := &survey.Confirm{
		Message: "Configure lifecycle policies?",
		Help:    "Automatically delete or archive objects after a certain time",
		Default: false,
	}
	if err := survey.AskOne(lifecyclePrompt, &addLifecycle); err != nil {
		return config, err
	}

	if addLifecycle {
		lifecycle, err := isc.getLifecycleConfig()
		if err != nil {
			return config, err
		}
		config.Lifecycle = lifecycle
	}

	return config, nil
}

func (isc *InteractiveS3Config) getTags() (map[string]string, error) {
	tags := make(map[string]string)

	// Add default tags
	defaultTags := map[string]string{
		"Environment": "development",
		"ManagedBy":   "Genesys",
		"Purpose":     "demo",
	}

	fmt.Println("\nDefault tags will be added:")
	for key, value := range defaultTags {
		fmt.Printf("  %s: %s\n", key, value)
		tags[key] = value
	}

	// Ask for additional tags
	var addMore bool
	moreTagsPrompt := &survey.Confirm{
		Message: "Add additional custom tags?",
		Default: false,
	}
	if err := survey.AskOne(moreTagsPrompt, &addMore); err != nil {
		return tags, err
	}

	if addMore {
		for {
			var key, value string

			keyPrompt := &survey.Input{
				Message: "Tag key (empty to stop):",
			}
			if err := survey.AskOne(keyPrompt, &key); err != nil {
				return tags, err
			}

			if key == "" {
				break
			}

			valuePrompt := &survey.Input{
				Message: fmt.Sprintf("Value for '%s':", key),
			}
			if err := survey.AskOne(valuePrompt, &value, survey.WithValidator(survey.Required)); err != nil {
				return tags, err
			}

			tags[key] = value
		}
	}

	return tags, nil
}

func (isc *InteractiveS3Config) getLifecycleConfig() (*S3LifecycleConfig, error) {

	lifecycle := &S3LifecycleConfig{}

	// Archive configuration
	var enableArchive bool
	archivePrompt := &survey.Confirm{
		Message: "Archive objects to cheaper storage?",
		Help:    "Move objects to Glacier after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(archivePrompt, &enableArchive); err != nil {
		return nil, err
	}

	if enableArchive {
		archiveDaysPrompt := &survey.Input{
			Message: "Archive objects after how many days?",
			Default: "90",
		}
		var archiveDaysStr string
		if err := survey.AskOne(archiveDaysPrompt, &archiveDaysStr); err != nil {
			return nil, err
		}

		var archiveDays int
		if _, err := fmt.Sscanf(archiveDaysStr, "%d", &archiveDays); err != nil {
			archiveDays = 90
		}
		lifecycle.ArchiveAfterDays = archiveDays
	}

	// Delete configuration
	var enableDelete bool
	deletePrompt := &survey.Confirm{
		Message: "Automatically delete objects?",
		Help:    "Permanently delete objects after a specified number of days",
		Default: false,
	}
	if err := survey.AskOne(deletePrompt, &enableDelete); err != nil {
		return nil, err
	}

	if enableDelete {
		deleteDaysPrompt := &survey.Input{
			Message: "Delete objects after how many days?",
			Default: "365",
		}
		var deleteDaysStr string
		if err := survey.AskOne(deleteDaysPrompt, &deleteDaysStr); err != nil {
			return nil, err
		}

		var deleteDays int
		if _, err := fmt.Sscanf(deleteDaysStr, "%d", &deleteDays); err != nil {
			deleteDays = 365
		}
		lifecycle.DeleteAfterDays = deleteDays
	}

	// Return nil if no lifecycle policies were configured
	if lifecycle.ArchiveAfterDays == 0 && lifecycle.DeleteAfterDays == 0 {
		return nil, nil
	}

	return lifecycle, nil
}

func (isc *InteractiveS3Config) getIAMConfiguration(bucketName string) (*UnifiedIAMConfig, error) {
	var configureIAM bool
	iamPrompt := &survey.Confirm{
		Message: "Configure IAM role for S3 bucket access?",
		Help:    "Create or use an existing IAM role for applications that need to access this bucket",
		Default: false,
	}
	if err := survey.AskOne(iamPrompt, &configureIAM); err != nil {
		return nil, err
	}

	if !configureIAM {
		return nil, nil
	}

	iamConfig := &UnifiedIAMConfig{}

	// Ask if they want to use existing role or create new one
	roleOptions := []string{
		"create-new (Create a new IAM role automatically)",
		"use-existing (Use an existing IAM role)",
	}

	var selectedOption string
	roleOptionPrompt := &survey.Select{
		Message: "IAM Role Configuration:",
		Options: roleOptions,
		Default: roleOptions[0],
		Help:    "Genesys can create a new role with proper S3 permissions or use an existing role",
	}
	if err := survey.AskOne(roleOptionPrompt, &selectedOption); err != nil {
		return nil, err
	}

	if strings.HasPrefix(selectedOption, "use-existing") {
		// Use existing role
		var existingRoleName string
		existingRolePrompt := &survey.Input{
			Message: "Existing IAM role name or ARN:",
			Help:    "Enter the name or full ARN of an existing IAM role",
		}
		if err := survey.AskOne(existingRolePrompt, &existingRoleName, survey.WithValidator(survey.Required)); err != nil {
			return nil, err
		}

		// Check if it's an ARN or just a name
		if strings.HasPrefix(existingRoleName, "arn:aws:iam::") {
			iamConfig.RoleArn = existingRoleName
			// Extract role name from ARN
			parts := strings.Split(existingRoleName, "/")
			if len(parts) > 0 {
				iamConfig.RoleName = parts[len(parts)-1]
			}
		} else {
			iamConfig.RoleName = existingRoleName
		}

		iamConfig.AutoManage = false
		iamConfig.AutoCleanup = false
		iamConfig.ManagedBy = "external"
	} else {
		// Create new role
		iamConfig.AutoManage = true
		iamConfig.AutoCleanup = true
		
		// Generate default role name
		defaultRoleName := FormatRoleName("s3", bucketName)
		
		var customName bool
		customNamePrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use default role name '%s'?", defaultRoleName),
			Default: true,
		}
		if err := survey.AskOne(customNamePrompt, &customName); err != nil {
			return nil, err
		}

		if customName {
			iamConfig.RoleName = defaultRoleName
		} else {
			var customRoleName string
			customRolePrompt := &survey.Input{
				Message: "Custom IAM role name:",
				Help:    "Enter a custom name for the new IAM role",
			}
			if err := survey.AskOne(customRolePrompt, &customRoleName, survey.WithValidator(survey.Required)); err != nil {
				return nil, err
			}
			iamConfig.RoleName = customRoleName
		}
	}

	// Configure permissions
	fmt.Println("\nIAM Role Permissions:")
	
	defaultPolicies := getDefaultPoliciesForResource("s3")
	fmt.Println("Default policies for S3 access:")
	for i, policy := range defaultPolicies {
		fmt.Printf("  %d. %s\n", i+1, policy)
	}

	var useDefaults bool
	defaultsPrompt := &survey.Confirm{
		Message: "Use default S3 permissions?",
		Help:    "These permissions allow full S3 access and CloudWatch logging",
		Default: true,
	}
	if err := survey.AskOne(defaultsPrompt, &useDefaults); err != nil {
		return nil, err
	}

	if useDefaults {
		iamConfig.RequiredPolicies = defaultPolicies
	} else {
		// Let them choose custom policies
		availablePolicies := []string{
			"S3 full access",
			"S3 read-only access",
			"CloudWatch full access",
			"CloudWatch logs access",
			"DynamoDB read/write access",
			"Lambda full access",
			"Systems Manager Parameter access",
		}

		var selectedPolicies []string
		policyPrompt := &survey.MultiSelect{
			Message: "Select required policies:",
			Options: availablePolicies,
			Default: []string{"S3 full access", "CloudWatch full access"},
			Help:    "Choose the AWS managed policies this role should have",
		}
		if err := survey.AskOne(policyPrompt, &selectedPolicies); err != nil {
			return nil, err
		}
		iamConfig.RequiredPolicies = selectedPolicies
	}

	// Set trust policy
	iamConfig.TrustPolicy = "s3"

	return iamConfig, nil
}

// SaveConfig saves the S3 bucket configuration to a file
func (isc *InteractiveS3Config) SaveConfig(config *S3BucketConfig, bucketName string) (string, error) {
	// Create directory structure: resources/s3/
	dirPath := filepath.Join("resources", "s3")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate filename based on bucket name
	fileName := fmt.Sprintf("%s.toml", bucketName)
	filePath := filepath.Join(dirPath, fileName)

	// Convert to TOML
	buf := new(bytes.Buffer)
	encoder := toml.NewEncoder(buf)
	encoder.Indent = ""
	if err := encoder.Encode(config); err != nil {
		return "", fmt.Errorf("failed to marshal config to TOML: %w", err)
	}

	// Save to file
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return filePath, nil
}
