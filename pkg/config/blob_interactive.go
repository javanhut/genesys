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

// BlobStorageResource represents a blob storage resource configuration
type BlobStorageResource struct {
	Name           string            `toml:"name"`
	Type           string            `toml:"type"`
	Performance    string            `toml:"performance"`     // Standard or Premium
	Redundancy     string            `toml:"redundancy"`      // LRS, ZRS, GRS, RA-GRS, GZRS, RA-GZRS
	AccessTier     string            `toml:"access_tier"`     // Hot, Cool, Cold
	PublicAccess   string            `toml:"public_access"`
	Versioning     bool              `toml:"versioning"`
	StorageType    string            `toml:"storage_type,omitempty"`    // Preferred storage type
	ExtendedZone   bool              `toml:"extended_zone,omitempty"`   // Deploy to Azure Extended Zone
	Tags           map[string]string `toml:"tags,omitempty"`
	Lifecycle      *BlobLifecycleConfig `toml:"lifecycle,omitempty"`
	
	// Security settings
	Security       *BlobSecurityConfig `toml:"security,omitempty"`
	
	// Networking settings
	Network        *BlobNetworkConfig  `toml:"network,omitempty"`
	
	// Recovery settings
	Recovery       *BlobRecoveryConfig `toml:"recovery,omitempty"`
	
	// Advanced features
	Advanced       *BlobAdvancedConfig `toml:"advanced,omitempty"`
}

// BlobLifecycleConfig represents lifecycle configuration
type BlobLifecycleConfig struct {
	DeleteAfterDays      int `toml:"delete_after_days,omitempty"`
	TierToCoolAfterDays  int `toml:"tier_to_cool_after_days,omitempty"`
	TierToArchiveAfterDays int `toml:"tier_to_archive_after_days,omitempty"`
}

// BlobSecurityConfig represents security settings
type BlobSecurityConfig struct {
	RequireSecureTransfer     bool   `toml:"require_secure_transfer"`
	AllowAnonymousAccess      bool   `toml:"allow_anonymous_access"`
	EnableKeyAccess           bool   `toml:"enable_key_access"`
	DefaultToEntraAuth        bool   `toml:"default_to_entra_auth"`
	MinimumTLSVersion         string `toml:"minimum_tls_version"`
	EncryptionType            string `toml:"encryption_type"`
	CustomerManagedKeys       bool   `toml:"customer_managed_keys"`
	InfrastructureEncryption  bool   `toml:"infrastructure_encryption"`
	ImmutabilitySupport       bool   `toml:"immutability_support"`
}

// BlobNetworkConfig represents networking settings  
type BlobNetworkConfig struct {
	PublicNetworkAccess       string `toml:"public_network_access"`
	PublicNetworkAccessScope  string `toml:"public_network_access_scope,omitempty"`
	RoutingPreference         string `toml:"routing_preference"`
	PrivateEndpoint           bool   `toml:"private_endpoint"`
}

// BlobRecoveryConfig represents recovery and backup settings
type BlobRecoveryConfig struct {
	PointInTimeRestore        bool `toml:"point_in_time_restore"`
	BlobSoftDelete            bool `toml:"blob_soft_delete"`
	BlobSoftDeleteDays        int  `toml:"blob_soft_delete_days"`
	ContainerSoftDelete       bool `toml:"container_soft_delete"`
	ContainerSoftDeleteDays   int  `toml:"container_soft_delete_days"`
	FileShareSoftDelete       bool `toml:"file_share_soft_delete"`
	FileShareSoftDeleteDays   int  `toml:"file_share_soft_delete_days"`
	VersioningEnabled         bool `toml:"versioning_enabled"`
	ChangeFeedException       bool `toml:"change_feed_enabled"`
}

// BlobAdvancedConfig represents advanced features
type BlobAdvancedConfig struct {
	HierarchicalNamespace    bool `toml:"hierarchical_namespace"`
	EnableSFTP               bool `toml:"enable_sftp"`
	EnableNFSv3              bool `toml:"enable_nfsv3"`
	CrossTenantReplication   bool `toml:"cross_tenant_replication"`
	LargeFileShares          bool `toml:"large_file_shares"`
}

