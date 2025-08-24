package config

import (
	"fmt"
)

// Validator interface for configuration validation
type Validator interface {
	Validate() error
}

// ValidateConfig performs comprehensive configuration validation
func ValidateConfig(config *Config) error {
	if err := validateProvider(config); err != nil {
		return err
	}

	if err := validateResources(config); err != nil {
		return err
	}

	if err := validatePolicies(config); err != nil {
		return err
	}

	return nil
}

// validateProvider validates provider-specific configuration
func validateProvider(config *Config) error {
	validProviders := []string{"aws", "gcp", "azure", "alibaba", "tencent"}
	
	for _, p := range validProviders {
		if config.Provider == p {
			return nil
		}
	}
	
	return fmt.Errorf("invalid provider: %s, must be one of: %v", config.Provider, validProviders)
}

// validateResources validates all resource configurations
func validateResources(config *Config) error {
	// Validate compute resources
	for i, compute := range config.Resources.Compute {
		if err := validateComputeResource(&compute, i); err != nil {
			return err
		}
	}

	// Validate storage resources
	for i, storage := range config.Resources.Storage {
		if err := validateStorageResource(&storage, i); err != nil {
			return err
		}
	}

	// Validate database resources
	for i, database := range config.Resources.Database {
		if err := validateDatabaseResource(&database, i); err != nil {
			return err
		}
	}

	// Validate serverless resources
	for i, serverless := range config.Resources.Serverless {
		if err := validateServerlessResource(&serverless, i); err != nil {
			return err
		}
	}

	return nil
}

// validateComputeResource validates a compute resource configuration
func validateComputeResource(compute *ComputeResource, index int) error {
	if compute.Name == "" {
		return fmt.Errorf("compute resource at index %d must have a name", index)
	}

	validTypes := []string{"small", "medium", "large", "xlarge"}
	if !contains(validTypes, compute.Type) {
		return fmt.Errorf("compute resource '%s' has invalid type: %s, must be one of: %v", 
			compute.Name, compute.Type, validTypes)
	}

	if compute.Count < 1 {
		return fmt.Errorf("compute resource '%s' must have count >= 1", compute.Name)
	}

	return nil
}

// validateStorageResource validates a storage resource configuration
func validateStorageResource(storage *StorageResource, index int) error {
	if storage.Name == "" {
		return fmt.Errorf("storage resource at index %d must have a name", index)
	}

	validTypes := []string{"bucket", "volume"}
	if !contains(validTypes, storage.Type) {
		return fmt.Errorf("storage resource '%s' has invalid type: %s, must be one of: %v", 
			storage.Name, storage.Type, validTypes)
	}

	// Validate lifecycle configuration if present
	if storage.Lifecycle != nil {
		if storage.Lifecycle.DeleteAfterDays < 0 {
			return fmt.Errorf("storage resource '%s' has negative delete_after_days", storage.Name)
		}
		if storage.Lifecycle.ArchiveAfterDays < 0 {
			return fmt.Errorf("storage resource '%s' has negative archive_after_days", storage.Name)
		}
	}

	return nil
}

// validateDatabaseResource validates a database resource configuration
func validateDatabaseResource(database *DatabaseResource, index int) error {
	if database.Name == "" {
		return fmt.Errorf("database resource at index %d must have a name", index)
	}

	if database.Engine == "" {
		return fmt.Errorf("database resource '%s' must have an engine", database.Name)
	}

	validEngines := []string{"postgres", "mysql", "mariadb", "mongodb"}
	if !contains(validEngines, database.Engine) {
		return fmt.Errorf("database resource '%s' has invalid engine: %s, must be one of: %v", 
			database.Name, database.Engine, validEngines)
	}

	validSizes := []string{"small", "medium", "large"}
	if !contains(validSizes, database.Size) {
		return fmt.Errorf("database resource '%s' has invalid size: %s, must be one of: %v", 
			database.Name, database.Size, validSizes)
	}

	if database.Storage < 10 {
		return fmt.Errorf("database resource '%s' must have storage >= 10 GB", database.Name)
	}

	// Validate backup configuration if present
	if database.Backup != nil {
		if database.Backup.RetentionDays < 1 || database.Backup.RetentionDays > 365 {
			return fmt.Errorf("database resource '%s' backup retention must be between 1-365 days", database.Name)
		}
	}

	return nil
}

// validateServerlessResource validates a serverless resource configuration
func validateServerlessResource(serverless *ServerlessResource, index int) error {
	if serverless.Name == "" {
		return fmt.Errorf("serverless resource at index %d must have a name", index)
	}

	if serverless.Runtime == "" {
		return fmt.Errorf("serverless resource '%s' must have a runtime", serverless.Name)
	}

	validRuntimes := []string{"python3.9", "python3.10", "python3.11", "nodejs16", "nodejs18", "nodejs20", "go1.19", "go1.20", "go1.21", "java11", "java17"}
	if !contains(validRuntimes, serverless.Runtime) {
		return fmt.Errorf("serverless resource '%s' has invalid runtime: %s, must be one of: %v", 
			serverless.Name, serverless.Runtime, validRuntimes)
	}

	if serverless.Memory < 128 || serverless.Memory > 10240 {
		return fmt.Errorf("serverless resource '%s' memory must be between 128-10240 MB", serverless.Name)
	}

	if serverless.Timeout < 1 || serverless.Timeout > 900 {
		return fmt.Errorf("serverless resource '%s' timeout must be between 1-900 seconds", serverless.Name)
	}

	// Validate triggers
	for i, trigger := range serverless.Triggers {
		if err := validateTrigger(&trigger, serverless.Name, i); err != nil {
			return err
		}
	}

	return nil
}

// validateTrigger validates a trigger configuration
func validateTrigger(trigger *TriggerConfig, functionName string, index int) error {
	validTypes := []string{"http", "schedule", "queue", "storage"}
	if !contains(validTypes, trigger.Type) {
		return fmt.Errorf("function '%s' trigger %d has invalid type: %s, must be one of: %v", 
			functionName, index, trigger.Type, validTypes)
	}

	// Type-specific validation
	switch trigger.Type {
	case "http":
		if trigger.Path == "" {
			return fmt.Errorf("function '%s' HTTP trigger %d must have a path", functionName, index)
		}
		if len(trigger.Methods) == 0 {
			return fmt.Errorf("function '%s' HTTP trigger %d must have at least one method", functionName, index)
		}
	case "schedule":
		if trigger.Schedule == "" {
			return fmt.Errorf("function '%s' schedule trigger %d must have a schedule", functionName, index)
		}
	}

	return nil
}

// validatePolicies validates policy configuration
func validatePolicies(config *Config) error {
	if config.Policies.MaxCostPerMonth < 0 {
		return fmt.Errorf("max_cost_per_month cannot be negative")
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}