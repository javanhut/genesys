package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/javanhut/genesys/pkg/validation"
)

// S3StorageResource represents a storage resource configuration
type S3StorageResource struct {
	Name         string             `yaml:"name" toml:"name"`
	Type         string             `yaml:"type" toml:"type"`
	Versioning   bool               `yaml:"versioning" toml:"versioning"`
	Encryption   bool               `yaml:"encryption" toml:"encryption"`
	PublicAccess bool               `yaml:"public_access" toml:"public_access"`
	Tags         map[string]string  `yaml:"tags,omitempty" toml:"tags,omitempty"`
	Lifecycle    *S3LifecycleConfig `yaml:"lifecycle,omitempty" toml:"lifecycle,omitempty"`
}

// S3LifecycleConfig represents lifecycle configuration
type S3LifecycleConfig struct {
	DeleteAfterDays  int `yaml:"delete_after_days,omitempty" toml:"delete_after_days,omitempty"`
	ArchiveAfterDays int `yaml:"archive_after_days,omitempty" toml:"archive_after_days,omitempty"`
}

// S3BucketConfig represents a simple S3 bucket configuration
type S3BucketConfig struct {
	Provider string `yaml:"provider" toml:"provider"`
	Region   string `yaml:"region" toml:"region"`

	Resources struct {
		Storage []S3StorageResource `yaml:"storage" toml:"storage"`
	} `yaml:"resources" toml:"resources"`

	Policies struct {
		RequireEncryption bool     `yaml:"require_encryption" toml:"require_encryption"`
		NoPublicBuckets   bool     `yaml:"no_public_buckets" toml:"no_public_buckets"`
		RequireTags       []string `yaml:"require_tags,omitempty" toml:"require_tags,omitempty"`
	} `yaml:"policies" toml:"policies"`
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

	return config, bucketName, nil
}

func (isc *InteractiveS3Config) getBucketName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Bucket name:",
		Help:    "Enter any name - it will be automatically formatted for AWS S3 (globally unique)",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("s3", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid bucket name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for AWS S3: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "AWS S3 requires globally unique, DNS-compliant bucket names",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return isc.getBucketName() // Ask again
		}
	}

	return formattedName, nil
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
