package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// ProviderCredentials represents credentials for a cloud provider
type ProviderCredentials struct {
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	Credentials   map[string]string `json:"credentials"`
	UseLocal      bool              `json:"use_local"`
	DefaultConfig bool              `json:"default_config"`
}

// InteractiveConfig manages interactive credential configuration
type InteractiveConfig struct {
	configDir string
	providers map[string]*ProviderCredentials
}

// NewInteractiveConfig creates a new interactive configuration manager
func NewInteractiveConfig() (*InteractiveConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".genesys")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &InteractiveConfig{
		configDir: configDir,
		providers: make(map[string]*ProviderCredentials),
	}, nil
}

// ConfigureProvider starts the interactive configuration process
func (ic *InteractiveConfig) ConfigureProvider() error {
	// Provider selection
	provider, err := ic.selectProvider()
	if err != nil {
		return fmt.Errorf("failed to select provider: %w", err)
	}

	// Check for existing local credentials
	hasLocal, localInfo := ic.checkLocalCredentials(provider)
	
	var useLocal bool
	if hasLocal {
		fmt.Printf("\n[OK] Found existing %s credentials:\n%s\n", strings.ToUpper(provider), localInfo)
		
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Use existing local %s credentials?", strings.ToUpper(provider)),
			Default: true,
		}
		if err := survey.AskOne(prompt, &useLocal); err != nil {
			return fmt.Errorf("failed to get local credential preference: %w", err)
		}
	}

	var credentials map[string]string
	var region string

	if !useLocal {
		// Get credentials interactively
		credentials, err = ic.getProviderCredentials(provider)
		if err != nil {
			return fmt.Errorf("failed to get credentials for %s: %w", provider, err)
		}
	}

	// Get region
	region, err = ic.getRegion(provider)
	if err != nil {
		return fmt.Errorf("failed to get region for %s: %w", provider, err)
	}

	// Create provider config
	providerConfig := &ProviderCredentials{
		Provider:    provider,
		Region:      region,
		Credentials: credentials,
		UseLocal:    useLocal,
	}

	// Ask if this should be the default
	var isDefault bool
	defaultPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Set %s as your default provider?", strings.ToUpper(provider)),
		Default: len(ic.providers) == 0, // Default true if first provider
	}
	if err := survey.AskOne(defaultPrompt, &isDefault); err != nil {
		return fmt.Errorf("failed to get default preference: %w", err)
	}
	providerConfig.DefaultConfig = isDefault

	// Save configuration
	if err := ic.SaveProviderConfig(providerConfig); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Validate credentials
	if err := ic.ValidateCredentials(providerConfig); err != nil {
		fmt.Printf("[WARNING] Credential validation failed: %v\n", err)
		fmt.Println("Configuration saved, but please verify your credentials manually.")
	} else {
		fmt.Printf("[OK] %s configuration saved and validated successfully!\n", strings.ToUpper(provider))
	}

	return nil
}

// selectProvider prompts user to select a cloud provider
func (ic *InteractiveConfig) selectProvider() (string, error) {
	providers := []string{
		"aws",
		"gcp", 
		"azure",
		"tencent",
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select a cloud provider to configure:",
		Options: providers,
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

	if err := survey.AskOne(prompt, &selected); err != nil {
		return "", err
	}

	return selected, nil
}

// checkLocalCredentials checks if local credentials exist for a provider
func (ic *InteractiveConfig) checkLocalCredentials(provider string) (bool, string) {
	switch provider {
	case "aws":
		return ic.checkAWSLocal()
	case "gcp":
		return ic.checkGCPLocal()
	case "azure":
		return ic.checkAzureLocal()
	case "tencent":
		return ic.checkTencentLocal()
	default:
		return false, ""
	}
}

// checkAWSLocal checks for AWS credentials
func (ic *InteractiveConfig) checkAWSLocal() (bool, string) {
	var info []string
	
	// Check environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		info = append(info, "  - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)")
		if profile := os.Getenv("AWS_PROFILE"); profile != "" {
			info = append(info, fmt.Sprintf("  - AWS Profile: %s", profile))
		}
		if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
			info = append(info, fmt.Sprintf("  - Region: %s", region))
		}
	}

	// Check AWS credentials file
	homeDir, _ := os.UserHomeDir()
	credFile := filepath.Join(homeDir, ".aws", "credentials")
	if _, err := os.Stat(credFile); err == nil {
		info = append(info, "  - AWS credentials file (~/.aws/credentials)")
	}

	// Check AWS config file
	configFile := filepath.Join(homeDir, ".aws", "config")
	if _, err := os.Stat(configFile); err == nil {
		info = append(info, "  - AWS config file (~/.aws/config)")
	}

	if len(info) > 0 {
		return true, strings.Join(info, "\n")
	}
	return false, ""
}

