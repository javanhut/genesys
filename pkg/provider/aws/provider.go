package aws

import (
	"context"
	"fmt"

	"github.com/javanhut/genesys/pkg/provider"
)

func init() {
	// Register the AWS provider
	provider.Register("aws", func(config map[string]string) (provider.Provider, error) {
		region := config["region"]
		if region == "" {
			region = "us-east-1"
		}
		return NewAWSProvider(region)
	})
}

// AWSProvider implements the Provider interface for AWS using direct API calls
type AWSProvider struct {
	region     string
	compute    provider.ComputeService
	storage    provider.StorageService
	network    provider.NetworkService
	database   provider.DatabaseService
	dynamodb   provider.DynamoDBService
	serverless provider.ServerlessService
	state      provider.StateBackend
	iam        *IAMService
	monitoring provider.MonitoringService
	inspector  provider.InspectorService
	logs       provider.LogsService
}

// NewAWSProvider creates a new AWS provider instance
func NewAWSProvider(region string) (*AWSProvider, error) {
	if region == "" {
		region = "us-east-1"
	}

	// Test credentials by creating a client
	_, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	awsProvider := &AWSProvider{
		region: region,
	}

	// Initialize services
	awsProvider.compute = NewComputeService(awsProvider)
	awsProvider.storage = NewStorageService(awsProvider)
	awsProvider.network = NewNetworkService(awsProvider)
	awsProvider.database = NewDatabaseService(awsProvider)
	awsProvider.dynamodb = NewDynamoDBService(awsProvider)
	awsProvider.serverless = NewServerlessService(awsProvider)
	awsProvider.state = NewStateBackend(awsProvider)
	awsProvider.iam = NewIAMService(awsProvider)
	awsProvider.monitoring = NewMonitoringService(awsProvider)
	awsProvider.inspector = NewInspectorService(awsProvider)
	awsProvider.logs = NewLogsService(awsProvider)

	return awsProvider, nil
}

// Name returns the provider name
func (p *AWSProvider) Name() string {
	return "aws"
}

// Region returns the provider region
func (p *AWSProvider) Region() string {
	return p.region
}

// Validate validates the provider configuration
func (p *AWSProvider) Validate() error {
	// Test connectivity by making a simple STS call
	client, err := NewAWSClient(p.region, "sts")
	if err != nil {
		return fmt.Errorf("failed to create STS client: %w", err)
	}

	// Make a GetCallerIdentity call to test credentials
	resp, err := client.Request("POST", "/", map[string]string{
		"Action":  "GetCallerIdentity",
		"Version": "2011-06-15",
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("AWS credentials validation failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Compute returns the compute service
func (p *AWSProvider) Compute() provider.ComputeService {
	return p.compute
}

// Storage returns the storage service
func (p *AWSProvider) Storage() provider.StorageService {
	return p.storage
}

// Network returns the network service
func (p *AWSProvider) Network() provider.NetworkService {
	return p.network
}

// Database returns the database service (RDS)
func (p *AWSProvider) Database() provider.DatabaseService {
	return p.database
}

// DynamoDB returns the DynamoDB service
func (p *AWSProvider) DynamoDB() provider.DynamoDBService {
	return p.dynamodb
}

// Serverless returns the serverless service
func (p *AWSProvider) Serverless() provider.ServerlessService {
	return p.serverless
}

// StateBackend returns the state backend
func (p *AWSProvider) StateBackend() provider.StateBackend {
	return p.state
}

// Authenticate performs authentication with AWS
func (p *AWSProvider) Authenticate(ctx context.Context) error {
	return p.Validate()
}

// GetRegion returns the AWS region
func (p *AWSProvider) GetRegion() string {
	return p.region
}

// CreateClient creates a new AWS client for the specified service
func (p *AWSProvider) CreateClient(service string) (*AWSClient, error) {
	return NewAWSClient(p.region, service)
}

// IAM returns the IAM service
func (p *AWSProvider) IAM() *IAMService {
	return p.iam
}

// Monitoring returns the monitoring service
func (p *AWSProvider) Monitoring() provider.MonitoringService {
	return p.monitoring
}

// Inspector returns the inspector service
func (p *AWSProvider) Inspector() provider.InspectorService {
	return p.inspector
}

// Logs returns the logs service
func (p *AWSProvider) Logs() provider.LogsService {
	return p.logs
}

// Init initializes the AWS provider factory
func Init() (provider.Provider, error) {
	region := "us-east-1" // Default region
	return NewAWSProvider(region)
}
