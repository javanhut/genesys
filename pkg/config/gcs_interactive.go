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

// GCSStorageResource represents a GCS storage resource configuration
type GCSStorageResource struct {
	Name              string            `toml:"name"`
	Type              string            `toml:"type"`
	StorageClass      string            `toml:"storage_class"`
	PublicAccess      bool              `toml:"public_access"`
	Versioning        bool              `toml:"versioning"`
	UniformBucketLevel bool              `toml:"uniform_bucket_level"`
	Labels            map[string]string `toml:"labels,omitempty"`
	Lifecycle         *GCSLifecycleConfig `toml:"lifecycle,omitempty"`
}

// GCSLifecycleConfig represents lifecycle configuration
type GCSLifecycleConfig struct {
	DeleteAfterDays           int `toml:"delete_after_days,omitempty"`
	TierToNearlideAfterDays   int `toml:"tier_to_nearline_after_days,omitempty"`
	TierToColdlineAfterDays   int `toml:"tier_to_coldline_after_days,omitempty"`
	TierToArchiveAfterDays    int `toml:"tier_to_archive_after_days,omitempty"`
}

// GCSBucketConfig represents a Google Cloud Storage configuration
type GCSBucketConfig struct {
	Provider string `toml:"provider"`
	Region   string `toml:"region"`
	Project  string `toml:"project"`

	Resources struct {
		Storage []GCSStorageResource `toml:"storage"`
	} `toml:"resources"`

	Policies struct {
		RequireEncryption      bool     `toml:"require_encryption"`
		NoPublicBuckets        bool     `toml:"no_public_buckets"`
		UniformBucketLevelAccess bool   `toml:"uniform_bucket_level_access"`
		RequireLabels          []string `toml:"require_labels,omitempty"`
	} `toml:"policies"`
}

// InteractiveGCSConfig manages interactive GCS bucket configuration
type InteractiveGCSConfig struct {
	configDir string
}

// NewInteractiveGCSConfig creates a new interactive GCS configuration manager
func NewInteractiveGCSConfig() (*InteractiveGCSConfig, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveGCSConfig{
		configDir: ic.configDir,
	}, nil
}

// CreateBucketConfig creates an interactive GCS bucket configuration
func (igc *InteractiveGCSConfig) CreateBucketConfig() (*GCSBucketConfig, string, error) {
	fmt.Println("Google Cloud Storage Bucket Configuration Wizard")
	fmt.Println("Let's create a Google Cloud Storage bucket configuration!")
	fmt.Println("")

	config := &GCSBucketConfig{
		Provider: "gcp",
	}

	// Get bucket name
	bucketName, err := igc.getBucketName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket name: %w", err)
	}

	// Get project
	project, err := igc.getProject()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get project: %w", err)
	}
	config.Project = project

	// Get region
	region, err := igc.getRegion()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get region: %w", err)
	}
	config.Region = region

	// Get bucket configuration
	bucketConfig, err := igc.getBucketConfiguration(bucketName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket configuration: %w", err)
	}

	// Add storage resource
	config.Resources.Storage = []GCSStorageResource{bucketConfig}

	// Set policies
	config.Policies.RequireEncryption = true // Always enabled in GCS
	config.Policies.NoPublicBuckets = !bucketConfig.PublicAccess
	config.Policies.UniformBucketLevelAccess = bucketConfig.UniformBucketLevel
	if len(bucketConfig.Labels) > 0 {
		for label := range bucketConfig.Labels {
			config.Policies.RequireLabels = append(config.Policies.RequireLabels, label)
		}
	}

	return config, bucketName, nil
}

func (igc *InteractiveGCSConfig) getBucketName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Bucket name:",
		Help:    "Enter any name - it will be automatically formatted for GCS (globally unique)",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("gcs", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid bucket name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for GCS: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "GCS requires globally unique, DNS-compliant bucket names",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return igc.getBucketName() // Ask again
		}
	}

	return formattedName, nil
}