// checkGCPLocal checks for GCP credentials
func (ic *InteractiveConfig) checkGCPLocal() (bool, string) {
	var info []string

	// Check environment variables
	if creds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); creds != "" {
		info = append(info, fmt.Sprintf("  - Service account key: %s", creds))
	}

	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		info = append(info, fmt.Sprintf("  - Project: %s", project))
	}

	// Check gcloud default credentials
	homeDir, _ := os.UserHomeDir()
	gcloudDir := filepath.Join(homeDir, ".config", "gcloud")
	if _, err := os.Stat(gcloudDir); err == nil {
		info = append(info, "  - gcloud CLI authentication")
	}

	if len(info) > 0 {
		return true, strings.Join(info, "\n")
	}
	return false, ""
}

// checkAzureLocal checks for Azure credentials
func (ic *InteractiveConfig) checkAzureLocal() (bool, string) {
	var info []string

	// Check environment variables
	if clientId := os.Getenv("AZURE_CLIENT_ID"); clientId != "" {
		info = append(info, "  - Service Principal (AZURE_CLIENT_ID)")
	}

	if tenantId := os.Getenv("AZURE_TENANT_ID"); tenantId != "" {
		info = append(info, fmt.Sprintf("  - Tenant ID: %s", tenantId))
	}

	if subscription := os.Getenv("AZURE_SUBSCRIPTION_ID"); subscription != "" {
		info = append(info, fmt.Sprintf("  - Subscription: %s", subscription))
	}

	// Check Azure CLI
	homeDir, _ := os.UserHomeDir()
	azureDir := filepath.Join(homeDir, ".azure")
	if _, err := os.Stat(azureDir); err == nil {
		info = append(info, "  - Azure CLI authentication")
	}

	if len(info) > 0 {
		return true, strings.Join(info, "\n")
	}
	return false, ""
}

// checkTencentLocal checks for Tencent Cloud credentials
func (ic *InteractiveConfig) checkTencentLocal() (bool, string) {
	var info []string

	// Check environment variables
	if secretId := os.Getenv("TENCENTCLOUD_SECRET_ID"); secretId != "" {
		info = append(info, "  - Environment variables (TENCENTCLOUD_SECRET_ID)")
	}

	if region := os.Getenv("TENCENTCLOUD_REGION"); region != "" {
		info = append(info, fmt.Sprintf("  - Region: %s", region))
	}

	// Check Tencent CLI config
	homeDir, _ := os.UserHomeDir()
	tencentDir := filepath.Join(homeDir, ".tccli")
	if _, err := os.Stat(tencentDir); err == nil {
		info = append(info, "  - Tencent CLI authentication")
	}

	if len(info) > 0 {
		return true, strings.Join(info, "\n")
	}
	return false, ""
}

// getProviderCredentials gets credentials for a specific provider
func (ic *InteractiveConfig) getProviderCredentials(provider string) (map[string]string, error) {
	switch provider {
	case "aws":
		return ic.getAWSCredentials()
	case "gcp":
		return ic.getGCPCredentials()
	case "azure":
		return ic.getAzureCredentials()
	case "tencent":
		return ic.getTencentCredentials()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getRegion gets the region for a provider
func (ic *InteractiveConfig) getRegion(provider string) (string, error) {
	switch provider {
	case "aws":
		return ic.getAWSRegion()
	case "gcp":
		return ic.getGCPRegion()
	case "azure":
		return ic.getAzureRegion()
	case "tencent":
		return ic.getTencentRegion()
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}