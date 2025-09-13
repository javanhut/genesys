package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/javanhut/genesys/pkg/provider/aws"
	"github.com/javanhut/genesys/pkg/state"
	"github.com/javanhut/genesys/pkg/validation"
)

// EC2ComputeResource represents a compute resource configuration
type EC2ComputeResource struct {
	Name           string            `toml:"name"`
	Type           string            `toml:"type"`
	Image          string            `toml:"image"`
	Count          int               `toml:"count,omitempty"`
	KeyPair        string            `toml:"key_pair,omitempty"`
	SecurityGroups []string          `toml:"security_groups,omitempty"`
	SubnetId       string            `toml:"subnet_id,omitempty"`
	PublicIP       bool              `toml:"public_ip,omitempty"`
	UserData       string            `toml:"user_data,omitempty"`
	Storage        *EC2StorageConfig `toml:"storage,omitempty"`
	Tags           map[string]string `toml:"tags,omitempty"`
}

// EC2StorageConfig represents EBS storage configuration
type EC2StorageConfig struct {
	Size         int    `toml:"size"`                                   // GB
	VolumeType   string `toml:"volume_type,omitempty"` // gp3, gp2, io1, etc.
	Encrypted    bool   `toml:"encrypted,omitempty"`
	DeleteOnTerm bool   `toml:"delete_on_termination,omitempty"`
}

// EC2InstanceConfig represents a simple EC2 instance configuration
type EC2InstanceConfig struct {
	Provider string `toml:"provider"`
	Region   string `toml:"region"`

	Resources struct {
		Compute []EC2ComputeResource `toml:"compute"`
	} `toml:"resources"`

	Policies struct {
		RequireEncryption bool     `toml:"require_encryption"`
		NoPublicInstances bool     `toml:"no_public_instances"`
		RequireTags       []string `toml:"require_tags,omitempty"`
	} `toml:"policies"`

	AMILookup *EC2AMILookupConfig `toml:"ami_lookup,omitempty"`
	IAM       *UnifiedIAMConfig   `toml:"iam,omitempty"`
}

// EC2AMILookupConfig controls AMI resolution behavior
type EC2AMILookupConfig struct {
	Strategy         string `toml:"strategy,omitempty"`                     // "auto", "ssm", "describe", "static"
	DisableCache     bool   `toml:"disable_cache,omitempty"`           // Disable AMI caching
	CacheTTLHours    int    `toml:"cache_ttl_hours,omitempty"`       // Cache TTL in hours (default 24)
	FallbackToStatic bool   `toml:"fallback_to_static,omitempty"` // Allow fallback to static mappings
}

// InteractiveEC2Config manages interactive EC2 instance configuration
type InteractiveEC2Config struct {
	configDir string
}

// NewInteractiveEC2Config creates a new interactive EC2 configuration manager
func NewInteractiveEC2Config() (*InteractiveEC2Config, error) {
	ic, err := NewInteractiveConfig()
	if err != nil {
		return nil, err
	}

	return &InteractiveEC2Config{
		configDir: ic.configDir,
	}, nil
}

// CreateInstanceConfig creates an interactive EC2 instance configuration
func (iec *InteractiveEC2Config) CreateInstanceConfig() (*EC2InstanceConfig, string, error) {
	fmt.Println("EC2 Instance Configuration Wizard")
	fmt.Println("Let's create a simple EC2 instance configuration!")
	fmt.Println("")

	config := &EC2InstanceConfig{
		Provider: "aws",
	}

	// Get instance name
	instanceName, err := iec.getInstanceName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get instance name: %w", err)
	}

	// Get region
	region, err := iec.getRegion()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get region: %w", err)
	}
	config.Region = region

	// Get instance configuration
	instanceConfig, err := iec.getInstanceConfiguration(instanceName, region)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get instance configuration: %w", err)
	}

	// Add compute resource
	config.Resources.Compute = []EC2ComputeResource{instanceConfig}

	// Set policies
	config.Policies.RequireEncryption = instanceConfig.Storage != nil && instanceConfig.Storage.Encrypted
	config.Policies.NoPublicInstances = !instanceConfig.PublicIP
	if len(instanceConfig.Tags) > 0 {
		for tag := range instanceConfig.Tags {
			config.Policies.RequireTags = append(config.Policies.RequireTags, tag)
		}
	}

	// Get IAM role configuration
	iamConfig, err := iec.getIAMConfiguration(instanceName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get IAM configuration: %w", err)
	}
	if iamConfig != nil {
		config.IAM = iamConfig
	}

	return config, instanceName, nil
}

