package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// CreateFunction creates a new Lambda function with enhanced error handling and role validation
func (s *ServerlessService) CreateFunction(ctx context.Context, config *provider.FunctionConfig) (*provider.Function, error) {
	client, err := s.provider.CreateClient("lambda")
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda client: %w", err)
	}

	// Resolve role ARN properly (handle both role names and ARNs)
	roleArn, err := s.resolveRoleArn(ctx, config.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IAM role: %w", err)
	}

	// Wait for role and policies to be ready before creating function
	if err := s.waitForRoleReady(ctx, roleArn); err != nil {
		return nil, fmt.Errorf("IAM role not ready: %w", err)
	}

	// Build the request body
	requestBody := map[string]interface{}{
		"FunctionName": config.Name,
		"Runtime":      config.Runtime,
		"Handler":      config.Handler,
		"MemorySize":   config.Memory,
		"Timeout":      config.Timeout,
		"Role":         roleArn, // Use resolved ARN
	}

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

	// Create function with retry logic for IAM propagation delays
	return s.createFunctionWithRetry(ctx, client, requestBody, 3)
}

// createDummyZip creates a minimal Lambda deployment package for testing
func (s *ServerlessService) createDummyZip() string {
	// Simple base64 encoded ZIP with a minimal Lambda function
	dummyZip := "UEsDBAoAAAAAAIdYU1QAAAAAAAAAAAAAAAAJAAAAaW5kZXguanNleHBvcnRzLmhhbmRsZXIgPSBhc3luYyAoZXZlbnQpID0+ICh7CiAgICBzdGF0dXNDb2RlOiAyMDAsCiAgICBib2R5OiBKU09OLnN0cmluZ2lmeSgnSGVsbG8gZnJvbSBMYW1iZGEhJykKfSk7UEsBAhQAFAAAAAAIdYU1QAAAAAAAAAAAAAAAAJAAAAaW5kZXguanNQSwUGAAAAAAEAAQA3AAAAMwAAAAA="
	return dummyZip
}

// resolveRoleArn converts a role name or ARN to a full ARN
func (s *ServerlessService) resolveRoleArn(ctx context.Context, roleInput string) (string, error) {
	if roleInput == "" {
		return "", fmt.Errorf("IAM role ARN or name is required")
	}

	// If it's already an ARN, return it
	if strings.HasPrefix(roleInput, "arn:aws:iam::") {
		return roleInput, nil
	}

	// It's a role name, look up the ARN
	iamService := NewIAMService(s.provider)
	role, err := iamService.GetRole(ctx, roleInput)
	if err != nil {
		return "", fmt.Errorf("failed to find IAM role '%s': %w", roleInput, err)
	}

	return role.ARN, nil
}

