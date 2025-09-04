package config

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Provider: "aws",
				Region:   "us-east-1",
				Resources: Resources{
					Compute: []ComputeResource{
						{Name: "web-server", Type: "medium", Count: 2},
					},
					Database: []DatabaseResource{
						{Name: "app-db", Engine: "postgres", Size: "medium", Storage: 100},
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			config: &Config{
				Provider: "invalid-provider",
			},
			wantError: true,
			errorMsg:  "invalid provider",
		},
		{
			name: "compute resource without name",
			config: &Config{
				Provider: "aws",
				Resources: Resources{
					Compute: []ComputeResource{
						{Type: "medium", Count: 1}, // Missing name
					},
				},
			},
			wantError: true,
			errorMsg:  "must have a name",
		},
		{
			name: "compute resource with invalid type",
			config: &Config{
				Provider: "aws",
				Resources: Resources{
					Compute: []ComputeResource{
						{Name: "server", Type: "invalid", Count: 1},
					},
				},
			},
			wantError: true,
			errorMsg:  "invalid type",
		},
		{
			name: "database without engine",
			config: &Config{
				Provider: "aws",
				Resources: Resources{
					Database: []DatabaseResource{
						{Name: "db", Size: "small", Storage: 20}, // Missing engine
					},
				},
			},
			wantError: true,
			errorMsg:  "must have an engine",
		},
		{
			name: "serverless with invalid runtime",
			config: &Config{
				Provider: "aws",
				Resources: Resources{
					Serverless: []ServerlessResource{
						{Name: "func", Runtime: "invalid-runtime", Memory: 256, Timeout: 60},
					},
				},
			},
			wantError: true,
			errorMsg:  "invalid runtime",
		},
		{
			name: "serverless with memory out of range",
			config: &Config{
				Provider: "aws",
				Resources: Resources{
					Serverless: []ServerlessResource{
						{Name: "func", Runtime: "python3.11", Memory: 50, Timeout: 60}, // Memory too low
					},
				},
			},
			wantError: true,
			errorMsg:  "memory must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateConfig() expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateConfig() error = %v, expected to contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateComputeResource(t *testing.T) {
	tests := []struct {
		name     string
		compute  ComputeResource
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "valid compute resource",
			compute: ComputeResource{Name: "web-server", Type: "medium", Count: 2},
			wantErr: false,
		},
		{
			name:     "missing name",
			compute:  ComputeResource{Type: "medium", Count: 1},
			wantErr:  true,
			errorMsg: "must have a name",
		},
		{
			name:     "invalid type",
			compute:  ComputeResource{Name: "server", Type: "huge", Count: 1},
			wantErr:  true,
			errorMsg: "invalid type",
		},
		{
			name:     "zero count",
			compute:  ComputeResource{Name: "server", Type: "small", Count: 0},
			wantErr:  true,
			errorMsg: "must have count >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateComputeResource(&tt.compute, 0)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateComputeResource() expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateComputeResource() error = %v, expected to contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateComputeResource() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateDatabaseResource(t *testing.T) {
	tests := []struct {
		name     string
		database DatabaseResource
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "valid database resource",
			database: DatabaseResource{Name: "app-db", Engine: "postgres", Size: "medium", Storage: 100},
			wantErr:  false,
		},
		{
			name:     "missing engine",
			database: DatabaseResource{Name: "db", Size: "small", Storage: 20},
			wantErr:  true,
			errorMsg: "must have an engine",
		},
		{
			name:     "invalid engine",
			database: DatabaseResource{Name: "db", Engine: "invalid-engine", Size: "small", Storage: 20},
			wantErr:  true,
			errorMsg: "invalid engine",
		},
		{
			name:     "storage too small",
			database: DatabaseResource{Name: "db", Engine: "postgres", Size: "small", Storage: 5},
			wantErr:  true,
			errorMsg: "must have storage >= 10 GB",
		},
		{
			name: "invalid backup retention",
			database: DatabaseResource{
				Name:    "db",
				Engine:  "postgres",
				Size:    "small",
				Storage: 20,
				Backup: &BackupConfig{
					RetentionDays: 400, // Too long
				},
			},
			wantErr:  true,
			errorMsg: "backup retention must be between 1-365 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseResource(&tt.database, 0)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateDatabaseResource() expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateDatabaseResource() error = %v, expected to contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateDatabaseResource() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateServerlessResource(t *testing.T) {
	tests := []struct {
		name       string
		serverless ServerlessResource
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "valid serverless resource",
			serverless: ServerlessResource{
				Name:    "api-handler",
				Runtime: "python3.11",
				Memory:  512,
				Timeout: 30,
			},
			wantErr: false,
		},
		{
			name: "invalid runtime",
			serverless: ServerlessResource{
				Name:    "func",
				Runtime: "cobol",
				Memory:  256,
				Timeout: 60,
			},
			wantErr:  true,
			errorMsg: "invalid runtime",
		},
		{
			name: "memory too high",
			serverless: ServerlessResource{
				Name:    "func",
				Runtime: "python3.11",
				Memory:  15000,
				Timeout: 60,
			},
			wantErr:  true,
			errorMsg: "memory must be between",
		},
		{
			name: "timeout too long",
			serverless: ServerlessResource{
				Name:    "func",
				Runtime: "python3.11",
				Memory:  256,
				Timeout: 1000,
			},
			wantErr:  true,
			errorMsg: "timeout must be between",
		},
		{
			name: "invalid trigger type",
			serverless: ServerlessResource{
				Name:    "func",
				Runtime: "python3.11",
				Memory:  256,
				Timeout: 60,
				Triggers: []TriggerConfig{
					{Type: "invalid-trigger"},
				},
			},
			wantErr:  true,
			errorMsg: "invalid type",
		},
		{
			name: "http trigger without path",
			serverless: ServerlessResource{
				Name:    "func",
				Runtime: "python3.11",
				Memory:  256,
				Timeout: 60,
				Triggers: []TriggerConfig{
					{Type: "http", Methods: []string{"GET"}}, // Missing path
				},
			},
			wantErr:  true,
			errorMsg: "must have a path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerlessResource(&tt.serverless, 0)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateServerlessResource() expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateServerlessResource() error = %v, expected to contain %v", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateServerlessResource() unexpected error = %v", err)
				}
			}
		})
	}
}

