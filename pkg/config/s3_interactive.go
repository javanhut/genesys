package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/yaml.v3"
)

// S3BucketConfig represents a simple S3 bucket configuration
type S3BucketConfig struct {
	Provider string `yaml:"provider"`
	Region   string `yaml:"region"`
	
	Resources struct {
		Storage []struct {
			Name         string            `yaml:"name"`
			Type         string            `yaml:"type"`
			Versioning   bool              `yaml:"versioning"`
			Encryption   bool              `yaml:"encryption"`
			PublicAccess bool              `yaml:"public_access"`
			Tags         map[string]string `yaml:"tags,omitempty"`
			Lifecycle    *struct {
				DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
				ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
			} `yaml:"lifecycle,omitempty"`
		} `yaml:"storage"`
	} `yaml:"resources"`
	
	Policies struct {
		RequireEncryption bool     `yaml:"require_encryption"`
		NoPublicBuckets   bool     `yaml:"no_public_buckets"`
		RequireTags       []string `yaml:"require_tags,omitempty"`
	} `yaml:"policies"`
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
	fmt.Println("🪣 S3 Bucket Configuration Wizard")
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
	config.Resources.Storage = []struct {
		Name         string            `yaml:"name"`
		Type         string            `yaml:"type"`
		Versioning   bool              `yaml:"versioning"`
		Encryption   bool              `yaml:"encryption"`
		PublicAccess bool              `yaml:"public_access"`
		Tags         map[string]string `yaml:"tags,omitempty"`
		Lifecycle    *struct {
			DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
			ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
		} `yaml:"lifecycle,omitempty"`
	}{bucketConfig}

	// Set policies
	config.Policies.RequireEncryption = bucketConfig.Encryption
	config.Policies.NoPublicBuckets = !bucketConfig.PublicAccess
	if len(bucketConfig.Tags) > 0 {
		for tag := range bucketConfig.Tags {
			config.Policies.RequireTags = append(config.Policies.RequireTags, tag)
		}
	}

	// Generate config file name
	fileName := fmt.Sprintf("s3-%s-%d.yaml", bucketName, time.Now().Unix())

	return config, fileName, nil
}

func (isc *InteractiveS3Config) getBucketName() (string, error) {
	var bucketName string
	prompt := &survey.Input{
		Message: "Bucket name:",
		Help:    "S3 bucket names must be globally unique and DNS-compliant",
	}
	
	validator := func(val interface{}) error {
		str := val.(string)
		if len(str) < 3 {
			return fmt.Errorf("bucket name must be at least 3 characters")
		}
		if len(str) > 63 {
			return fmt.Errorf("bucket name must be less than 64 characters")
		}
		if strings.Contains(str, " ") || strings.ContainsAny(str, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			return fmt.Errorf("bucket name must be lowercase with no spaces")
		}
		return nil
	}

	if err := survey.AskOne(prompt, &bucketName, survey.WithValidator(validator)); err != nil {
		return "", err
	}

	return bucketName, nil
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

func (isc *InteractiveS3Config) getBucketConfiguration(bucketName string) (struct {
	Name         string            `yaml:"name"`
	Type         string            `yaml:"type"`
	Versioning   bool              `yaml:"versioning"`
	Encryption   bool              `yaml:"encryption"`
	PublicAccess bool              `yaml:"public_access"`
	Tags         map[string]string `yaml:"tags,omitempty"`
	Lifecycle    *struct {
		DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
		ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
	} `yaml:"lifecycle,omitempty"`
}, error) {
	
	config := struct {
		Name         string            `yaml:"name"`
		Type         string            `yaml:"type"`
		Versioning   bool              `yaml:"versioning"`
		Encryption   bool              `yaml:"encryption"`
		PublicAccess bool              `yaml:"public_access"`
		Tags         map[string]string `yaml:"tags,omitempty"`
		Lifecycle    *struct {
			DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
			ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
		} `yaml:"lifecycle,omitempty"`
	}{
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

func (isc *InteractiveS3Config) getLifecycleConfig() (*struct {
	DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
	ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
}, error) {
	
	lifecycle := &struct {
		DeleteAfterDays  int `yaml:"delete_after_days,omitempty"`
		ArchiveAfterDays int `yaml:"archive_after_days,omitempty"`
	}{}

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
func (isc *InteractiveS3Config) SaveConfig(config *S3BucketConfig, fileName string) (string, error) {
	// Convert to YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Save to file in current directory
	filePath := fmt.Sprintf("./%s", fileName)
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return filePath, nil
}