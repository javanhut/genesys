package config

// ApplyDefaults sets default values for configuration
func ApplyDefaults(config *Config) {
	// Provider defaults
	if config.Provider == "" {
		config.Provider = "aws"
	}

	// Region defaults based on provider
	if config.Region == "" {
		switch config.Provider {
		case "aws":
			config.Region = "us-east-1"
		case "gcp":
			config.Region = "us-central1"
		case "azure":
			config.Region = "eastus"
		default:
			config.Region = "us-east-1"
		}
	}

	// Apply defaults to resources
	applyComputeDefaults(config)
	applyStorageDefaults(config)
	applyDatabaseDefaults(config)
	applyServerlessDefaults(config)

	// Apply state defaults
	applyStateDefaults(config)

	// Apply policy defaults
	applyPolicyDefaults(config)
}

// applyComputeDefaults applies defaults to compute resources
func applyComputeDefaults(config *Config) {
	for i := range config.Resources.Compute {
		compute := &config.Resources.Compute[i]

		if compute.Count == 0 {
			compute.Count = 1
		}

		if compute.Type == "" {
			compute.Type = "small"
		}

		if compute.Image == "" {
			compute.Image = "ubuntu-lts"
		}

		if compute.Tags == nil {
			compute.Tags = make(map[string]string)
		}
	}
}

// applyStorageDefaults applies defaults to storage resources
func applyStorageDefaults(config *Config) {
	for i := range config.Resources.Storage {
		storage := &config.Resources.Storage[i]

		if storage.Type == "" {
			storage.Type = "bucket"
		}

		// Default to secure settings
		if !storage.Versioning && storage.Type == "bucket" {
			storage.Versioning = true
		}

		if !storage.Encryption {
			storage.Encryption = true
		}

		if storage.Tags == nil {
			storage.Tags = make(map[string]string)
		}
	}
}

// applyDatabaseDefaults applies defaults to database resources
func applyDatabaseDefaults(config *Config) {
	for i := range config.Resources.Database {
		database := &config.Resources.Database[i]

		if database.Engine == "" {
			database.Engine = "postgres"
		}

		if database.Version == "" {
			switch database.Engine {
			case "postgres":
				database.Version = "15"
			case "mysql":
				database.Version = "8.0"
			case "mariadb":
				database.Version = "10.11"
			default:
				database.Version = "latest"
			}
		}

		if database.Size == "" {
			database.Size = "small"
		}

		if database.Storage == 0 {
			database.Storage = 20
		}

		// Default backup configuration
		if database.Backup == nil {
			database.Backup = &BackupConfig{
				RetentionDays: 7,
				Window:        "03:00-04:00",
			}
		}

		if database.Tags == nil {
			database.Tags = make(map[string]string)
		}
	}
}

// applyServerlessDefaults applies defaults to serverless resources
func applyServerlessDefaults(config *Config) {
	for i := range config.Resources.Serverless {
		serverless := &config.Resources.Serverless[i]

		if serverless.Runtime == "" {
			serverless.Runtime = "python3.11"
		}

		if serverless.Handler == "" {
			serverless.Handler = "main.handler"
		}

		if serverless.Memory == 0 {
			serverless.Memory = 256
		}

		if serverless.Timeout == 0 {
			serverless.Timeout = 60
		}

		if serverless.Environment == nil {
			serverless.Environment = make(map[string]string)
		}

		if serverless.Tags == nil {
			serverless.Tags = make(map[string]string)
		}
	}
}

// applyStateDefaults applies defaults to state configuration
func applyStateDefaults(config *Config) {
	if config.State.Backend == "" {
		switch config.Provider {
		case "aws":
			config.State.Backend = "s3"
		case "gcp":
			config.State.Backend = "gcs"
		case "azure":
			config.State.Backend = "azureblob"
		default:
			config.State.Backend = "local"
		}
	}

	// Default to encryption for security
	if !config.State.Encrypt {
		config.State.Encrypt = true
	}
}

// applyPolicyDefaults applies defaults to policy configuration
func applyPolicyDefaults(config *Config) {
	// Default to secure policies
	if !config.Policies.NoPublicBuckets {
		config.Policies.NoPublicBuckets = true
	}

	if !config.Policies.RequireEncryption {
		config.Policies.RequireEncryption = true
	}

}
