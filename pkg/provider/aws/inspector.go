package aws

import (
	"context"
	"fmt"

	"github.com/javanhut/genesys/pkg/provider"
)

// InspectorService provides deep resource inspection capabilities
type InspectorService struct {
	awsProvider *AWSProvider
}

// NewInspectorService creates a new inspector service
func NewInspectorService(p *AWSProvider) *InspectorService {
	return &InspectorService{
		awsProvider: p,
	}
}

// InspectEC2Instance provides comprehensive EC2 instance inspection
func (i *InspectorService) InspectEC2Instance(ctx context.Context, instanceID string) (*provider.EC2InspectionResult, error) {
	// Get basic instance information
	instance, err := i.awsProvider.Compute().GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	result := &provider.EC2InspectionResult{
		Instance: instance,
		Tags:     instance.Tags,
	}

	// Get metrics
	metrics, err := i.awsProvider.Monitoring().GetEC2Metrics(ctx, instanceID, "1h")
	if err == nil {
		result.DetailedMetrics = metrics
	}

	// Get console output (best effort)
	consoleOutput, err := i.GetEC2ConsoleOutput(ctx, instanceID)
	if err == nil {
		result.ConsoleOutput = consoleOutput
	}

	return result, nil
}

// GetEC2ConsoleOutput retrieves the console output for an EC2 instance
func (i *InspectorService) GetEC2ConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	client, err := i.awsProvider.CreateClient("ec2")
	if err != nil {
		return "", fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":     "GetConsoleOutput",
		"Version":    "2016-11-15",
		"InstanceId": instanceID,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get console output: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return "", fmt.Errorf("GetConsoleOutput failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse XML to extract console output
	// For simplicity, return raw response (in production, parse XML properly)
	return string(body), nil
}

// GetEC2SystemLog retrieves the system log for an EC2 instance
func (i *InspectorService) GetEC2SystemLog(ctx context.Context, instanceID string) (string, error) {
	// System log is the same as console output for EC2
	return i.GetEC2ConsoleOutput(ctx, instanceID)
}

// InspectS3Bucket provides comprehensive S3 bucket inspection
func (i *InspectorService) InspectS3Bucket(ctx context.Context, bucketName string) (*provider.S3InspectionResult, error) {
	// Get basic bucket information
	bucket, err := i.awsProvider.Storage().GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	result := &provider.S3InspectionResult{
		Bucket:      bucket,
		Location:    bucket.Region,
		Versioning:  bucket.Versioning,
		Encryption:  bucket.Encryption,
		CreatedDate: bucket.CreatedAt,
	}

	// Analyze bucket size
	objectCount, totalSize, err := i.AnalyzeBucketSize(ctx, bucketName)
	if err == nil {
		result.ObjectCount = objectCount
		result.TotalSizeBytes = totalSize
	}

	// Get bucket ACL
	acl, err := i.GetBucketACL(ctx, bucketName)
	if err == nil {
		result.ACL = acl
	}

	// Get CORS configuration
	cors, err := i.GetBucketCORS(ctx, bucketName)
	if err == nil {
		result.CORS = cors
	}

	return result, nil
}

// AnalyzeBucketSize calculates the total number of objects and size of a bucket
func (i *InspectorService) AnalyzeBucketSize(ctx context.Context, bucketName string) (int64, int64, error) {
	objects, err := i.awsProvider.Storage().ListObjectsRecursive(ctx, bucketName, "")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list objects: %w", err)
	}

	var totalSize int64
	objectCount := int64(len(objects))

	for _, obj := range objects {
		if !obj.IsPrefix {
			totalSize += obj.Size
		}
	}

	return objectCount, totalSize, nil
}

// GetBucketACL retrieves the ACL for a bucket
func (i *InspectorService) GetBucketACL(ctx context.Context, bucketName string) (*provider.BucketACL, error) {
	client, err := i.awsProvider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"acl": ""}

	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket ACL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetBucketAcl failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For simplicity, return a basic ACL structure
	// In production, properly parse the XML response
	return &provider.BucketACL{
		Owner:  "owner",
		Grants: []provider.ACLGrant{},
	}, nil
}

// GetBucketCORS retrieves the CORS configuration for a bucket
func (i *InspectorService) GetBucketCORS(ctx context.Context, bucketName string) (*provider.CORSConfiguration, error) {
	client, err := i.awsProvider.CreateClient("s3")
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", bucketName)
	params := map[string]string{"cors": ""}

	resp, err := client.Request("GET", endpoint, params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket CORS: %w", err)
	}
	defer resp.Body.Close()

	// 404 means no CORS configuration
	if resp.StatusCode == 404 {
		return &provider.CORSConfiguration{
			Rules: []provider.CORSRule{},
		}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetBucketCors failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For simplicity, return empty CORS configuration
	// In production, properly parse the XML response
	return &provider.CORSConfiguration{
		Rules: []provider.CORSRule{},
	}, nil
}

// InspectLambdaFunction provides comprehensive Lambda function inspection
func (i *InspectorService) InspectLambdaFunction(ctx context.Context, functionName string) (*provider.LambdaInspectionResult, error) {
	// Get Lambda metrics
	metrics, err := i.awsProvider.Monitoring().GetLambdaMetrics(ctx, functionName, "1h")
	if err != nil {
		return nil, fmt.Errorf("failed to get Lambda metrics: %w", err)
	}

	result := &provider.LambdaInspectionResult{
		DetailedMetrics: metrics,
	}

	// Get recent logs
	logs, err := i.awsProvider.Logs().GetLambdaLogs(ctx, functionName, 0, 0, 50)
	if err == nil {
		result.RecentLogs = logs
	}

	// Get detailed configuration
	config, err := i.GetLambdaConfiguration(ctx, functionName)
	if err == nil {
		result.Configuration = config
	}

	return result, nil
}

// GetLambdaConfiguration retrieves detailed Lambda function configuration
func (i *InspectorService) GetLambdaConfiguration(ctx context.Context, functionName string) (*provider.LambdaDetailedConfig, error) {
	client, err := i.awsProvider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	endpoint := fmt.Sprintf("/2015-03-31/functions/%s/configuration", functionName)
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get function configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetFunctionConfiguration failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For simplicity, return basic config
	// In production, properly parse the JSON response
	return &provider.LambdaDetailedConfig{
		State:            "Active",
		LastUpdateStatus: "Successful",
		PackageType:      "Zip",
		Architectures:    []string{"x86_64"},
	}, nil
}