// BlobStorageConfig represents an Azure Blob Storage configuration
type BlobStorageConfig struct {
	Provider      string `toml:"provider"`
	Subscription  string `toml:"subscription,omitempty"`
	Location      string `toml:"location"`
	ResourceGroup string `toml:"resource_group"`

	Resources struct {
		Storage []BlobStorageResource `toml:"storage"`
	} `toml:"resources"`

	Policies struct {
		RequireEncryption bool     `toml:"require_encryption"`
		NoPublicAccess    bool     `toml:"no_public_access"`
		RequireTags       []string `toml:"require_tags,omitempty"`
	} `toml:"policies"`
}

// InteractiveBlobConfig manages interactive Azure Blob Storage configuration
type InteractiveBlobConfig struct {
	configDir string
}

// NewInteractiveBlobConfig creates a new interactive Blob Storage configuration manager
func NewInteractiveBlobConfig() (*InteractiveBlobConfig, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveBlobConfig{
		configDir: ic.configDir,
	}, nil
}

// CreateStorageConfig creates an interactive Azure Blob Storage configuration
func (ibc *InteractiveBlobConfig) CreateStorageConfig() (*BlobStorageConfig, string, error) {
	fmt.Println("Azure Blob Storage Configuration Wizard")
	fmt.Println("Let's create an Azure Blob Storage configuration!")
	fmt.Println("")

	config := &BlobStorageConfig{
		Provider: "azure",
	}

	// Get subscription (optional - for organization purposes)
	subscription, err := ibc.getSubscription()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get subscription: %w", err)
	}
	if subscription != "" {
		config.Subscription = subscription
	}

	// Get storage account name
	accountName, err := ibc.getStorageAccountName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get storage account name: %w", err)
	}

	// Get location
	location, err := ibc.getLocation()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get location: %w", err)
	}
	config.Location = location

	// Get resource group
	resourceGroup, err := ibc.getResourceGroup()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get resource group: %w", err)
	}
	config.ResourceGroup = resourceGroup

	// Get storage configuration
	storageConfig, err := ibc.getStorageConfiguration(accountName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get storage configuration: %w", err)
	}

	// Add storage resource
	config.Resources.Storage = []BlobStorageResource{storageConfig}

	// Set policies
	config.Policies.RequireEncryption = true // Always enabled in Azure
	config.Policies.NoPublicAccess = storageConfig.PublicAccess == "none"
	if len(storageConfig.Tags) > 0 {
		for tag := range storageConfig.Tags {
			config.Policies.RequireTags = append(config.Policies.RequireTags, tag)
		}
	}

	return config, accountName, nil
}

func (ibc *InteractiveBlobConfig) getSubscription() (string, error) {
	var addSubscription bool
	subscriptionPrompt := &survey.Confirm{
		Message: "Specify Azure subscription ID?",
		Default: false,
		Help:    "Optional: Add subscription ID for organization and billing tracking",
	}
	if err := survey.AskOne(subscriptionPrompt, &addSubscription); err != nil {
		return "", err
	}

	if !addSubscription {
		return "", nil
	}

	var subscription string
	prompt := &survey.Input{
		Message: "Azure subscription ID or name:",
		Help:    "Enter your Azure subscription ID (GUID) or subscription name",
	}
	if err := survey.AskOne(prompt, &subscription); err != nil {
		return "", err
	}

	return subscription, nil
}

func (ibc *InteractiveBlobConfig) getStorageAccountName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Storage account name:",
		Help:    "Enter any name - it will be automatically formatted for Azure (globally unique)",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("blob", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid storage account name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for Azure Blob Storage: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "Azure requires globally unique storage account names",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return ibc.getStorageAccountName() // Ask again
		}
	}

	return formattedName, nil
}