// waitForRoleReady waits for the IAM role to be ready for Lambda use
func (s *ServerlessService) waitForRoleReady(ctx context.Context, roleArn string) error {
	// Extract role name from ARN
	parts := strings.Split(roleArn, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid role ARN format: %s", roleArn)
	}
	roleName := parts[len(parts)-1]

	iamService := NewIAMService(s.provider)

	// Wait for role to be ready with exponential backoff
	maxAttempts := 10
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if role exists and is assumable
		_, err := iamService.GetRole(ctx, roleName)
		if err != nil {
			if attempt == maxAttempts-1 {
				return fmt.Errorf("role not ready after %d attempts: %w", maxAttempts, err)
			}
			// Wait before retrying
			waitTime := time.Duration(1<<uint(attempt)) * time.Second
			if waitTime > 30*time.Second {
				waitTime = 30 * time.Second
			}
			time.Sleep(waitTime)
			continue
		}

		// Role exists, check if policies are attached
		policies, err := iamService.ListAttachedPolicies(ctx, roleName)
		if err == nil && len(policies) > 0 {
			// Role is ready
			return nil
		}

		// Wait a bit more for policy propagation
		if attempt < maxAttempts-1 {
			waitTime := time.Duration(1+attempt) * time.Second
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("role policies not ready after %d attempts", maxAttempts)
}

// createFunctionWithRetry creates a Lambda function with retry logic for IAM propagation delays
func (s *ServerlessService) createFunctionWithRetry(ctx context.Context, client *AWSClient, requestBody map[string]interface{}, maxRetries int) (*provider.Function, error) {
	// Convert to JSON
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Request("POST", "/2015-03-31/functions", nil, body)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		responseBody, err := ReadResponse(resp)
		if err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode == 201 {
			// Success - parse response
			var lambdaFunc LambdaFunction
			if err := json.Unmarshal(responseBody, &lambdaFunc); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("failed to parse response: %w", err)
			}

			resp.Body.Close()
			return &provider.Function{
				Name:        lambdaFunc.FunctionName,
				Runtime:     lambdaFunc.Runtime,
				Handler:     lambdaFunc.Handler,
				Memory:      lambdaFunc.MemorySize,
				Timeout:     lambdaFunc.Timeout,
				Environment: lambdaFunc.Environment.Variables,
			}, nil
		}

		responseStr := string(responseBody)
		resp.Body.Close()

		// Check for IAM role assumption errors
		if resp.StatusCode == 400 && (strings.Contains(responseStr, "cannot be assumed") ||
			strings.Contains(responseStr, "Invalid role") ||
			strings.Contains(responseStr, "role is not authorized")) {

			if attempt < maxRetries-1 {
				// Wait with exponential backoff for IAM propagation
				waitTime := time.Duration(2<<uint(attempt)) * time.Second
				if waitTime > 30*time.Second {
					waitTime = 30 * time.Second
				}
				fmt.Printf("IAM role not ready, waiting %v before retry %d/%d...\n", waitTime, attempt+1, maxRetries)
				time.Sleep(waitTime)
				continue
			}
		}

		// Parse AWS error for better messaging
		var errorResp map[string]interface{}
		if json.Unmarshal(responseBody, &errorResp) == nil {
			if errorType, ok := errorResp["errorType"].(string); ok {
				if message, ok := errorResp["errorMessage"].(string); ok {
					return nil, fmt.Errorf("AWS Lambda error (%s): %s", errorType, message)
				}
			}
		}

		return nil, fmt.Errorf("CreateFunction failed with status %d: %s", resp.StatusCode, responseStr)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to create function after %d attempts: %w", maxRetries, lastErr)
	}

	return nil, fmt.Errorf("failed to create function after %d attempts", maxRetries)
}

func (s *ServerlessService) convertToProviderFunction(lambdaFunc *LambdaFunction, tags map[string]string) *provider.Function {
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

// UpdateFunction updates an existing Lambda function
func (s *ServerlessService) UpdateFunction(ctx context.Context, id string, config *provider.FunctionConfig) error {
	// TODO: Implement function update
	return fmt.Errorf("UpdateFunction not yet implemented")
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

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		responseBody, _ := ReadResponse(resp)
		return fmt.Errorf("delete function failed with status %d: %s", resp.StatusCode, string(responseBody))
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
		return nil, fmt.Errorf("invoke function failed with status %d: %s", resp.StatusCode, string(responseBody))
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

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("list functions failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var listResp ListFunctionsResponse
	if err := json.Unmarshal(responseBody, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var functions []*provider.Function
	for _, lambdaFunc := range listResp.Functions {
		function := s.convertToProviderFunction(&lambdaFunc, nil)
		functions = append(functions, function)
	}

	return functions, nil
}

// AdoptFunction adopts an existing Lambda function into Genesys management
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

	responseBody, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get function failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	var lambdaFunc LambdaFunction
	if err := json.Unmarshal(responseBody, &lambdaFunc); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return s.convertToProviderFunction(&lambdaFunc, nil), nil
}

// CreateLayer creates a Lambda layer (placeholder for missing method in commands)
func (s *ServerlessService) CreateLayer(ctx context.Context, config interface{}) (interface{}, error) {
	// TODO: Implement layer creation
	return nil, fmt.Errorf("CreateLayer not yet implemented")
}
