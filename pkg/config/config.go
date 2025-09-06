package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ConfigManager handles configuration loading and reloading
type ConfigManager struct {
	configPath   string
	lastModified time.Time
	cachedConfig *Config
}

// Config represents the main configuration structure
type Config struct {
	Provider  string             `yaml:"provider" toml:"provider"`
	Region    string             `yaml:"region" toml:"region"`
	Project   string             `yaml:"project,omitempty" toml:"project,omitempty"` // For GCP
	Outcomes  map[string]Outcome `yaml:"outcomes,omitempty" toml:"outcomes,omitempty"`
	Resources Resources          `yaml:"resources,omitempty" toml:"resources,omitempty"`
	State     StateConfig        `yaml:"state,omitempty" toml:"state,omitempty"`
	Policies  Policies           `yaml:"policies,omitempty" toml:"policies,omitempty"`
}

// Outcome represents a high-level deployment outcome
type Outcome struct {
	// Static Site
	Domain      string `yaml:"domain,omitempty" toml:"domain,omitempty"`
	EnableCDN   bool   `yaml:"enable_cdn,omitempty" toml:"enable_cdn,omitempty"`
	EnableHTTPS bool   `yaml:"enable_https,omitempty" toml:"enable_https,omitempty"`

	// API/Function
	Name    string `yaml:"name,omitempty" toml:"name,omitempty"`
	Runtime string `yaml:"runtime,omitempty" toml:"runtime,omitempty"`
	Memory  int    `yaml:"memory,omitempty" toml:"memory,omitempty"`
	Timeout int    `yaml:"timeout,omitempty" toml:"timeout,omitempty"`

	// Database
	Engine  string `yaml:"engine,omitempty" toml:"engine,omitempty"`
	Version string `yaml:"version,omitempty" toml:"version,omitempty"`
	Size    string `yaml:"size,omitempty" toml:"size,omitempty"`
}

// Resources represents infrastructure resources
type Resources struct {
	Compute    []ComputeResource    `yaml:"compute,omitempty" toml:"compute,omitempty"`
	Storage    []StorageResource    `yaml:"storage,omitempty" toml:"storage,omitempty"`
	Network    []NetworkResource    `yaml:"network,omitempty" toml:"network,omitempty"`
	Database   []DatabaseResource   `yaml:"database,omitempty" toml:"database,omitempty"`
	Serverless []ServerlessResource `yaml:"serverless,omitempty" toml:"serverless,omitempty"`
}

// ComputeResource represents a compute instance configuration
type ComputeResource struct {
	Name           string            `yaml:"name" toml:"name"`
	Type           string            `yaml:"type" toml:"type"` // small|medium|large|xlarge
	Image          string            `yaml:"image" toml:"image"`
	Count          int               `yaml:"count,omitempty" toml:"count,omitempty"`
	Network        string            `yaml:"network,omitempty" toml:"network,omitempty"`
	SecurityGroups []string          `yaml:"security_groups,omitempty" toml:"security_groups,omitempty"`
	Tags           map[string]string `yaml:"tags,omitempty" toml:"tags,omitempty"`
}

// StorageResource represents storage configuration
type StorageResource struct {
	Name         string            `yaml:"name" toml:"name"`
	Type         string            `yaml:"type" toml:"type"` // bucket|volume
	Versioning   bool              `yaml:"versioning,omitempty" toml:"versioning,omitempty"`
	Encryption   bool              `yaml:"encryption,omitempty" toml:"encryption,omitempty"`
	PublicAccess bool              `yaml:"public_access,omitempty" toml:"public_access,omitempty"`
	Lifecycle    *LifecycleConfig  `yaml:"lifecycle,omitempty" toml:"lifecycle,omitempty"`
	Tags         map[string]string `yaml:"tags,omitempty" toml:"tags,omitempty"`
}

// LifecycleConfig for storage lifecycle
type LifecycleConfig struct {
	DeleteAfterDays  int `yaml:"delete_after_days,omitempty" toml:"delete_after_days,omitempty"`
	ArchiveAfterDays int `yaml:"archive_after_days,omitempty" toml:"archive_after_days,omitempty"`
}

