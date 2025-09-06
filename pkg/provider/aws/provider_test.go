package aws

import (
	"testing"
)

func TestNewAWSProvider(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{
			name:   "default region",
			region: "",
			want:   "us-east-1",
		},
		{
			name:   "specified region",
			region: "us-west-2",
			want:   "us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test if no AWS credentials are available
			t.Skip("Skipping AWS provider test - requires AWS credentials")

			provider, err := NewAWSProvider(tt.region)
			if err != nil {
				t.Fatalf("NewAWSProvider() error = %v", err)
			}

			if provider.Name() != "aws" {
				t.Errorf("Name() = %v, want aws", provider.Name())
			}

			if provider.Region() != tt.want {
				t.Errorf("Region() = %v, want %v", provider.Region(), tt.want)
			}
		})
	}
}

func TestAWSClient(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		service string
	}{
		{
			name:    "ec2 client",
			region:  "us-east-1",
			service: "ec2",
		},
		{
			name:    "s3 client",
			region:  "us-west-2",
			service: "s3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test if no AWS credentials are available
			t.Skip("Skipping AWS client test - requires AWS credentials")

			client, err := NewAWSClient(tt.region, tt.service)
			if err != nil {
				t.Fatalf("NewAWSClient() error = %v", err)
			}

			if client.Region != tt.region {
				t.Errorf("Region = %v, want %v", client.Region, tt.region)
			}

			if client.Service != tt.service {
				t.Errorf("Service = %v, want %v", client.Service, tt.service)
			}
		})
	}
}

func TestInstanceTypeMapping(t *testing.T) {
	provider := &AWSProvider{region: "us-east-1"}
	compute := NewComputeService(provider)

	tests := []struct {
		input string
		want  string
	}{
		{"small", "t3.small"},
		{"medium", "t3.medium"},
		{"large", "t3.large"},
		{"xlarge", "t3.xlarge"},
		{"unknown", "t3.medium"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := compute.mapInstanceType(tt.input)
			if got != tt.want {
				t.Errorf("mapInstanceType(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDatabaseSizeMapping(t *testing.T) {
	provider := &AWSProvider{region: "us-east-1"}
	database := NewDatabaseService(provider)

	tests := []struct {
		input string
		want  string
	}{
		{"small", "db.t3.micro"},
		{"medium", "db.t3.small"},
		{"large", "db.t3.medium"},
		{"xlarge", "db.t3.large"},
		{"unknown", "db.t3.small"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := database.mapDatabaseSize(tt.input)
			if got != tt.want {
				t.Errorf("mapDatabaseSize(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}