func (igc *InteractiveGCSConfig) getProject() (string, error) {
	var project string
	prompt := &survey.Input{
		Message: "GCP Project ID:",
		Help:    "Your Google Cloud Project ID",
	}

	if err := survey.AskOne(prompt, &project, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return project, nil
}

func (igc *InteractiveGCSConfig) getRegion() (string, error) {
	commonRegions := []string{
		"us-central1",
		"us-east1",
		"us-east4",
		"us-west1",
		"us-west2",
		"us-west3",
		"us-west4",
		"europe-north1",
		"europe-west1",
		"europe-west2",
		"europe-west3",
		"europe-west4",
		"europe-west6",
		"asia-east1",
		"asia-east2",
		"asia-northeast1",
		"asia-northeast2",
		"asia-northeast3",
		"asia-south1",
		"asia-southeast1",
		"asia-southeast2",
		"australia-southeast1",
		"southamerica-east1",
	}

	regionDescriptions := map[string]string{
		"us-central1":           "Iowa",
		"us-east1":              "South Carolina",
		"us-east4":              "Northern Virginia",
		"us-west1":              "Oregon",
		"us-west2":              "Los Angeles",
		"us-west3":              "Salt Lake City",
		"us-west4":              "Las Vegas",
		"europe-north1":         "Finland",
		"europe-west1":          "Belgium",
		"europe-west2":          "London",
		"europe-west3":          "Frankfurt",
		"europe-west4":          "Netherlands",
		"europe-west6":          "Zurich",
		"asia-east1":            "Taiwan",
		"asia-east2":            "Hong Kong",
		"asia-northeast1":       "Tokyo",
		"asia-northeast2":       "Osaka",
		"asia-northeast3":       "Seoul",
		"asia-south1":           "Mumbai",
		"asia-southeast1":       "Singapore",
		"asia-southeast2":       "Jakarta",
		"australia-southeast1":  "Sydney",
		"southamerica-east1":    "São Paulo",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message: "Select GCP region:",
		Options: commonRegions,
		Default: "us-central1",
		Description: func(value string, index int) string {
			return regionDescriptions[value]
		},
	}

	if err := survey.AskOne(prompt, &selectedRegion); err != nil {
		return "", err
	}

	return selectedRegion, nil
}

func (igc *InteractiveGCSConfig) getBucketConfiguration(bucketName string) (GCSStorageResource, error) {
	config := GCSStorageResource{
		Name: bucketName,
		Type: "bucket",
	}

	// Storage Class
	var storageClass string
	classPrompt := &survey.Select{
		Message: "Select storage class:",
		Options: []string{"STANDARD", "NEARLINE", "COLDLINE", "ARCHIVE"},
		Default: "STANDARD",
		Description: func(value string, index int) string {
			switch value {
			case "STANDARD":
				return "Best for frequently accessed data"
			case "NEARLINE":
				return "Best for data accessed once per month or less"
			case "COLDLINE":
				return "Best for data accessed once per quarter or less"
			case "ARCHIVE":
				return "Best for data accessed once per year or less"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(classPrompt, &storageClass); err != nil {
		return config, err
	}
	config.StorageClass = storageClass

	// Uniform bucket-level access
	uniformPrompt := &survey.Confirm{
		Message: "Enable uniform bucket-level access?",
		Help:    "Disables object-level access control lists (ACLs)",
		Default: true,
	}
	if err := survey.AskOne(uniformPrompt, &config.UniformBucketLevel); err != nil {
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

	// Versioning
	versioningPrompt := &survey.Confirm{
		Message: "Enable object versioning?",
		Help:    "Keep multiple versions of objects in the bucket",
		Default: true,
	}
	if err := survey.AskOne(versioningPrompt, &config.Versioning); err != nil {
		return config, err
	}

	// Labels
	var addLabels bool
	labelsPrompt := &survey.Confirm{
		Message: "Add labels to the bucket?",
		Help:    "Labels help organize and manage your resources",
		Default: true,
	}
	if err := survey.AskOne(labelsPrompt, &addLabels); err != nil {
		return config, err
	}

	if addLabels {
		labels, err := igc.getLabels()
		if err != nil {
			return config, err
		}
		if len(labels) > 0 {
			config.Labels = labels
		}
	}

	// Lifecycle policies
	var addLifecycle bool
	lifecyclePrompt := &survey.Confirm{
		Message: "Configure lifecycle policies?",
		Help:    "Automatically tier or delete objects after a certain time",
		Default: false,
	}
	if err := survey.AskOne(lifecyclePrompt, &addLifecycle); err != nil {
		return config, err
	}

	if addLifecycle {
		lifecycle, err := igc.getLifecycleConfig()
		if err != nil {
			return config, err
		}
		config.Lifecycle = lifecycle
	}

	return config, nil
}

func (igc *InteractiveGCSConfig) getLabels() (map[string]string, error) {
	labels := make(map[string]string)

	// Add default labels
	defaultLabels := map[string]string{
		"environment": "development",
		"managed-by":  "genesys",
		"purpose":     "demo",
	}

	fmt.Println("\nDefault labels will be added:")
	for key, value := range defaultLabels {
		fmt.Printf("  %s: %s\n", key, value)
		labels[key] = value
	}

	// Ask for additional labels
	var addMore bool
	moreLabelsPrompt := &survey.Confirm{
		Message: "Add additional custom labels?",
		Default: false,
	}
	if err := survey.AskOne(moreLabelsPrompt, &addMore); err != nil {
		return labels, err
	}

	if addMore {
		for {
			var key, value string

			keyPrompt := &survey.Input{
				Message: "Label key (empty to stop):",
				Help:    "Must be lowercase with hyphens, no underscores",
			}
			if err := survey.AskOne(keyPrompt, &key); err != nil {
				return labels, err
			}

			if key == "" {
				break
			}

			valuePrompt := &survey.Input{
				Message: fmt.Sprintf("Value for '%s':", key),
			}
			if err := survey.AskOne(valuePrompt, &value, survey.WithValidator(survey.Required)); err != nil {
				return labels, err
			}

			labels[key] = value
		}
	}

	return labels, nil
}

func (igc *InteractiveGCSConfig) getLifecycleConfig() (*GCSLifecycleConfig, error) {
	lifecycle := &GCSLifecycleConfig{}

	// Tier to Nearline
	var enableNearline bool
	nearlinePrompt := &survey.Confirm{
		Message: "Tier objects to Nearline storage?",
		Help:    "Move objects to Nearline after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(nearlinePrompt, &enableNearline); err != nil {
		return nil, err
	}

	if enableNearline {
		nearlineDaysPrompt := &survey.Input{
			Message: "Tier to Nearline after how many days?",
			Default: "30",
		}
		var nearlineDaysStr string
		if err := survey.AskOne(nearlineDaysPrompt, &nearlineDaysStr); err != nil {
			return nil, err
		}

		var nearlineDays int
		if _, err := fmt.Sscanf(nearlineDaysStr, "%d", &nearlineDays); err != nil {
			nearlineDays = 30
		}
		lifecycle.TierToNearlideAfterDays = nearlineDays
	}

	// Tier to Coldline
	var enableColdline bool
	coldlinePrompt := &survey.Confirm{
		Message: "Tier objects to Coldline storage?",
		Help:    "Move objects to Coldline after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(coldlinePrompt, &enableColdline); err != nil {
		return nil, err
	}

	if enableColdline {
		coldlineDaysPrompt := &survey.Input{
			Message: "Tier to Coldline after how many days?",
			Default: "90",
		}
		var coldlineDaysStr string
		if err := survey.AskOne(coldlineDaysPrompt, &coldlineDaysStr); err != nil {
			return nil, err
		}

		var coldlineDays int
		if _, err := fmt.Sscanf(coldlineDaysStr, "%d", &coldlineDays); err != nil {
			coldlineDays = 90
		}
		lifecycle.TierToColdlineAfterDays = coldlineDays
	}

	// Tier to Archive
	var enableArchive bool
	archivePrompt := &survey.Confirm{
		Message: "Tier objects to Archive storage?",
		Help:    "Move objects to Archive after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(archivePrompt, &enableArchive); err != nil {
		return nil, err
	}

	if enableArchive {
		archiveDaysPrompt := &survey.Input{
			Message: "Tier to Archive after how many days?",
			Default: "365",
		}
		var archiveDaysStr string
		if err := survey.AskOne(archiveDaysPrompt, &archiveDaysStr); err != nil {
			return nil, err
		}

		var archiveDays int
		if _, err := fmt.Sscanf(archiveDaysStr, "%d", &archiveDays); err != nil {
			archiveDays = 365
		}
		lifecycle.TierToArchiveAfterDays = archiveDays
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
			Default: "2555", // 7 years
		}
		var deleteDaysStr string
		if err := survey.AskOne(deleteDaysPrompt, &deleteDaysStr); err != nil {
			return nil, err
		}

		var deleteDays int
		if _, err := fmt.Sscanf(deleteDaysStr, "%d", &deleteDays); err != nil {
			deleteDays = 2555
		}
		lifecycle.DeleteAfterDays = deleteDays
	}

	// Return nil if no lifecycle policies were configured
	if lifecycle.TierToNearlideAfterDays == 0 && lifecycle.TierToColdlineAfterDays == 0 && 
		lifecycle.TierToArchiveAfterDays == 0 && lifecycle.DeleteAfterDays == 0 {
		return nil, nil
	}

	return lifecycle, nil
}

// SaveConfig saves the GCS bucket configuration to a file
func (igc *InteractiveGCSConfig) SaveConfig(config *GCSBucketConfig, bucketName string) (string, error) {
	// Create directory structure: resources/gcs/
	dirPath := filepath.Join("resources", "gcs")
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