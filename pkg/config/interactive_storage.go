package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/javanhut/genesys/pkg/provider/aws"
)

// saveProviderConfig saves provider configuration to disk
func (ic *InteractiveConfig) SaveProviderConfig(config *ProviderCredentials) error {
	ic.providers[config.Provider] = config

	// Save individual provider config
	providerFile := filepath.Join(ic.configDir, fmt.Sprintf("%s.json", config.Provider))
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(providerFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Update or create global config with default provider
	if config.DefaultConfig {
		if err := ic.saveGlobalConfig(config.Provider); err != nil {
			return fmt.Errorf("failed to save global config: %w", err)
		}
	}

	// Set environment variables if not using local credentials
	if !config.UseLocal && len(config.Credentials) > 0 {
		if err := ic.setEnvironmentVariables(config); err != nil {
			fmt.Printf("[WARNING] Could not set environment variables: %v\n", err)
		}
	}

	fmt.Printf("Configuration saved to: %s\n", providerFile)
	return nil
}

// saveGlobalConfig saves global configuration
func (ic *InteractiveConfig) saveGlobalConfig(defaultProvider string) error {
	globalConfig := map[string]interface{}{
		"default_provider": defaultProvider,
		"version":         "1.0",
	}

	globalFile := filepath.Join(ic.configDir, "config.json")
	data, err := json.MarshalIndent(globalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	return os.WriteFile(globalFile, data, 0644)
}

// setEnvironmentVariables sets environment variables for the session
func (ic *InteractiveConfig) setEnvironmentVariables(config *ProviderCredentials) error {
	switch config.Provider {
	case "aws":
		return ic.setAWSEnvironmentVariables(config)
	case "gcp":
		return ic.setGCPEnvironmentVariables(config)
	case "azure":
		return ic.setAzureEnvironmentVariables(config)
	case "tencent":
		return ic.setTencentEnvironmentVariables(config)
	}
	return nil
}

func (ic *InteractiveConfig) setAWSEnvironmentVariables(config *ProviderCredentials) error {
	if accessKey, ok := config.Credentials["access_key_id"]; ok {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	}
	if secretKey, ok := config.Credentials["secret_access_key"]; ok {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}
	if sessionToken, ok := config.Credentials["session_token"]; ok {
		os.Setenv("AWS_SESSION_TOKEN", sessionToken)
	}
	if config.Region != "" {
		os.Setenv("AWS_DEFAULT_REGION", config.Region)
	}
	if profile, ok := config.Credentials["profile"]; ok {
		os.Setenv("AWS_PROFILE", profile)
	}
	return nil
}

func (ic *InteractiveConfig) setGCPEnvironmentVariables(config *ProviderCredentials) error {
	if projectID, ok := config.Credentials["project_id"]; ok {
		os.Setenv("GOOGLE_CLOUD_PROJECT", projectID)
	}
	if keyFile, ok := config.Credentials["service_account_key"]; ok {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", keyFile)
	}
	return nil
}

func (ic *InteractiveConfig) setAzureEnvironmentVariables(config *ProviderCredentials) error {
	if clientID, ok := config.Credentials["client_id"]; ok {
		os.Setenv("AZURE_CLIENT_ID", clientID)
	}
	if clientSecret, ok := config.Credentials["client_secret"]; ok {
		os.Setenv("AZURE_CLIENT_SECRET", clientSecret)
	}
	if tenantID, ok := config.Credentials["tenant_id"]; ok {
		os.Setenv("AZURE_TENANT_ID", tenantID)
	}
	if subscriptionID, ok := config.Credentials["subscription_id"]; ok {
		os.Setenv("AZURE_SUBSCRIPTION_ID", subscriptionID)
	}
	return nil
}

func (ic *InteractiveConfig) setTencentEnvironmentVariables(config *ProviderCredentials) error {
	if secretID, ok := config.Credentials["secret_id"]; ok {
		os.Setenv("TENCENTCLOUD_SECRET_ID", secretID)
	}
	if secretKey, ok := config.Credentials["secret_key"]; ok {
		os.Setenv("TENCENTCLOUD_SECRET_KEY", secretKey)
	}
	if securityToken, ok := config.Credentials["security_token"]; ok {
		os.Setenv("TENCENTCLOUD_SECURITY_TOKEN", securityToken)
	}
	if config.Region != "" {
		os.Setenv("TENCENTCLOUD_REGION", config.Region)
	}
	return nil
}

// ValidateCredentials validates provider credentials
func (ic *InteractiveConfig) ValidateCredentials(config *ProviderCredentials) error {
	switch config.Provider {
	case "aws":
		return ic.validateAWSCredentials(config)
	case "gcp":
		return ic.validateGCPCredentials(config)
	case "azure":
		return ic.validateAzureCredentials(config)
	case "tencent":
		return ic.validateTencentCredentials(config)
	default:
		return fmt.Errorf("validation not implemented for provider: %s", config.Provider)
	}
}

func (ic *InteractiveConfig) validateAWSCredentials(config *ProviderCredentials) error {
	if config.UseLocal {
		// For local credentials, just try to create a provider
		_, err := aws.NewAWSProvider(config.Region)
		return err
	}

	// For provided credentials, set them temporarily and test
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	originalSessionToken := os.Getenv("AWS_SESSION_TOKEN")

	defer func() {
		// Restore original values
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
		os.Setenv("AWS_SESSION_TOKEN", originalSessionToken)
	}()

	// Set test credentials
	ic.setAWSEnvironmentVariables(config)

	// Test credentials
	provider, err := aws.NewAWSProvider(config.Region)
	if err != nil {
		return err
	}

	return provider.Validate()
}

func (ic *InteractiveConfig) validateGCPCredentials(config *ProviderCredentials) error {
	// Basic validation - check if required fields are present
	if config.UseLocal {
		// Check if local credentials are available
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
			return fmt.Errorf("no local GCP credentials found")
		}
	} else {
		if _, ok := config.Credentials["project_id"]; !ok {
			return fmt.Errorf("project_id is required for GCP")
		}
	}
	return nil
}

func (ic *InteractiveConfig) validateAzureCredentials(config *ProviderCredentials) error {
	// Basic validation - check if required fields are present
	if config.UseLocal {
		return nil // Assume Azure CLI is properly configured
	}

	authMethod := config.Credentials["auth_method"]
	switch authMethod {
	case "service_principal":
		required := []string{"client_id", "client_secret", "tenant_id", "subscription_id"}
		for _, field := range required {
			if _, ok := config.Credentials[field]; !ok {
				return fmt.Errorf("%s is required for Azure service principal authentication", field)
			}
		}
	case "managed_identity":
		if _, ok := config.Credentials["subscription_id"]; !ok {
			return fmt.Errorf("subscription_id is required for Azure managed identity")
		}
	}
	return nil
}

func (ic *InteractiveConfig) validateTencentCredentials(config *ProviderCredentials) error {
	// Basic validation - check if required fields are present
	if config.UseLocal {
		if os.Getenv("TENCENTCLOUD_SECRET_ID") == "" || os.Getenv("TENCENTCLOUD_SECRET_KEY") == "" {
			return fmt.Errorf("no local Tencent Cloud credentials found")
		}
	} else {
		required := []string{"secret_id", "secret_key"}
		for _, field := range required {
			if _, ok := config.Credentials[field]; !ok {
				return fmt.Errorf("%s is required for Tencent Cloud", field)
			}
		}
	}
	return nil
}

// LoadProviderConfig loads a provider configuration from disk
func (ic *InteractiveConfig) LoadProviderConfig(provider string) (*ProviderCredentials, error) {
	providerFile := filepath.Join(ic.configDir, fmt.Sprintf("%s.json", provider))
	
	data, err := os.ReadFile(providerFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider config: %w", err)
	}

	var config ProviderCredentials
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse provider config: %w", err)
	}

	return &config, nil
}

// ListConfiguredProviders lists all configured providers
func (ic *InteractiveConfig) ListConfiguredProviders() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(ic.configDir, "*.json"))
	if err != nil {
		return nil, err
	}

	var providers []string
	for _, file := range files {
		basename := filepath.Base(file)
		if basename != "config.json" { // Skip global config
			provider := basename[:len(basename)-5] // Remove .json extension
			providers = append(providers, provider)
		}
	}

	return providers, nil
}