func (ibc *InteractiveBlobConfig) getLocation() (string, error) {
	commonLocations := []string{
		"eastus",
		"eastus2",
		"westus",
		"westus2",
		"westus3",
		"centralus",
		"northcentralus",
		"southcentralus",
		"westeurope",
		"northeurope",
		"ukwest",
		"uksouth",
		"francecentral",
		"germanywestcentral",
		"norwayeast",
		"switzerlandnorth",
		"swedencentral",
		"eastasia",
		"southeastasia",
		"japaneast",
		"japanwest",
		"koreacentral",
		"koreasouth",
		"southindia",
		"westindia",
		"centralindia",
		"australiaeast",
		"australiasoutheast",
		"brazilsouth",
		"canadacentral",
		"canadaeast",
	}

	var selectedLocation string
	prompt := &survey.Select{
		Message: "Select Azure location:",
		Options: commonLocations,
		Default: "eastus",
	}

	if err := survey.AskOne(prompt, &selectedLocation); err != nil {
		return "", err
	}

	return selectedLocation, nil
}

func (ibc *InteractiveBlobConfig) getResourceGroup() (string, error) {
	// First ask if they want to create new or use existing
	var resourceGroupOption string
	optionPrompt := &survey.Select{
		Message: "Resource group:",
		Options: []string{"Create new", "Use existing"},
		Default: "Create new",
		Help:    "Choose a new or existing resource group to organize and manage your storage account together with other resources",
	}
	if err := survey.AskOne(optionPrompt, &resourceGroupOption); err != nil {
		return "", err
	}

	var resourceGroup string
	if resourceGroupOption == "Create new" {
		prompt := &survey.Input{
			Message: "New resource group name:",
			Default: "genesys-rg",
			Help:    "Resource groups help organize and manage Azure resources together",
		}
		if err := survey.AskOne(prompt, &resourceGroup, survey.WithValidator(survey.Required)); err != nil {
			return "", err
		}
	} else {
		// For existing resource groups, we'd normally query Azure, but for now just ask for name
		prompt := &survey.Input{
			Message: "Existing resource group name:",
			Help:    "Enter the name of an existing resource group",
		}
		if err := survey.AskOne(prompt, &resourceGroup, survey.WithValidator(survey.Required)); err != nil {
			return "", err
		}
	}

	return resourceGroup, nil
}