// NetworkResource represents network configuration
type NetworkResource struct {
	Name    string            `yaml:"name" toml:"name"`
	CIDR    string            `yaml:"cidr" toml:"cidr"`
	Subnets []SubnetConfig    `yaml:"subnets,omitempty" toml:"subnets,omitempty"`
	Tags    map[string]string `yaml:"tags,omitempty" toml:"tags,omitempty"`
}

// SubnetConfig represents subnet configuration
type SubnetConfig struct {
	Name   string `yaml:"name" toml:"name"`
	CIDR   string `yaml:"cidr" toml:"cidr"`
	Public bool   `yaml:"public,omitempty" toml:"public,omitempty"`
	AZ     string `yaml:"az,omitempty" toml:"az,omitempty"`
}

// DatabaseResource represents database configuration
type DatabaseResource struct {
	Name    string            `yaml:"name" toml:"name"`
	Engine  string            `yaml:"engine" toml:"engine"`
	Version string            `yaml:"version" toml:"version"`
	Size    string            `yaml:"size" toml:"size"`       // small|medium|large
	Storage int               `yaml:"storage" toml:"storage"` // GB
	MultiAZ bool              `yaml:"multi_az,omitempty" toml:"multi_az,omitempty"`
	Backup  *BackupConfig     `yaml:"backup,omitempty" toml:"backup,omitempty"`
	Tags    map[string]string `yaml:"tags,omitempty" toml:"tags,omitempty"`
}

// BackupConfig for database backups
type BackupConfig struct {
	RetentionDays int    `yaml:"retention_days" toml:"retention_days"`
	Window        string `yaml:"window,omitempty" toml:"window,omitempty"`
}

// ServerlessResource represents serverless function configuration
type ServerlessResource struct {
	Name        string            `yaml:"name" toml:"name"`
	Runtime     string            `yaml:"runtime" toml:"runtime"`
	Handler     string            `yaml:"handler" toml:"handler"`
	Memory      int               `yaml:"memory,omitempty" toml:"memory,omitempty"`
	Timeout     int               `yaml:"timeout,omitempty" toml:"timeout,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty" toml:"environment,omitempty"`
	Triggers    []TriggerConfig   `yaml:"triggers,omitempty" toml:"triggers,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" toml:"tags,omitempty"`
}

// TriggerConfig for serverless triggers
type TriggerConfig struct {
	Type     string   `yaml:"type" toml:"type"` // http|schedule|queue|storage
	Path     string   `yaml:"path,omitempty" toml:"path,omitempty"`
	Methods  []string `yaml:"methods,omitempty" toml:"methods,omitempty"`
	Schedule string   `yaml:"schedule,omitempty" toml:"schedule,omitempty"`
}

// StateConfig for state management
type StateConfig struct {
	Backend   string `yaml:"backend,omitempty" toml:"backend,omitempty"` // s3|gcs|azureblob|local
	Bucket    string `yaml:"bucket,omitempty" toml:"bucket,omitempty"`
	LockTable string `yaml:"lock_table,omitempty" toml:"lock_table,omitempty"`
	Encrypt   bool   `yaml:"encrypt,omitempty" toml:"encrypt,omitempty"`
}

// Policies for governance
type Policies struct {
	NoPublicBuckets   bool     `yaml:"no_public_buckets,omitempty" toml:"no_public_buckets,omitempty"`
	RequireEncryption bool     `yaml:"require_encryption,omitempty" toml:"require_encryption,omitempty"`
	RequireTags       []string `yaml:"require_tags,omitempty" toml:"require_tags,omitempty"`
	MaxCostPerMonth   float64  `yaml:"max_cost_per_month,omitempty" toml:"max_cost_per_month,omitempty"`
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config

	// Detect format by extension
	switch filepath.Ext(path) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".toml":
		_, err = toml.Decode(string(data), &config)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TOML: %w", err)
		}
	default:
		// Try to auto-detect format
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			_, err = toml.Decode(string(data), &config)
			if err != nil {
				return nil, fmt.Errorf("failed to parse config (tried YAML and TOML): %w", err)
			}
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
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	case ".toml":
		buf := new(bytes.Buffer)
		err = toml.NewEncoder(buf).Encode(config)
		data = buf.Bytes()
	default:
		// Default to YAML
		data, err = yaml.Marshal(config)
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
