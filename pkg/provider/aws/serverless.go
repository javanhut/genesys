package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// ServerlessService implements AWS Lambda operations using direct API calls
type ServerlessService struct {
	provider *AWSProvider
}

// NewServerlessService creates a new serverless service
func NewServerlessService(p *AWSProvider) *ServerlessService {
	return &ServerlessService{
		provider: p,
	}
}

// Lambda API response structures
type LambdaFunction struct {
	FunctionName string       `json:"FunctionName"`
	FunctionArn  string       `json:"FunctionArn"`
	Runtime      string       `json:"Runtime"`
	Handler      string       `json:"Handler"`
	CodeSize     int64        `json:"CodeSize"`
	Description  string       `json:"Description"`
	Timeout      int          `json:"Timeout"`
	MemorySize   int          `json:"MemorySize"`
	LastModified string       `json:"LastModified"`
	State        string       `json:"State"`
	Environment  *Environment `json:"Environment,omitempty"`
}

type Environment struct {
	Variables map[string]string `json:"Variables,omitempty"`
}

type ListFunctionsResponse struct {
	Functions  []LambdaFunction `json:"Functions"`
	NextMarker string           `json:"NextMarker,omitempty"`
}

// CreateFunction creates a new Lambda function
func (s *ServerlessService) CreateFunction(ctx context.Context, config *provider.FunctionConfig) (*provider.Function, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Build the request body
	requestBody := map[string]interface{}{
		"FunctionName": config.Name,
		"Runtime":      config.Runtime,
		"Handler":      config.Handler,
		"MemorySize":   config.Memory,
		"Timeout":      config.Timeout,
	}

	// Role must be provided by the caller
	if config.Role == "" {
		return nil, fmt.Errorf("IAM role ARN is required")
	}
	requestBody["Role"] = config.Role

	// Handle code upload
	if config.Code.LocalPath != "" {
		// Read ZIP file from local path
		zipData, err := os.ReadFile(config.Code.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read function ZIP: %w", err)
		}
		requestBody["Code"] = map[string]interface{}{
			"ZipFile": base64.StdEncoding.EncodeToString(zipData),
		}
	} else if config.Code.S3Bucket != "" && config.Code.S3Key != "" {
		// Use S3 location
		requestBody["Code"] = map[string]interface{}{
			"S3Bucket": config.Code.S3Bucket,
			"S3Key":    config.Code.S3Key,
		}
	} else if len(config.Code.ZipFile) > 0 {
		// Use provided ZIP data
		requestBody["Code"] = map[string]interface{}{
			"ZipFile": base64.StdEncoding.EncodeToString(config.Code.ZipFile),
		}
	} else {
		// Use dummy function for testing
		requestBody["Code"] = map[string]interface{}{
			"ZipFile": s.createDummyZip(),
		}
	}

	if config.Environment != nil && len(config.Environment) > 0 {
		requestBody["Environment"] = map[string]interface{}{
			"Variables": config.Environment,
		}
	}

	// Add layers if specified
	if len(config.Code.Layers) > 0 {
		requestBody["Layers"] = config.Code.Layers
	}

	// Convert to JSON
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Make the request with retry for role assumption issues
	var resp *http.Response
	var lastErr error
	
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Request("POST", "/2015-03-31/functions", nil, body)
		if err != nil {
			lastErr = fmt.Errorf("failed to create function: %w", err)
			continue
		}
		
		if resp.StatusCode == 201 {
			break // Success
		}
		
		// Check if it's a role assumption error
		responseBody, _ := ReadResponse(resp)
		responseStr := string(responseBody)
		
		if resp.StatusCode == 400 && strings.Contains(responseStr, "cannot be assumed by Lambda") {
			if attempt < maxRetries-1 {
				// Wait before retrying (IAM propagation delay)
				time.Sleep(time.Duration(2*(attempt+1)) * time.Second)
				resp.Body.Close()
				continue
			}
		}
		
		resp.Body.Close()
		return nil, fmt.Errorf("CreateFunction failed with status %d: %s", resp.StatusCode, responseStr)
	}
	
	if resp == nil {
		return nil, fmt.Errorf("failed to create function after %d attempts: %v", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	// Parse response
	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var lambdaFunc LambdaFunction
	if err := json.Unmarshal(responseBody, &lambdaFunc); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to provider function
	return s.convertToProviderFunction(lambdaFunc, config.Tags), nil
}

// UpdateFunction updates a Lambda function
func (s *ServerlessService) UpdateFunction(ctx context.Context, id string, config *provider.FunctionConfig) error {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Build the request body for configuration update
	requestBody := map[string]interface{}{}

	if config.Runtime != "" {
		requestBody["Runtime"] = config.Runtime
	}

	if config.Handler != "" {
		requestBody["Handler"] = config.Handler
	}

	if config.Memory > 0 {
		requestBody["MemorySize"] = config.Memory
	}

	if config.Timeout > 0 {
		requestBody["Timeout"] = config.Timeout
	}

	if config.Environment != nil {
		requestBody["Environment"] = map[string]interface{}{
			"Variables": config.Environment,
		}
	}

	// Convert to JSON
	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Make the request
	endpoint := fmt.Sprintf("/2015-03-31/functions/%s/configuration", id)
	resp, err := client.Request("PUT", endpoint, nil, body)
	if err != nil {
		return fmt.Errorf("failed to update function: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("UpdateFunctionConfiguration failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// DeleteFunction deletes a Lambda function
func (s *ServerlessService) DeleteFunction(ctx context.Context, id string) error {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return fmt.Errorf("failed to create Lambda client: %w", err)
	}

	endpoint := fmt.Sprintf("/2015-03-31/functions/%s", id)
	resp, err := client.Request("DELETE", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("DeleteFunction failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

// InvokeFunction invokes a Lambda function
func (s *ServerlessService) InvokeFunction(ctx context.Context, id string, payload []byte) ([]byte, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	endpoint := fmt.Sprintf("/2015-03-31/functions/%s/invocations", id)
	resp, err := client.Request("POST", endpoint, nil, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke function: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("InvokeFunction failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// DiscoverFunctions discovers existing Lambda functions
func (s *ServerlessService) DiscoverFunctions(ctx context.Context) ([]*provider.Function, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	resp, err := client.Request("GET", "/2015-03-31/functions", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("ListFunctions failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResp ListFunctionsResponse
	if err := json.Unmarshal(responseBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var functions []*provider.Function
	for _, lambdaFunc := range listResp.Functions {
		functions = append(functions, s.convertToProviderFunction(lambdaFunc, nil))
	}

	return functions, nil
}

// AdoptFunction adopts an existing Lambda function into management
func (s *ServerlessService) AdoptFunction(ctx context.Context, id string) (*provider.Function, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	endpoint := fmt.Sprintf("/2015-03-31/functions/%s", id)
	resp, err := client.Request("GET", endpoint, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		responseBody, _ := ReadResponse(resp)
		return nil, fmt.Errorf("GetFunction failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var lambdaFunc LambdaFunction
	if err := json.Unmarshal(responseBody, &lambdaFunc); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return s.convertToProviderFunction(lambdaFunc, nil), nil
}

// Helper methods

func (s *ServerlessService) createDummyZip() []byte {
	// Create a basic ZIP file with a simple Lambda function
	// This is a minimal implementation - in practice you'd want proper ZIP creation
	dummyCode := `def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello from Lambda!'
    }`

	// For now, just return the code as bytes - proper ZIP handling would be needed
	return []byte(dummyCode)
}

// CreateLayer creates a new Lambda layer
func (s *ServerlessService) CreateLayer(ctx context.Context, config *provider.LambdaLayerConfig) (*provider.LambdaLayer, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Build request body
	requestBody := map[string]interface{}{
		"LayerName":          config.Name,
		"Description":        config.Description,
		"CompatibleRuntimes": config.CompatibleRuntimes,
	}

	// Handle content upload
	if config.Content.LocalPath != "" {
		// Read ZIP file from local path
		zipData, err := os.ReadFile(config.Content.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read layer ZIP: %w", err)
		}
		requestBody["Content"] = map[string]interface{}{
			"ZipFile": base64.StdEncoding.EncodeToString(zipData),
		}
	} else if config.Content.S3Bucket != "" && config.Content.S3Key != "" {
		// Use S3 location
		requestBody["Content"] = map[string]interface{}{
			"S3Bucket": config.Content.S3Bucket,
			"S3Key":    config.Content.S3Key,
		}
	} else if len(config.Content.ZipFile) > 0 {
		// Use provided ZIP data
		requestBody["Content"] = map[string]interface{}{
			"ZipFile": base64.StdEncoding.EncodeToString(config.Content.ZipFile),
		}
	} else {
		return nil, fmt.Errorf("no layer content provided")
	}

	// Convert to JSON
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Make the request - layers API requires layer name in path
	endpoint := fmt.Sprintf("/2018-10-31/layers/%s/versions", config.Name)
	resp, err := client.Request("POST", endpoint, nil, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create layer: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("CreateLayer failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var layerResp struct {
		LayerArn        string `json:"LayerArn"`
		LayerVersionArn string `json:"LayerVersionArn"`
		Version         int    `json:"Version"`
		CreatedDate     string `json:"CreatedDate"`
	}
	if err := json.Unmarshal(responseBody, &layerResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to provider layer
	createdAt := time.Now()
	if layerResp.CreatedDate != "" {
		if t, err := time.Parse(time.RFC3339, layerResp.CreatedDate); err == nil {
			createdAt = t
		}
	}

	return &provider.LambdaLayer{
		ID:                 config.Name,
		Name:               config.Name,
		Description:        config.Description,
		Version:            layerResp.Version,
		CompatibleRuntimes: config.CompatibleRuntimes,
		LayerArn:           layerResp.LayerArn,
		LayerVersionArn:    layerResp.LayerVersionArn,
		CreatedAt:          createdAt,
		ProviderData: map[string]interface{}{
			"layerArn":        layerResp.LayerArn,
			"layerVersionArn": layerResp.LayerVersionArn,
		},
	}, nil
}

func (s *ServerlessService) convertToProviderFunction(lambdaFunc LambdaFunction, tags map[string]string) *provider.Function {
	createdAt := time.Now()
	if lambdaFunc.LastModified != "" {
		if t, err := time.Parse(time.RFC3339, lambdaFunc.LastModified); err == nil {
			createdAt = t
		}
	}

	var environment map[string]string
	if lambdaFunc.Environment != nil {
		environment = lambdaFunc.Environment.Variables
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	// Generate a simple URL for the function
	functionURL := fmt.Sprintf("https://%s.lambda-url.%s.on.aws/",
		lambdaFunc.FunctionName, s.provider.region)

	return &provider.Function{
		ID:          lambdaFunc.FunctionName,
		Name:        lambdaFunc.FunctionName,
		Runtime:     lambdaFunc.Runtime,
		Handler:     lambdaFunc.Handler,
		Memory:      lambdaFunc.MemorySize,
		Timeout:     lambdaFunc.Timeout,
		Environment: environment,
		URL:         functionURL,
		Tags:        tags,
		CreatedAt:   createdAt,
		ProviderData: map[string]interface{}{
			"arn":   lambdaFunc.FunctionArn,
			"state": lambdaFunc.State,
		},
	}
}