func (ibc *InteractiveBlobConfig) getStorageConfiguration(accountName string) (BlobStorageResource, error) {
	config := BlobStorageResource{
		Name: accountName,
		Type: "storage_account",
	}

	// Performance tier (Standard/Premium)
	var performance string
	performancePrompt := &survey.Select{
		Message: "Select performance tier:",
		Options: []string{"Standard", "Premium"},
		Default: "Standard",
		Description: func(value string, index int) string {
			switch value {
			case "Standard":
				return "Recommended for most scenarios (general-purpose v2 account)"
			case "Premium":
				return "Recommended for scenarios that require low latency"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(performancePrompt, &performance); err != nil {
		return config, err
	}
	config.Performance = performance

	// Redundancy options
	var redundancy string
	var redundancyOptions []string
	var redundancyDescriptions = map[string]string{
		"LRS":     "Locally redundant storage - 3 copies in single zone",
		"ZRS":     "Zone-redundant storage - 3 copies across zones",
		"GRS":     "Geo-redundant storage - 6 copies across regions", 
		"RA-GRS":  "Read-access geo-redundant - 6 copies + read access",
		"GZRS":    "Geo-zone-redundant - highest durability",
		"RA-GZRS": "Read-access geo-zone-redundant - highest availability",
	}

	if performance == "Standard" {
		redundancyOptions = []string{"LRS", "ZRS", "GRS", "RA-GRS", "GZRS", "RA-GZRS"}
	} else {
		redundancyOptions = []string{"LRS", "ZRS"} // Premium only supports LRS and ZRS
	}

	redundancyPrompt := &survey.Select{
		Message: "Select redundancy option:",
		Options: redundancyOptions,
		Default: "LRS",
		Description: func(value string, index int) string {
			return redundancyDescriptions[value]
		},
	}
	if err := survey.AskOne(redundancyPrompt, &redundancy); err != nil {
		return config, err
	}
	config.Redundancy = redundancy

	// Preferred storage type
	var storageType string
	storageTypePrompt := &survey.Select{
		Message: "Preferred storage type:",
		Options: []string{"Blob storage", "Data Lake Storage", "General purpose", "File shares", "Queue storage", "Table storage"},
		Default: "General purpose",
		Help:    "This helps provide relevant guidance. It doesn't restrict your storage to this resource type.",
	}
	if err := survey.AskOne(storageTypePrompt, &storageType); err != nil {
		return config, err
	}
	config.StorageType = storageType

	// Extended Zone deployment (optional advanced feature)
	var useExtendedZone bool
	extendedZonePrompt := &survey.Confirm{
		Message: "Deploy to an Azure Extended Zone?",
		Default: false,
		Help:    "Extended Zones provide low-latency access closer to on-premises infrastructure",
	}
	if err := survey.AskOne(extendedZonePrompt, &useExtendedZone); err != nil {
		return config, err
	}
	config.ExtendedZone = useExtendedZone

	// Access Tier
	var accessTier string
	tierPrompt := &survey.Select{
		Message: "Select access tier:",
		Options: []string{"Hot", "Cool", "Cold"},
		Default: "Hot",
		Description: func(value string, index int) string {
			switch value {
			case "Hot":
				return "Optimized for frequently accessed data and everyday usage scenarios"
			case "Cool":
				return "Optimized for infrequently accessed data and backup scenarios"
			case "Cold":
				return "Optimized for rarely accessed data and backup scenarios"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(tierPrompt, &accessTier); err != nil {
		return config, err
	}
	config.AccessTier = accessTier

	// Public access
	var publicAccess string
	publicPrompt := &survey.Select{
		Message: "Public access level:",
		Options: []string{"none", "blob", "container"},
		Default: "none",
		Help:    "none: No public access. blob: Public read for blobs only. container: Public read for containers and blobs",
	}
	if err := survey.AskOne(publicPrompt, &publicAccess); err != nil {
		return config, err
	}
	config.PublicAccess = publicAccess

	// Versioning
	versioningPrompt := &survey.Confirm{
		Message: "Enable blob versioning?",
		Help:    "Keep multiple versions of blobs",
		Default: true,
	}
	if err := survey.AskOne(versioningPrompt, &config.Versioning); err != nil {
		return config, err
	}

	// Tags
	var addTags bool
	tagsPrompt := &survey.Confirm{
		Message: "Add tags to the storage account?",
		Help:    "Tags help organize and manage your resources",
		Default: true,
	}
	if err := survey.AskOne(tagsPrompt, &addTags); err != nil {
		return config, err
	}

	if addTags {
		tags, err := ibc.getTags()
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
		Help:    "Automatically tier or delete blobs after a certain time",
		Default: false,
	}
	if err := survey.AskOne(lifecyclePrompt, &addLifecycle); err != nil {
		return config, err
	}

	if addLifecycle {
		lifecycle, err := ibc.getLifecycleConfig()
		if err != nil {
			return config, err
		}
		config.Lifecycle = lifecycle
	}

	// Ask if user wants to configure advanced settings
	var configureAdvanced bool
	advancedPrompt := &survey.Confirm{
		Message: "Configure advanced settings (Security, Networking, Recovery)?",
		Default: false,
		Help:    "Configure additional Azure Storage features like security, networking, and data protection",
	}
	if err := survey.AskOne(advancedPrompt, &configureAdvanced); err != nil {
		return config, err
	}

	if configureAdvanced {
		// Security settings
		security, err := ibc.getSecurityConfig()
		if err != nil {
			return config, err
		}
		if security != nil {
			config.Security = security
		}

		// Network settings
		network, err := ibc.getNetworkConfig()
		if err != nil {
			return config, err
		}
		if network != nil {
			config.Network = network
		}

		// Recovery settings
		recovery, err := ibc.getRecoveryConfig()
		if err != nil {
			return config, err
		}
		if recovery != nil {
			config.Recovery = recovery
		}

		// Advanced features
		advanced, err := ibc.getAdvancedConfig()
		if err != nil {
			return config, err
		}
		if advanced != nil {
			config.Advanced = advanced
		}
	}

	return config, nil
}

func (ibc *InteractiveBlobConfig) getTags() (map[string]string, error) {
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

func (ibc *InteractiveBlobConfig) getLifecycleConfig() (*BlobLifecycleConfig, error) {
	lifecycle := &BlobLifecycleConfig{}

	// Tier to Cool
	var enableCool bool
	coolPrompt := &survey.Confirm{
		Message: "Tier blobs to Cool storage?",
		Help:    "Move blobs to Cool tier after a specified number of days",
		Default: true,
	}
	if err := survey.AskOne(coolPrompt, &enableCool); err != nil {
		return nil, err
	}

	if enableCool {
		coolDaysPrompt := &survey.Input{
			Message: "Tier to Cool after how many days?",
			Default: "30",
		}
		var coolDaysStr string
		if err := survey.AskOne(coolDaysPrompt, &coolDaysStr); err != nil {
			return nil, err
		}

		var coolDays int
		if _, err := fmt.Sscanf(coolDaysStr, "%d", &coolDays); err != nil {
			coolDays = 30
		}
		lifecycle.TierToCoolAfterDays = coolDays
	}

	// Tier to Archive
	var enableArchive bool
	archivePrompt := &survey.Confirm{
		Message: "Tier blobs to Archive storage?",
		Help:    "Move blobs to Archive tier after a specified number of days",
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

	// Delete configuration
	var enableDelete bool
	deletePrompt := &survey.Confirm{
		Message: "Automatically delete blobs?",
		Help:    "Permanently delete blobs after a specified number of days",
		Default: false,
	}
	if err := survey.AskOne(deletePrompt, &enableDelete); err != nil {
		return nil, err
	}

	if enableDelete {
		deleteDaysPrompt := &survey.Input{
			Message: "Delete blobs after how many days?",
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
	if lifecycle.TierToCoolAfterDays == 0 && lifecycle.TierToArchiveAfterDays == 0 && lifecycle.DeleteAfterDays == 0 {
		return nil, nil
	}

	return lifecycle, nil
}

// SaveConfig saves the Blob Storage configuration to a file
func (ibc *InteractiveBlobConfig) SaveConfig(config *BlobStorageConfig, accountName string) (string, error) {
	// Create directory structure: resources/blob/
	dirPath := filepath.Join("resources", "blob")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate filename based on account name
	fileName := fmt.Sprintf("%s.toml", accountName)
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

// getSecurityConfig configures security settings
func (ibc *InteractiveBlobConfig) getSecurityConfig() (*BlobSecurityConfig, error) {
	fmt.Println("\nSecurity Configuration")
	fmt.Println("Configure security settings that impact your storage account")
	
	security := &BlobSecurityConfig{}

	// Secure transfer
	secureTransferPrompt := &survey.Confirm{
		Message: "Require secure transfer for REST API operations?",
		Default: true,
		Help:    "Enforces HTTPS for all storage account operations",
	}
	if err := survey.AskOne(secureTransferPrompt, &security.RequireSecureTransfer); err != nil {
		return nil, err
	}

	// Anonymous access
	anonymousPrompt := &survey.Confirm{
		Message: "Allow enabling anonymous access on individual containers?",
		Default: false,
		Help:    "Allows containers to be configured for public blob access",
	}
	if err := survey.AskOne(anonymousPrompt, &security.AllowAnonymousAccess); err != nil {
		return nil, err
	}

	// Storage account key access
	keyAccessPrompt := &survey.Confirm{
		Message: "Enable storage account key access?",
		Default: true,
		Help:    "Allows access using storage account keys",
	}
	if err := survey.AskOne(keyAccessPrompt, &security.EnableKeyAccess); err != nil {
		return nil, err
	}

	// Default to Entra authorization
	entraPrompt := &survey.Confirm{
		Message: "Default to Microsoft Entra authorization in the Azure portal?",
		Default: true,
		Help:    "Use Azure AD for portal access instead of account keys",
	}
	if err := survey.AskOne(entraPrompt, &security.DefaultToEntraAuth); err != nil {
		return nil, err
	}

	// Minimum TLS version
	var tlsVersion string
	tlsPrompt := &survey.Select{
		Message: "Minimum TLS version:",
		Options: []string{"TLS 1.0", "TLS 1.1", "TLS 1.2"},
		Default: "TLS 1.2",
		Help:    "Minimum TLS version required for requests",
	}
	if err := survey.AskOne(tlsPrompt, &tlsVersion); err != nil {
		return nil, err
	}
	security.MinimumTLSVersion = tlsVersion

	return security, nil
}

// getNetworkConfig configures networking settings
func (ibc *InteractiveBlobConfig) getNetworkConfig() (*BlobNetworkConfig, error) {
	fmt.Println("\nNetwork Configuration")
	fmt.Println("Configure network access and routing")
	
	network := &BlobNetworkConfig{}

	// Public network access
	var publicAccess string
	publicAccessPrompt := &survey.Select{
		Message: "Public network access:",
		Options: []string{
			"Allow from all networks",
			"Allow from selected networks", 
			"Disable public access",
		},
		Default: "Allow from all networks",
		Description: func(value string, index int) string {
			switch value {
			case "Allow from all networks":
				return "Allow inbound and outbound access with the option to restrict select inbound access"
			case "Allow from selected networks":
				return "Restrict inbound access while allowing outbound access"
			case "Disable public access":
				return "Restrict inbound and outbound access using a network security perimeter"
			default:
				return ""
			}
		},
	}
	if err := survey.AskOne(publicAccessPrompt, &publicAccess); err != nil {
		return nil, err
	}
	network.PublicNetworkAccess = publicAccess

	// Routing preference
	var routing string
	routingPrompt := &survey.Select{
		Message: "Network routing preference:",
		Options: []string{"Microsoft network", "Internet routing"},
		Default: "Microsoft network",
		Help:    "Microsoft network routing is recommended for most customers",
	}
	if err := survey.AskOne(routingPrompt, &routing); err != nil {
		return nil, err
	}
	network.RoutingPreference = routing

	// Private endpoint
	privateEndpointPrompt := &survey.Confirm{
		Message: "Create a private endpoint?",
		Default: false,
		Help:    "Allow private connection to this resource through a private endpoint",
	}
	if err := survey.AskOne(privateEndpointPrompt, &network.PrivateEndpoint); err != nil {
		return nil, err
	}

	return network, nil
}

// getRecoveryConfig configures data protection and recovery settings
func (ibc *InteractiveBlobConfig) getRecoveryConfig() (*BlobRecoveryConfig, error) {
	fmt.Println("\nData Protection Configuration")
	fmt.Println("Protect your data from accidental or erroneous deletion or modification")
	
	recovery := &BlobRecoveryConfig{}

	// Point-in-time restore
	pitPrompt := &survey.Confirm{
		Message: "Enable point-in-time restore for containers?",
		Default: false,
		Help:    "Restore one or more containers to an earlier state (requires versioning, change feed, and blob soft delete)",
	}
	if err := survey.AskOne(pitPrompt, &recovery.PointInTimeRestore); err != nil {
		return nil, err
	}

	// Blob soft delete
	blobSoftDeletePrompt := &survey.Confirm{
		Message: "Enable soft delete for blobs?",
		Default: true,
		Help:    "Recover blobs that were previously marked for deletion, including blobs that were overwritten",
	}
	if err := survey.AskOne(blobSoftDeletePrompt, &recovery.BlobSoftDelete); err != nil {
		return nil, err
	}

	if recovery.BlobSoftDelete {
		var daysStr string
		daysPrompt := &survey.Input{
			Message: "Days to retain deleted blobs:",
			Default: "7",
		}
		if err := survey.AskOne(daysPrompt, &daysStr); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscanf(daysStr, "%d", &recovery.BlobSoftDeleteDays); err != nil {
			recovery.BlobSoftDeleteDays = 7
		}
	}

	// Container soft delete
	containerSoftDeletePrompt := &survey.Confirm{
		Message: "Enable soft delete for containers?",
		Default: true,
		Help:    "Recover containers that were previously marked for deletion",
	}
	if err := survey.AskOne(containerSoftDeletePrompt, &recovery.ContainerSoftDelete); err != nil {
		return nil, err
	}

	if recovery.ContainerSoftDelete {
		var daysStr string
		daysPrompt := &survey.Input{
			Message: "Days to retain deleted containers:",
			Default: "7",
		}
		if err := survey.AskOne(daysPrompt, &daysStr); err != nil {
			return nil, err
		}
		if _, err := fmt.Sscanf(daysStr, "%d", &recovery.ContainerSoftDeleteDays); err != nil {
			recovery.ContainerSoftDeleteDays = 7
		}
	}

	// Versioning
	versioningPrompt := &survey.Confirm{
		Message: "Enable versioning for blobs?",
		Default: true,
		Help:    "Automatically maintain previous versions of your blobs",
	}
	if err := survey.AskOne(versioningPrompt, &recovery.VersioningEnabled); err != nil {
		return nil, err
	}

	// Change feed
	changeFeedPrompt := &survey.Confirm{
		Message: "Enable blob change feed?",
		Default: false,
		Help:    "Keep track of create, modification, and delete changes to blobs in your account",
	}
	if err := survey.AskOne(changeFeedPrompt, &recovery.ChangeFeedException); err != nil {
		return nil, err
	}

	return recovery, nil
}

// getAdvancedConfig configures advanced features
func (ibc *InteractiveBlobConfig) getAdvancedConfig() (*BlobAdvancedConfig, error) {
	fmt.Println("\nAdvanced Features Configuration")
	fmt.Println("Configure advanced storage capabilities")
	
	advanced := &BlobAdvancedConfig{}

	// Hierarchical namespace
	hierarchicalPrompt := &survey.Confirm{
		Message: "Enable hierarchical namespace (Data Lake Storage Gen2)?",
		Default: false,
		Help:    "Enables file and directory semantics, accelerates big data analytics workloads, and enables access control lists (ACLs)",
	}
	if err := survey.AskOne(hierarchicalPrompt, &advanced.HierarchicalNamespace); err != nil {
		return nil, err
	}

	// SFTP (only if hierarchical namespace is enabled)
	if advanced.HierarchicalNamespace {
		sftpPrompt := &survey.Confirm{
			Message: "Enable SFTP?",
			Default: false,
			Help:    "SFTP can only be enabled for hierarchical namespace accounts",
		}
		if err := survey.AskOne(sftpPrompt, &advanced.EnableSFTP); err != nil {
			return nil, err
		}

		// NFS v3 (only if hierarchical namespace is enabled)
		nfsPrompt := &survey.Confirm{
			Message: "Enable network file system v3 (NFS v3)?",
			Default: false,
			Help:    "To enable NFS v3 'hierarchical namespace' must be enabled",
		}
		if err := survey.AskOne(nfsPrompt, &advanced.EnableNFSv3); err != nil {
			return nil, err
		}
	}

	// Cross-tenant replication
	crossTenantPrompt := &survey.Confirm{
		Message: "Allow cross-tenant replication?",
		Default: true,
		Help:    "Allow replication between different Azure tenants",
	}
	if err := survey.AskOne(crossTenantPrompt, &advanced.CrossTenantReplication); err != nil {
		return nil, err
	}

	// Large file shares
	largeFilePrompt := &survey.Confirm{
		Message: "Enable large file shares (Azure Files)?",
		Default: false,
		Help:    "Support for file shares up to 100 TiB",
	}
	if err := survey.AskOne(largeFilePrompt, &advanced.LargeFileShares); err != nil {
		return nil, err
	}

	return advanced, nil
}