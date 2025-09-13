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

// COSStorageResource represents a COS storage resource configuration
type COSStorageResource struct {
	Name         string            `toml:"name"`
	Type         string            `toml:"type"`
	StorageClass string            `toml:"storage_class"`
	PublicAccess bool              `toml:"public_access"`
	Versioning   bool              `toml:"versioning"`
	Tags         map[string]string `toml:"tags,omitempty"`
	Lifecycle    *COSLifecycleConfig `toml:"lifecycle,omitempty"`
}

// COSLifecycleConfig represents lifecycle configuration
type COSLifecycleConfig struct {
	DeleteAfterDays              int `toml:"delete_after_days,omitempty"`
	TierToStandardIAAfterDays    int `toml:"tier_to_standard_ia_after_days,omitempty"`
	TierToArchiveAfterDays       int `toml:"tier_to_archive_after_days,omitempty"`
	TierToDeepArchiveAfterDays   int `toml:"tier_to_deep_archive_after_days,omitempty"`
}

// COSBucketConfig represents a Tencent COS configuration
type COSBucketConfig struct {
	Provider string `toml:"provider"`
	Region   string `toml:"region"`
	AppId    string `toml:"app_id"`

	Resources struct {
		Storage []COSStorageResource `toml:"storage"`
	} `toml:"resources"`

	Policies struct {
		RequireEncryption bool     `toml:"require_encryption"`
		NoPublicBuckets   bool     `toml:"no_public_buckets"`
		RequireTags       []string `toml:"require_tags,omitempty"`
	} `toml:"policies"`
}

// InteractiveCOSConfig manages interactive COS bucket configuration
type InteractiveCOSConfig struct {
	configDir string
}

// NewInteractiveCOSConfig creates a new interactive COS configuration manager
func NewInteractiveCOSConfig() (*InteractiveCOSConfig, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveCOSConfig{
		configDir: ic.configDir,
	}, nil
}

// CreateBucketConfig creates an interactive COS bucket configuration
func (icc *InteractiveCOSConfig) CreateBucketConfig() (*COSBucketConfig, string, error) {
	fmt.Println("Tencent Cloud Object Storage Configuration Wizard")
	fmt.Println("Let's create a Tencent COS bucket configuration!")
	fmt.Println("")

	config := &COSBucketConfig{
		Provider: "tencent",
	}

	// Get bucket name
	bucketName, err := icc.getBucketName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket name: %w", err)
	}

	// Get App ID
	appId, err := icc.getAppId()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get App ID: %w", err)
	}
	config.AppId = appId

	// Get region
	region, err := icc.getRegion()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get region: %w", err)
	}
	config.Region = region

	// Get bucket configuration
	bucketConfig, err := icc.getBucketConfiguration(bucketName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bucket configuration: %w", err)
	}

	// Add storage resource
	config.Resources.Storage = []COSStorageResource{bucketConfig}

	// Set policies
	config.Policies.RequireEncryption = true // COS supports encryption
	config.Policies.NoPublicBuckets = !bucketConfig.PublicAccess
	if len(bucketConfig.Tags) > 0 {
		for tag := range bucketConfig.Tags {
			config.Policies.RequireTags = append(config.Policies.RequireTags, tag)
		}
	}

	return config, bucketName, nil
}

func (icc *InteractiveCOSConfig) getBucketName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Bucket name:",
		Help:    "Enter any name - it will be automatically formatted for Tencent COS",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("cos", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid bucket name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for Tencent COS: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "Tencent COS requires specific naming conventions",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return icc.getBucketName() // Ask again
		}
	}

	return formattedName, nil
}