func (iec *InteractiveEC2Config) getInstanceName() (string, error) {
	var rawName string
	prompt := &survey.Input{
		Message: "Instance name:",
		Help:    "Enter any name - it will be automatically formatted for AWS EC2",
		Default: "my-ec2-instance",
	}

	if err := survey.AskOne(prompt, &rawName, survey.WithValidator(survey.Required)); err != nil {
		return "", err
	}

	// Auto-format the name
	formattedName, err := validation.ValidateAndFormatName("ec2", rawName)
	if err != nil {
		return "", fmt.Errorf("invalid instance name: %w", err)
	}

	// Show the user what will be used if it changed
	if formattedName != rawName {
		fmt.Printf("✓ Name formatted for AWS EC2: %s → %s\n", rawName, formattedName)

		// Confirm with user
		confirm := true
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Use formatted name '%s'?", formattedName),
			Default: true,
			Help:    "AWS EC2 requires specific naming rules",
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return "", err
		}

		if !confirm {
			fmt.Println("Please enter a different name...")
			return iec.getInstanceName() // Ask again
		}
	}

	// Check for existing instances with the same name
	if err := iec.validateUniqueName(formattedName); err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Please choose a different name...")
		return iec.getInstanceName() // Ask again
	}

	return formattedName, nil
}

