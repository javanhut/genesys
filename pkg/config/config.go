package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// ConfigManager handles configuration loading and reloading
type ConfigManager struct {
	configPath   string
	lastModified time.Time
	cachedConfig *Config
}

// Config represents the main configuration structure
type Config struct {
	Provider  string             `toml:"provider"`
	Region    string             `toml:"region"`
	Project   string             `toml:"project,omitempty"` // For GCP
	Outcomes  map[string]Outcome `toml:"outcomes,omitempty"`
	Resources Resources          `toml:"resources,omitempty"`
	State     StateConfig        `toml:"state,omitempty"`
	Policies  Policies           `toml:"policies,omitempty"`
}

// Outcome represents a high-level deployment outcome
type Outcome struct {
	// Static Site
	Domain      string `toml:"domain,omitempty"`
	EnableCDN   bool   `toml:"enable_cdn,omitempty"`
	EnableHTTPS bool   `toml:"enable_https,omitempty"`

	// API/Function
	Name    string `toml:"name,omitempty"`
	Runtime string `toml:"runtime,omitempty"`
	Memory  int    `toml:"memory,omitempty"`
	Timeout int    `toml:"timeout,omitempty"`

	// Database
	Engine  string `toml:"engine,omitempty"`
	Version string `toml:"version,omitempty"`
	Size    string `toml:"size,omitempty"`
}

// Resources represents infrastructure resources
type Resources struct {
	Compute    []ComputeResource    `toml:"compute,omitempty"`
	Storage    []StorageResource    `toml:"storage,omitempty"`
	Network    []NetworkResource    `toml:"network,omitempty"`
	Database   []DatabaseResource   `toml:"database,omitempty"`
	Serverless []ServerlessResource `toml:"serverless,omitempty"`
}

// ComputeResource represents a compute instance configuration
type ComputeResource struct {
	Name           string            `toml:"name"`
	Type           string            `toml:"type"` // small|medium|large|xlarge
	Image          string            `toml:"image"`
	Count          int               `toml:"count,omitempty"`
	Network        string            `toml:"network,omitempty"`
	SecurityGroups []string          `toml:"security_groups,omitempty"`
	Tags           map[string]string `toml:"tags,omitempty"`
}

// StorageResource represents storage configuration
type StorageResource struct {
	Name         string            `toml:"name"`
	Type         string            `toml:"type"` // bucket|volume
	Versioning   bool              `toml:"versioning,omitempty"`
	Encryption   bool              `toml:"encryption,omitempty"`
	PublicAccess bool              `toml:"public_access,omitempty"`
	Lifecycle    *LifecycleConfig  `toml:"lifecycle,omitempty"`
	Tags         map[string]string `toml:"tags,omitempty"`
}

// LifecycleConfig for storage lifecycle
type LifecycleConfig struct {
	DeleteAfterDays  int `toml:"delete_after_days,omitempty"`
	ArchiveAfterDays int `toml:"archive_after_days,omitempty"`
}

// NetworkResource represents network configuration
type NetworkResource struct {
	Name    string            `toml:"name"`
	CIDR    string            `toml:"cidr"`
	Subnets []SubnetConfig    `toml:"subnets,omitempty"`
	Tags    map[string]string `toml:"tags,omitempty"`
}

// SubnetConfig represents subnet configuration
type SubnetConfig struct {
	Name   string `toml:"name"`
	CIDR   string `toml:"cidr"`
	Public bool   `toml:"public,omitempty"`
	AZ     string `toml:"az,omitempty"`
}

// DatabaseResource represents database configuration
type DatabaseResource struct {
	Name    string            `toml:"name"`
	Engine  string            `toml:"engine"`
	Version string            `toml:"version"`
	Size    string            `toml:"size"`       // small|medium|large
	Storage int               `toml:"storage"` // GB
	MultiAZ bool              `toml:"multi_az,omitempty"`
	Backup  *BackupConfig     `toml:"backup,omitempty"`
	Tags    map[string]string `toml:"tags,omitempty"`
}

// BackupConfig for database backups
type BackupConfig struct {
	RetentionDays int    `toml:"retention_days"`
	Window        string `toml:"window,omitempty"`
}