func (icc *InteractiveCOSConfig) getAppId() (string, error) {
	var appId string
	prompt := &survey.Input{
		Message: "Tencent Cloud App ID:",
		Help:    "Your Tencent Cloud Application ID (APPID)",
	}

	if err := survey.AskOne(prompt, &appId, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	return appId, nil
}

func (icc *InteractiveCOSConfig) getRegion() (string, error) {
	commonRegions := []string{
		"ap-beijing",
		"ap-nanjing",
		"ap-shanghai",
		"ap-guangzhou",
		"ap-chengdu",
		"ap-chongqing",
		"ap-shenzhen-fsi",
		"ap-shanghai-fsi",
		"ap-beijing-fsi",
		"ap-hongkong",
		"ap-singapore",
		"ap-mumbai",
		"ap-seoul",
		"ap-bangkok",
		"ap-tokyo",
		"na-siliconvalley",
		"na-ashburn",
		"na-toronto",
		"sa-saopaulo",
		"eu-frankfurt",
		"eu-moscow",
	}

	regionDescriptions := map[string]string{
		"ap-beijing":          "Beijing",
		"ap-nanjing":          "Nanjing",
		"ap-shanghai":         "Shanghai",
		"ap-guangzhou":        "Guangzhou",
		"ap-chengdu":          "Chengdu",
		"ap-chongqing":        "Chongqing",
		"ap-shenzhen-fsi":     "Shenzhen Finance",
		"ap-shanghai-fsi":     "Shanghai Finance",
		"ap-beijing-fsi":      "Beijing Finance",
		"ap-hongkong":         "Hong Kong",
		"ap-singapore":        "Singapore",
		"ap-mumbai":           "Mumbai",
		"ap-seoul":            "Seoul",
		"ap-bangkok":          "Bangkok",
		"ap-tokyo":            "Tokyo",
		"na-siliconvalley":    "Silicon Valley",
		"na-ashburn":          "Virginia",
		"na-toronto":          "Toronto",
		"sa-saopaulo":         "São Paulo",
		"eu-frankfurt":        "Frankfurt",
		"eu-moscow":           "Moscow",
	}

	var selectedRegion string
	prompt := &survey.Select{
		Message: "Select Tencent Cloud region:",
		Options: commonRegions,
		Default: "ap-guangzhou",
		Description: func(value string, index int) string {
			return regionDescriptions[value]
		},
	}

	if err := survey.AskOne(prompt, &selectedRegion); err != nil {
		return "", err
	}

	return selectedRegion, nil
}

func (icc *InteractiveCOSConfig) getBucketConfiguration(bucketName string) (COSStorageResource, error) {
	config := COSStorageResource{
		Name: bucketName,
		Type: "bucket",
	}

	// Storage Class
	var storageClass string
	classPrompt := &survey.Select{
		Message: "Select storage class:",
		Options: []string{"STANDARD", "STANDARD_IA", "ARCHIVE", "DEEP_ARCHIVE"},
		Default: "STANDARD",
		Description: func(value string, index int) string {
			switch value {
			case "STANDARD":
				return "Standard storage for frequently accessed data"
			case "STANDARD_IA":
				return "Infrequent Access storage for less frequently accessed data"
			case "ARCHIVE":
				return "Archive storage for long-term backup"
			case "DEEP_ARCHIVE":
				return "Deep Archive storage for long-term archiving"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(classPrompt, &storageClass); err != nil {
		return config, err
	}
	config.StorageClass = storageClass

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
		tags, err := icc.getTags()
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
		Help:    "Automatically tier or delete objects after a certain time",
		Default: false,
	}
	if err := survey.AskOne(lifecyclePrompt, &addLifecycle); err != nil {
		return config, err
	}

	if addLifecycle {
		lifecycle, err := icc.getLifecycleConfig()
		if err != nil {
			return config, err
		}
		config.Lifecycle = lifecycle
	}

	return config, nil
}

func (icc *InteractiveCOSConfig) getTags() (map[string]string, error) {
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

func (icc *InteractiveCOSConfig) getLifecycleConfig() (*COSLifecycleConfig, error) {
	lifecycle := &COSLifecycleConfig{}

	// Tier to Standard IA
	var enableStandardIA bool
	standardIAPrompt := &survey.Confirm{
		Message: "Tier objects to Standard IA storage?",
		Help:    "Move objects to Standard IA after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(standardIAPrompt, &enableStandardIA); err != nil {
		return nil, err
	}

	if enableStandardIA {
		standardIADaysPrompt := &survey.Input{
			Message: "Tier to Standard IA after how many days?",
			Default: "30",
		}
		var standardIADaysStr string
		if err := survey.AskOne(standardIADaysPrompt, &standardIADaysStr); err != nil {
			return nil, err
		}

		var standardIADays int
		if _, err := fmt.Sscanf(standardIADaysStr, "%d", &standardIADays); err != nil {
			standardIADays = 30
		}
		lifecycle.TierToStandardIAAfterDays = standardIADays
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
		lifecycle.TierToArchiveAfterDays = archiveDays
	}

	// Tier to Deep Archive
	var enableDeepArchive bool
	deepArchivePrompt := &survey.Confirm{
		Message: "Tier objects to Deep Archive storage?",
		Help:    "Move objects to Deep Archive after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(deepArchivePrompt, &enableDeepArchive); err != nil {
		return nil, err
	}

	if enableDeepArchive {
		deepArchiveDaysPrompt := &survey.Input{
			Message: "Tier to Deep Archive after how many days?",
			Default: "180",
		}
		var deepArchiveDaysStr string
		if err := survey.AskOne(deepArchiveDaysPrompt, &deepArchiveDaysStr); err != nil {
			return nil, err
		}

		var deepArchiveDays int
		if _, err := fmt.Sscanf(deepArchiveDaysStr, "%d", &deepArchiveDays); err != nil {
			deepArchiveDays = 180
		}
		lifecycle.TierToDeepArchiveAfterDays = deepArchiveDays
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
	if lifecycle.TierToStandardIAAfterDays == 0 && lifecycle.TierToArchiveAfterDays == 0 && 
		lifecycle.TierToDeepArchiveAfterDays == 0 && lifecycle.DeleteAfterDays == 0 {
		return nil, nil
	}

	return lifecycle, nil
}

// SaveConfig saves the COS bucket configuration to a file
func (icc *InteractiveCOSConfig) SaveConfig(config *COSBucketConfig, bucketName string) (string, error) {
	// Create directory structure: resources/cos/
	dirPath := filepath.Join("resources", "cos")
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