func (iec *InteractiveEC2Config) getRegion() (string, error) {
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

func (iec *InteractiveEC2Config) getInstanceConfiguration(instanceName string, region string) (EC2ComputeResource, error) {
	config := EC2ComputeResource{
		Name:  instanceName,
		Count: 1,
	}

	// Instance type
	instanceTypes := []string{
		"t3.micro (t3.micro - 2 vCPU, 1GB RAM) ~ FREE TIER - $0/month for first year",
		"small (t3.small - 2 vCPU, 2GB RAM) ~ FREE TIER - $0/month for first year",
		"c7i-flex.large (c7i-flex.large - 2 vCPU, 4GB RAM) ~ FREE TIER - $0/month for first year",
		"m7i-flex.large (m7i-flex.large - 2 vCPU, 8GB RAM) ~ FREE TIER - $0/month for first year",
		"t2.micro (t2.micro - 1 vCPU, 1GB RAM) ~ $8/month",
		"medium (t3.medium - 2 vCPU, 4GB RAM) ~ $30/month",
		"large (t3.large - 2 vCPU, 8GB RAM) ~ $60/month",
		"xlarge (t3.xlarge - 4 vCPU, 16GB RAM) ~ $120/month",
	}

	var selectedType string
	typePrompt := &survey.Select{
		Message: "Select instance type:",
		Options: instanceTypes,
		Default: instanceTypes[0], // Default to Free Tier t2.micro
		Help:    "Free Tier options available for new AWS accounts (750 hours/month for first year). Estimated costs shown are for us-east-1 region, 24/7 usage",
	}
	if err := survey.AskOne(typePrompt, &selectedType); err != nil {
		return config, err
	}

	// Extract the actual type from the selection
	config.Type = strings.Split(strings.Split(selectedType, "(")[1], " ")[0]

	// Show cost warning for expensive instances
	if strings.Contains(config.Type, "xlarge") {
		fmt.Printf("\nHIGH COST WARNING: %s instances cost ~$120+/month\n", config.Type)
		fmt.Printf("Consider using a smaller instance type for development/testing.\n")

		var proceedWithExpensive bool
		expensivePrompt := &survey.Confirm{
			Message: fmt.Sprintf("Continue with %s instance type?", config.Type),
			Default: false,
		}
		if err := survey.AskOne(expensivePrompt, &proceedWithExpensive); err != nil {
			return config, err
		}
		if !proceedWithExpensive {
			// Let user re-select
			if err := survey.AskOne(typePrompt, &selectedType); err != nil {
				return config, err
			}
			config.Type = strings.Split(strings.Split(selectedType, "(")[1], " ")[0]
		}
	}

	// AMI/Image selection
	imageOptions := []string{
		"ubuntu-lts (Ubuntu 22.04 LTS)",
		"amazon-linux (Amazon Linux 2)",
		"centos (CentOS 7)",
		"custom (Enter custom AMI ID)",
	}

	var selectedImage string
	imagePrompt := &survey.Select{
		Message: "Select AMI/Image:",
		Options: imageOptions,
		Default: imageOptions[0], // ubuntu-lts
	}
	if err := survey.AskOne(imagePrompt, &selectedImage); err != nil {
		return config, err
	}

	if strings.HasPrefix(selectedImage, "custom") {
		var customAMI string
		customPrompt := &survey.Input{
			Message: "Enter custom AMI ID (e.g., ami-12345678):",
		}
		if err := survey.AskOne(customPrompt, &customAMI, survey.WithValidator(survey.Required)); err != nil {
			return config, err
		}
		config.Image = customAMI
	} else {
		config.Image = strings.Split(selectedImage, " ")[0]
	}

	// Key pair
	var useKeyPair bool
	keyPairPrompt := &survey.Confirm{
		Message: "Use an existing key pair for SSH access?",
		Help:    "Required if you want to SSH into the instance",
		Default: true,
	}
	if err := survey.AskOne(keyPairPrompt, &useKeyPair); err != nil {
		return config, err
	}

	if useKeyPair {
		var keyPair string
		keyPrompt := &survey.Input{
			Message: "Key pair name:",
			Help:    "The name of an existing EC2 key pair in your AWS account",
		}
		if err := survey.AskOne(keyPrompt, &keyPair); err != nil {
			return config, err
		}
		config.KeyPair = keyPair
	}

	// Public IP
	publicIPPrompt := &survey.Confirm{
		Message: "Assign a public IP address?",
		Help:    "Allows internet access to/from the instance",
		Default: true,
	}
	if err := survey.AskOne(publicIPPrompt, &config.PublicIP); err != nil {
		return config, err
	}

	// Storage configuration
	var configureStorage bool
	storagePrompt := &survey.Confirm{
		Message: "Configure custom storage settings?",
		Help:    "Customize EBS volume size, type, and encryption",
		Default: true,
	}
	if err := survey.AskOne(storagePrompt, &configureStorage); err != nil {
		return config, err
	}

	if configureStorage {
		storage, err := iec.getStorageConfig()
		if err != nil {
			return config, err
		}
		config.Storage = storage
	}

	// User data script
	var addUserData bool
	userDataPrompt := &survey.Confirm{
		Message: "Add startup script (user data)?",
		Help:    "Script that runs when the instance first boots",
		Default: false,
	}
	if err := survey.AskOne(userDataPrompt, &addUserData); err != nil {
		return config, err
	}

	if addUserData {
		userDataOptions := []string{
			"update-system (Update system packages)",
			"install-docker (Install Docker)",
			"install-node (Install Node.js and npm)",
			"custom (Enter custom script)",
		}

		var selectedUserData string
		userDataSelectPrompt := &survey.Select{
			Message: "Select user data script:",
			Options: userDataOptions,
		}
		if err := survey.AskOne(userDataSelectPrompt, &selectedUserData); err != nil {
			return config, err
		}

		switch {
		case strings.HasPrefix(selectedUserData, "update-system"):
			config.UserData = `#!/bin/bash
yum update -y || apt-get update && apt-get upgrade -y
`
		case strings.HasPrefix(selectedUserData, "install-docker"):
			config.UserData = `#!/bin/bash
yum update -y || apt-get update
yum install -y docker || apt-get install -y docker.io
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user || usermod -a -G docker ubuntu
`
		case strings.HasPrefix(selectedUserData, "install-node"):
			config.UserData = `#!/bin/bash
curl -fsSL https://rpm.nodesource.com/setup_lts.x | bash - || curl -fsSL https://deb.nodesource.com/setup_lts.x | bash -
yum install -y nodejs || apt-get install -y nodejs
`
		case strings.HasPrefix(selectedUserData, "custom"):
			var customUserData string
			customUserDataPrompt := &survey.Multiline{
				Message: "Enter custom user data script:",
			}
			if err := survey.AskOne(customUserDataPrompt, &customUserData); err != nil {
				return config, err
			}
			config.UserData = customUserData
		}
	}

	// Tags
	var addTags bool
	tagsPrompt := &survey.Confirm{
		Message: "Add tags to the instance?",
		Help:    "Tags help organize and manage your resources",
		Default: true,
	}
	if err := survey.AskOne(tagsPrompt, &addTags); err != nil {
		return config, err
	}

	if addTags {
		tags, err := iec.getTags()
		if err != nil {
			return config, err
		}
		if len(tags) > 0 {
			config.Tags = tags
		}
	}

	// Show cost estimate for the configuration
	fmt.Println("\n" + strings.Repeat("=", 80))
	estimate, err := EstimateEC2Costs(config, region)
	if err == nil {
		fmt.Println(estimate.FormatCostEstimate())
	} else {
		fmt.Printf("Cost estimate unavailable: %v\n", err)
	}
	fmt.Println(strings.Repeat("=", 80))

	// Ask for final confirmation
	var confirmProceed bool
	confirmPrompt := &survey.Confirm{
		Message: "Proceed with this configuration?",
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &confirmProceed); err != nil {
		return config, err
	}

	if !confirmProceed {
		return config, fmt.Errorf("configuration cancelled by user")
	}

	return config, nil
}

func (iec *InteractiveEC2Config) getStorageConfig() (*EC2StorageConfig, error) {
	storage := &EC2StorageConfig{
		DeleteOnTerm: true, // Default to delete on termination
	}

	// Volume size
	var sizeStr string
	sizePrompt := &survey.Input{
		Message: "Root volume size (GB):",
		Default: "20",
		Help:    "Size of the root EBS volume in gigabytes",
	}
	if err := survey.AskOne(sizePrompt, &sizeStr); err != nil {
		return nil, err
	}

	if _, err := fmt.Sscanf(sizeStr, "%d", &storage.Size); err != nil {
		storage.Size = 20 // Default
	}

	// Warn about large storage sizes
	if storage.Size > 100 {
		estimatedCost := float64(storage.Size) * 0.08 // Approximate gp3 cost
		fmt.Printf("\nSTORAGE COST WARNING: %d GB will cost ~$%.2f/month\n", storage.Size, estimatedCost)
		if storage.Size > 500 {
			fmt.Printf("Large storage volumes (>500GB) can be expensive!\n")
		}
	}

	// Volume type
	volumeTypes := []string{
		"gp3 (General Purpose SSD v3 - Latest)",
		"gp2 (General Purpose SSD v2)",
		"io1 (Provisioned IOPS SSD)",
		"st1 (Throughput Optimized HDD)",
	}

	var selectedVolumeType string
	volumeTypePrompt := &survey.Select{
		Message: "Select volume type:",
		Options: volumeTypes,
		Default: volumeTypes[0], // gp3
	}
	if err := survey.AskOne(volumeTypePrompt, &selectedVolumeType); err != nil {
		return nil, err
	}

	storage.VolumeType = strings.Split(selectedVolumeType, " ")[0]

	// Encryption
	encryptionPrompt := &survey.Confirm{
		Message: "Enable EBS encryption?",
		Help:    "Encrypt the EBS volume at rest",
		Default: true,
	}
	if err := survey.AskOne(encryptionPrompt, &storage.Encrypted); err != nil {
		return nil, err
	}

	// Delete on termination
	deletePrompt := &survey.Confirm{
		Message: "Delete volume when instance is terminated?",
		Help:    "If false, the EBS volume will persist after instance termination",
		Default: true,
	}
	if err := survey.AskOne(deletePrompt, &storage.DeleteOnTerm); err != nil {
		return nil, err
	}

	return storage, nil
}

func (iec *InteractiveEC2Config) getTags() (map[string]string, error) {
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

// validateUniqueName checks if an EC2 instance name is already in use
func (iec *InteractiveEC2Config) validateUniqueName(name string) error {
	// Basic name validation first
	if name == "" {
		return fmt.Errorf("instance name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("instance name cannot contain spaces")
	}

	// Check local state first (fastest check)
	if err := iec.checkLocalState(name); err != nil {
		return err
	}

	// Check AWS for existing instances with the same Name tag
	if err := iec.checkAWSInstances(name); err != nil {
		return err
	}

	return nil
}

// checkLocalState checks if name exists in local state
func (iec *InteractiveEC2Config) checkLocalState(name string) error {
	localState, err := state.LoadLocalState()
	if err != nil {
		// If we can't load local state, continue with AWS check
		return nil
	}

	// Check if any existing resource has this name
	for _, resource := range localState.Resources {
		if resource.Type == "ec2" && resource.Name == name {
			return fmt.Errorf("instance name '%s' already exists (created %s)", name, resource.CreatedAt.Format("2006-01-02 15:04"))
		}
	}

	return nil
}

// checkAWSInstances checks if name exists in AWS (running or stopped instances)
func (iec *InteractiveEC2Config) checkAWSInstances(name string) error {
	// Get region first
	region, err := iec.getRegion()
	if err != nil {
		// If we can't determine region, allow the name but don't block
		return nil
	}

	// Create AWS provider to check existing instances
	provider, err := aws.NewAWSProvider(region)
	if err != nil {
		// If we can't connect to AWS, allow the name but warn
		return nil
	}

	computeService := provider.Compute()

	// Search for instances with this Name tag
	ctx := context.Background()
	instances, err := computeService.ListInstances(ctx, map[string]string{
		"tag:Name":            name,
		"instance-state-name": "running,stopped,stopping,pending",
	})

	if err != nil {
		// If we can't query AWS, allow the name but don't block
		return nil
	}

	if len(instances) > 0 {
		states := make([]string, len(instances))
		for i, instance := range instances {
			states[i] = instance.State
		}
		return fmt.Errorf("instance name '%s' already exists in AWS (%d instance(s): %s)",
			name, len(instances), strings.Join(states, ", "))
	}

	return nil
}

func (iec *InteractiveEC2Config) getIAMConfiguration(instanceName string) (*UnifiedIAMConfig, error) {
	var configureIAM bool
	iamPrompt := &survey.Confirm{
		Message: "Configure IAM instance profile for EC2?",
		Help:    "Create or use an existing IAM role that will be attached as an instance profile for AWS service access",
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
		Help:    "Genesys can create a new role with proper EC2 permissions or use an existing role",
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
		defaultRoleName := FormatRoleName("ec2", instanceName)
		
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
	
	defaultPolicies := getDefaultPoliciesForResource("ec2")
	fmt.Println("Default policies for EC2 access:")
	for i, policy := range defaultPolicies {
		fmt.Printf("  %d. %s\n", i+1, policy)
	}

	var useDefaults bool
	defaultsPrompt := &survey.Confirm{
		Message: "Use default EC2 permissions?",
		Help:    "These permissions allow Systems Manager access and CloudWatch logging",
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
			"Systems Manager Parameter access",
			"CloudWatch full access",
			"S3 full access",
			"S3 read-only access", 
			"DynamoDB read/write access",
			"Lambda full access",
			"Secrets Manager read access",
			"X-Ray tracing",
		}

		var selectedPolicies []string
		policyPrompt := &survey.MultiSelect{
			Message: "Select required policies:",
			Options: availablePolicies,
			Default: []string{"Systems Manager Parameter access", "CloudWatch full access"},
			Help:    "Choose the AWS managed policies this role should have",
		}
		if err := survey.AskOne(policyPrompt, &selectedPolicies); err != nil {
			return nil, err
		}
		iamConfig.RequiredPolicies = selectedPolicies
	}

	// Set trust policy
	iamConfig.TrustPolicy = "ec2"

	return iamConfig, nil
}

// SaveConfig saves the EC2 instance configuration to a file
func (iec *InteractiveEC2Config) SaveConfig(config *EC2InstanceConfig, instanceName string) (string, error) {
	// Create directory structure: resources/ec2/
	dirPath := filepath.Join("resources", "ec2")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate filename based on instance name
	fileName := fmt.Sprintf("%s.toml", instanceName)
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
