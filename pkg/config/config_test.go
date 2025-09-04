package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_YAML(t *testing.T) {
	// Create temporary YAML file
	yamlContent := `
provider: aws
region: us-east-1

resources:
  compute:
    - name: test-server
      type: medium
      count: 2
  
  storage:
    - name: test-bucket
      type: bucket
      versioning: true

policies:
  require_encryption: true
  max_cost_per_month: 100
`

	tempFile := filepath.Join(t.TempDir(), "test.yaml")
	err := os.WriteFile(tempFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Test basic fields
	if config.Provider != "aws" {
		t.Errorf("Expected provider 'aws', got '%s'", config.Provider)
	}

	if config.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", config.Region)
	}

	// Test compute resources
	if len(config.Resources.Compute) != 1 {
		t.Errorf("Expected 1 compute resource, got %d", len(config.Resources.Compute))
	}

	compute := config.Resources.Compute[0]
	if compute.Name != "test-server" {
		t.Errorf("Expected compute name 'test-server', got '%s'", compute.Name)
	}

	if compute.Count != 2 {
		t.Errorf("Expected compute count 2, got %d", compute.Count)
	}

	// Test storage resources
	if len(config.Resources.Storage) != 1 {
		t.Errorf("Expected 1 storage resource, got %d", len(config.Resources.Storage))
	}

	storage := config.Resources.Storage[0]
	if storage.Name != "test-bucket" {
		t.Errorf("Expected storage name 'test-bucket', got '%s'", storage.Name)
	}

	// Test policies
	if !config.Policies.RequireEncryption {
		t.Error("Expected require_encryption to be true")
	}

	if config.Policies.MaxCostPerMonth != 100 {
		t.Errorf("Expected max_cost_per_month 100, got %f", config.Policies.MaxCostPerMonth)
	}
}

func TestLoadConfig_TOML(t *testing.T) {
	tomlContent := `
provider = "gcp"
region = "us-central1"

[[resources.database]]
name = "test-db"
engine = "postgres"
version = "15"
size = "large"
storage = 500

[policies]
no_public_buckets = true
require_tags = ["Environment", "Team"]
`

	tempFile := filepath.Join(t.TempDir(), "test.toml")
	err := os.WriteFile(tempFile, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.Provider != "gcp" {
		t.Errorf("Expected provider 'gcp', got '%s'", config.Provider)
	}

	if len(config.Resources.Database) != 1 {
		t.Errorf("Expected 1 database resource, got %d", len(config.Resources.Database))
	}

	db := config.Resources.Database[0]
	if db.Engine != "postgres" {
		t.Errorf("Expected database engine 'postgres', got '%s'", db.Engine)
	}

	if db.Storage != 500 {
		t.Errorf("Expected database storage 500, got %d", db.Storage)
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &Config{
		Resources: Resources{
			Compute: []ComputeResource{
				{Name: "test-server"}, // Missing defaults
			},
			Storage: []StorageResource{
				{Name: "test-bucket"}, // Missing defaults
			},
		},
	}

	ApplyDefaults(config)

	// Test provider default
	if config.Provider != "aws" {
		t.Errorf("Expected default provider 'aws', got '%s'", config.Provider)
	}

	// Test region default
	if config.Region != "us-east-1" {
		t.Errorf("Expected default region 'us-east-1', got '%s'", config.Region)
	}

	// Test compute defaults
	compute := config.Resources.Compute[0]
	if compute.Count != 1 {
		t.Errorf("Expected default count 1, got %d", compute.Count)
	}

	if compute.Type != "small" {
		t.Errorf("Expected default type 'small', got '%s'", compute.Type)
	}

	if compute.Image != "ubuntu-lts" {
		t.Errorf("Expected default image 'ubuntu-lts', got '%s'", compute.Image)
	}

	// Test storage defaults
	storage := config.Resources.Storage[0]
	if !storage.Encryption {
		t.Error("Expected default encryption to be true")
	}

	if storage.Type != "bucket" {
		t.Errorf("Expected default storage type 'bucket', got '%s'", storage.Type)
	}

	// Test state defaults
	if config.State.Backend != "s3" {
		t.Errorf("Expected default state backend 's3', got '%s'", config.State.Backend)
	}

	if !config.State.Encrypt {
		t.Error("Expected default state encryption to be true")
	}

	// Test policy defaults
	if !config.Policies.NoPublicBuckets {
		t.Error("Expected default no_public_buckets to be true")
	}

	if !config.Policies.RequireEncryption {
		t.Error("Expected default require_encryption to be true")
	}
}

func TestSaveConfig_YAML(t *testing.T) {
	config := &Config{
		Provider: "aws",
		Region:   "us-east-1",
		Resources: Resources{
			Compute: []ComputeResource{
				{
					Name:  "test-server",
					Type:  "medium",
					Count: 2,
					Tags:  map[string]string{"Environment": "test"},
				},
			},
		},
		Policies: Policies{
			RequireEncryption: true,
			MaxCostPerMonth:   100,
		},
	}

	tempFile := filepath.Join(t.TempDir(), "output.yaml")
	err := SaveConfig(config, tempFile)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Provider != config.Provider {
		t.Errorf("Saved provider mismatch: expected '%s', got '%s'", config.Provider, loadedConfig.Provider)
	}

	if len(loadedConfig.Resources.Compute) != 1 {
		t.Errorf("Expected 1 compute resource, got %d", len(loadedConfig.Resources.Compute))
	}
}