// ServerlessResource represents serverless function configuration
type ServerlessResource struct {
	Name        string            `toml:"name"`
	Runtime     string            `toml:"runtime"`
	Handler     string            `toml:"handler"`
	Memory      int               `toml:"memory,omitempty"`
	Timeout     int               `toml:"timeout,omitempty"`
	Environment map[string]string `toml:"environment,omitempty"`
	Triggers    []TriggerConfig   `toml:"triggers,omitempty"`
	Tags        map[string]string `toml:"tags,omitempty"`
}

// TriggerConfig for serverless triggers
type TriggerConfig struct {
	Type     string   `toml:"type"` // http|schedule|queue|storage
	Path     string   `toml:"path,omitempty"`
	Methods  []string `toml:"methods,omitempty"`
	Schedule string   `toml:"schedule,omitempty"`
}

// StateConfig for state management
type StateConfig struct {
	Backend   string `toml:"backend,omitempty"` // s3|gcs|azureblob|local
	Bucket    string `toml:"bucket,omitempty"`
	LockTable string `toml:"lock_table,omitempty"`
	Encrypt   bool   `toml:"encrypt,omitempty"`
}

// Policies for governance
type Policies struct {
	NoPublicBuckets   bool     `toml:"no_public_buckets,omitempty"`
	RequireEncryption bool     `toml:"require_encryption,omitempty"`
	RequireTags       []string `toml:"require_tags,omitempty"`
	MaxCostPerMonth   float64  `toml:"max_cost_per_month,omitempty"`
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config

	// Only support TOML format
	switch filepath.Ext(path) {
	case ".toml":
		_, err = toml.Decode(string(data), &config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TOML: %w", err)
		}
	default:
		// Try to parse as TOML
		_, err = toml.Decode(string(data), &config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config as TOML: %w", err)
		}
	}

	// Apply defaults and validate
	ApplyDefaults(&config)
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, path string) error {
	var data []byte
	var err error

	switch filepath.Ext(path) {
	case ".toml":
		buf := new(bytes.Buffer)
		err = toml.NewEncoder(buf).Encode(config)
		data = buf.Bytes()
	default:
		// Default to TOML
		buf := new(bytes.Buffer)
		err = toml.NewEncoder(buf).Encode(config)
		data = buf.Bytes()
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig loads configuration with caching and change detection
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	stat, err := os.Stat(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Check if we need to reload
	if cm.cachedConfig != nil && stat.ModTime().Equal(cm.lastModified) {
		return cm.cachedConfig, nil
	}

	// Load fresh configuration
	config, err := LoadConfig(cm.configPath)
	if err != nil {
		return nil, err
	}

	// Cache the configuration
	cm.cachedConfig = config
	cm.lastModified = stat.ModTime()

	return config, nil
}

// ReloadConfig forces a reload of the configuration
func (cm *ConfigManager) ReloadConfig() (*Config, error) {
	cm.cachedConfig = nil
	cm.lastModified = time.Time{}
	return cm.LoadConfig()
}

// IsConfigChanged checks if the configuration file has been modified
func (cm *ConfigManager) IsConfigChanged() (bool, error) {
	stat, err := os.Stat(cm.configPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	return !stat.ModTime().Equal(cm.lastModified), nil
}

// RefreshProviderCredentials reloads provider credentials from config files
func RefreshProviderCredentials() error {
	// Refresh AWS credentials
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsConfigFile := filepath.Join(homeDir, ".genesys", "aws.json")
	if _, err := os.Stat(awsConfigFile); err == nil {
		// AWS config exists, try to refresh credentials
		data, err := os.ReadFile(awsConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read AWS config: %w", err)
		}

		var creds struct {
			UseLocal bool `json:"use_local"`
		}
		if err := json.Unmarshal(data, &creds); err == nil && creds.UseLocal {
			// Try to refresh from AWS credentials file
			credFile := filepath.Join(homeDir, ".aws", "credentials")
			if _, err := os.Stat(credFile); err == nil {
				// TODO: Implement credential refresh logic
			}
		}
	}

	return nil
}
