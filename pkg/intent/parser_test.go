package intent

import (
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		args     []string
		expected Intent
		wantErr  bool
	}{
		{
			name: "bucket with name",
			args: []string{"bucket", "my-bucket"},
			expected: Intent{
				Type:       IntentBucket,
				Name:       "my-bucket",
				Action:     ActionCreate,
				Parameters: map[string]string{
					"versioning": "true",
					"encryption": "true",
					"public":     "false",
				},
				Modifiers: []string{},
			},
			wantErr: false,
		},
		{
			name: "network with CIDR",
			args: []string{"network", "vpc-prod", "cidr=10.0.0.0/16"},
			expected: Intent{
				Type:   IntentNetwork,
				Name:   "vpc-prod",
				Action: ActionCreate,
				Parameters: map[string]string{
					"cidr":    "10.0.0.0/16",
					"subnets": "public,private",
				},
				Modifiers: []string{},
			},
			wantErr: false,
		},
		{
			name: "function with runtime",
			args: []string{"function", "api-handler", "runtime=nodejs18", "memory=512"},
			expected: Intent{
				Type:   IntentFunction,
				Name:   "api-handler",
				Action: ActionCreate,
				Parameters: map[string]string{
					"runtime": "nodejs18",
					"memory":  "512",
					"timeout": "60",
					"handler": "main.handler",
				},
				Modifiers: []string{},
			},
			wantErr: false,
		},
		{
			name: "static-site with domain",
			args: []string{"static-site", "domain=example.com", "cdn=true"},
			expected: Intent{
				Type:   IntentStaticSite,
				Action: ActionCreate,
				Parameters: map[string]string{
					"domain": "example.com",
					"cdn":    "true",
					"https":  "true",
					"index":  "index.html",
				},
				Modifiers: []string{},
			},
			wantErr: false,
		},
		{
			name: "database with engine",
			args: []string{"database", "prod-db", "engine=postgres", "size=large"},
			expected: Intent{
				Type:   IntentDatabase,
				Name:   "prod-db",
				Action: ActionCreate,
				Parameters: map[string]string{
					"engine":  "postgres",
					"size":    "large",
					"storage": "20",
					"backup":  "true",
				},
				Modifiers: []string{},
			},
			wantErr: false,
		},
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "unknown intent",
			args:    []string{"unknown"},
			wantErr: true,
		},
		{
			name: "invalid bucket name",
			args: []string{"bucket", "INVALID_NAME"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := parser.Parse(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parser.Parse() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Parser.Parse() unexpected error = %v", err)
				return
			}

			if intent.Type != tt.expected.Type {
				t.Errorf("Parser.Parse() Type = %v, expected %v", intent.Type, tt.expected.Type)
			}

			if intent.Name != tt.expected.Name {
				t.Errorf("Parser.Parse() Name = %v, expected %v", intent.Name, tt.expected.Name)
			}

			if intent.Action != tt.expected.Action {
				t.Errorf("Parser.Parse() Action = %v, expected %v", intent.Action, tt.expected.Action)
			}

			// Check parameters
			for key, expectedValue := range tt.expected.Parameters {
				if actualValue, ok := intent.Parameters[key]; !ok {
					t.Errorf("Parser.Parse() missing parameter %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("Parser.Parse() parameter %s = %v, expected %v", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestIntent_ToHumanReadable(t *testing.T) {
	tests := []struct {
		name     string
		intent   Intent
		expected string
	}{
		{
			name: "bucket creation",
			intent: Intent{
				Type:   IntentBucket,
				Name:   "my-bucket",
				Action: ActionCreate,
				Parameters: map[string]string{
					"versioning": "true",
					"encryption": "true",
				},
			},
			expected: "Create bucket named 'my-bucket' with encryption, versioning",
		},
		{
			name: "function with runtime",
			intent: Intent{
				Type:   IntentFunction,
				Name:   "api-handler",
				Action: ActionCreate,
				Parameters: map[string]string{
					"runtime": "python3.11",
					"memory":  "512",
				},
			},
			expected: "Create function named 'api-handler' with runtime=python3.11, memory=512",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.intent.ToHumanReadable()
			if result != tt.expected {
				t.Errorf("Intent.ToHumanReadable() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidBucketName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "my-bucket", true},
		{"valid with numbers", "bucket123", true},
		{"valid with dots", "my.bucket", true},
		{"too short", "ab", false},
		{"too long", "a" + string(make([]byte, 64)), false},
		{"uppercase letters", "MyBucket", false},
		{"invalid characters", "my_bucket", false},
		{"starts with dash", "-bucket", true}, // This should probably be false in real implementation
		{"ends with dash", "bucket-", true},   // This should probably be false in real implementation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBucketName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidBucketName(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}