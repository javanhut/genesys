package aws

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// IAMService implements AWS IAM operations
type IAMService struct {
	provider *AWSProvider
}

// NewIAMService creates a new IAM service
func NewIAMService(p *AWSProvider) *IAMService {
	return &IAMService{
		provider: p,
	}
}

// Role represents an IAM role
type Role struct {
	Name             string
	ARN              string
	AssumeRolePolicy string
	Description      string
	Tags             map[string]string
	CreatedAt        time.Time
}

// Policy represents an IAM policy
type Policy struct {
	Name        string
	ARN         string
	Description string
	Document    string
}

// RoleConfig for creating/updating roles
type RoleConfig struct {
	Name        string
	Description string
	TrustPolicy string
	Tags        map[string]string
}

// IAM API response structures
type GetRoleResponse struct {
	XMLName xml.Name `xml:"GetRoleResponse"`
	Result  struct {
		Role struct {
			RoleName                 string `xml:"RoleName"`
			Arn                      string `xml:"Arn"`
			AssumeRolePolicyDocument string `xml:"AssumeRolePolicyDocument"`
			Description              string `xml:"Description"`
			CreateDate               string `xml:"CreateDate"`
		} `xml:"Role"`
	} `xml:"GetRoleResult"`
}

type CreateRoleResponse struct {
	XMLName xml.Name `xml:"CreateRoleResponse"`
	Result  struct {
		Role struct {
			RoleName   string `xml:"RoleName"`
			Arn        string `xml:"Arn"`
			CreateDate string `xml:"CreateDate"`
		} `xml:"Role"`
	} `xml:"CreateRoleResult"`
}

type ListAttachedRolePoliciesResponse struct {
	XMLName xml.Name `xml:"ListAttachedRolePoliciesResponse"`
	Result  struct {
		AttachedPolicies []struct {
			PolicyName string `xml:"PolicyName"`
			PolicyArn  string `xml:"PolicyArn"`
		} `xml:"AttachedPolicies>member"`
		IsTruncated bool   `xml:"IsTruncated"`
		Marker      string `xml:"Marker"`
	} `xml:"ListAttachedRolePoliciesResult"`
}

type ListRoleTagsResponse struct {
	XMLName xml.Name `xml:"ListRoleTagsResponse"`
	Result  struct {
		Tags []struct {
			Key   string `xml:"Key"`
			Value string `xml:"Value"`
		} `xml:"Tags>member"`
		IsTruncated bool   `xml:"IsTruncated"`
		Marker      string `xml:"Marker"`
	} `xml:"ListRoleTagsResult"`
}

// CreateRole creates a new IAM role
func (s *IAMService) CreateRole(ctx context.Context, config *RoleConfig) (*Role, error) {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":                   "CreateRole",
		"RoleName":                 config.Name,
		"AssumeRolePolicyDocument": config.TrustPolicy,
		"Version":                  "2010-05-08",
	}

	if config.Description != "" {
		params["Description"] = config.Description
	}

	// Add tags if provided
	tagIndex := 1
	for key, value := range config.Tags {
		params[fmt.Sprintf("Tags.member.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("Tags.member.%d.Value", tagIndex)] = value
		tagIndex++
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CreateRole failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var createResp CreateRoleResponse
	if err := xml.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	createdAt := time.Now()
	if createResp.Result.Role.CreateDate != "" {
		createdAt, _ = time.Parse(time.RFC3339, createResp.Result.Role.CreateDate)
	}

	return &Role{
		Name:             createResp.Result.Role.RoleName,
		ARN:              createResp.Result.Role.Arn,
		AssumeRolePolicy: config.TrustPolicy,
		Description:      config.Description,
		Tags:             config.Tags,
		CreatedAt:        createdAt,
	}, nil
}

// GetRole retrieves an existing IAM role
func (s *IAMService) GetRole(ctx context.Context, roleName string) (*Role, error) {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":   "GetRole",
		"RoleName": roleName,
		"Version":  "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	defer resp.Body.Close()

	// AWS IAM API returns 404 for non-existent roles, but may also return 400 with error details
	if resp.StatusCode == 404 || resp.StatusCode == 400 {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		// Check if this is a role not found error
		if strings.Contains(bodyStr, "NoSuchEntity") || strings.Contains(bodyStr, "Role not found") {
			return nil, fmt.Errorf("role not found: %s", roleName)
		}
		return nil, fmt.Errorf("GetRole failed with status %d: %s", resp.StatusCode, bodyStr)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GetRole failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var getResp GetRoleResponse
	if err := xml.Unmarshal(body, &getResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// URL decode the assume role policy document
	assumeRolePolicy, _ := url.QueryUnescape(getResp.Result.Role.AssumeRolePolicyDocument)

	createdAt := time.Now()
	if getResp.Result.Role.CreateDate != "" {
		createdAt, _ = time.Parse(time.RFC3339, getResp.Result.Role.CreateDate)
	}

	// Get tags
	tags, _ := s.ListRoleTags(ctx, roleName)

	return &Role{
		Name:             getResp.Result.Role.RoleName,
		ARN:              getResp.Result.Role.Arn,
		AssumeRolePolicy: assumeRolePolicy,
		Description:      getResp.Result.Role.Description,
		Tags:             tags,
		CreatedAt:        createdAt,
	}, nil
}

// DeleteRole deletes an IAM role
func (s *IAMService) DeleteRole(ctx context.Context, roleName string) error {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":   "DeleteRole",
		"RoleName": roleName,
		"Version":  "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DeleteRole failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AttachPolicy attaches a managed policy to a role
func (s *IAMService) AttachPolicy(ctx context.Context, roleName, policyArn string) error {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":    "AttachRolePolicy",
		"RoleName":  roleName,
		"PolicyArn": policyArn,
		"Version":   "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to attach policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AttachRolePolicy failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DetachPolicy detaches a managed policy from a role
func (s *IAMService) DetachPolicy(ctx context.Context, roleName, policyArn string) error {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":    "DetachRolePolicy",
		"RoleName":  roleName,
		"PolicyArn": policyArn,
		"Version":   "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to detach policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DetachRolePolicy failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListAttachedPolicies lists policies attached to a role
func (s *IAMService) ListAttachedPolicies(ctx context.Context, roleName string) ([]*Policy, error) {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":   "ListAttachedRolePolicies",
		"RoleName": roleName,
		"Version":  "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached policies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ListAttachedRolePolicies failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResp ListAttachedRolePoliciesResponse
	if err := xml.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	policies := make([]*Policy, 0, len(listResp.Result.AttachedPolicies))
	for _, p := range listResp.Result.AttachedPolicies {
		policies = append(policies, &Policy{
			Name: p.PolicyName,
			ARN:  p.PolicyArn,
		})
	}

	return policies, nil
}

// ListRoleTags lists tags for a role
func (s *IAMService) ListRoleTags(ctx context.Context, roleName string) (map[string]string, error) {
	client, err := s.provider.CreateClient("iam")
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	params := map[string]string{
		"Action":   "ListRoleTags",
		"RoleName": roleName,
		"Version":  "2010-05-08",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list role tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Tags might not be supported or accessible, return empty map
		return make(map[string]string), nil
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var listResp ListRoleTagsResponse
	if err := xml.Unmarshal(body, &listResp); err != nil {
		// If parsing fails, return empty map rather than error
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	for _, tag := range listResp.Result.Tags {
		tags[tag.Key] = tag.Value
	}

	return tags, nil
}

// GetLambdaTrustPolicy returns the default Lambda trust policy
func GetLambdaTrustPolicy() string {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]string{
					"Service": "lambda.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	}

	jsonBytes, _ := json.Marshal(policy)
	return string(jsonBytes)
}

// ConvertRequirementsToARNs converts human-readable policy requirements to ARNs
func ConvertRequirementsToARNs(requirements []string) []string {
	policyMap := map[string]string{
		// Lambda execution policies
		"Basic CloudWatch Logs access": "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
		"VPC access":                   "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole",

		// Lambda management policies (for deployment)
		"Lambda full access":            "arn:aws:iam::aws:policy/AWSLambda_FullAccess",
		"Lambda read-only access":       "arn:aws:iam::aws:policy/AWSLambda_ReadOnlyAccess",
		"Lambda layer management":       "arn:aws:iam::aws:policy/service-role/AWSLambdaRole",
		"Lambda deployment permissions": "INLINE_POLICY:lambda_deployment", // Special marker for inline policy

		// Service access policies
		"DynamoDB read/write access":       "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
		"DynamoDB read-only access":        "arn:aws:iam::aws:policy/AmazonDynamoDBReadOnlyAccess",
		"S3 full access":                   "arn:aws:iam::aws:policy/AmazonS3FullAccess",
		"S3 read-only access":              "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		"SQS full access":                  "arn:aws:iam::aws:policy/AmazonSQSFullAccess",
		"SNS full access":                  "arn:aws:iam::aws:policy/AmazonSNSFullAccess",
		"Secrets Manager read access":      "arn:aws:iam::aws:policy/SecretsManagerReadWrite",
		"X-Ray tracing":                    "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess",
		"Kinesis full access":              "arn:aws:iam::aws:policy/AmazonKinesisFullAccess",
		"EventBridge full access":          "arn:aws:iam::aws:policy/AmazonEventBridgeFullAccess",
		"API Gateway invocation":           "arn:aws:iam::aws:policy/AmazonAPIGatewayInvokeFullAccess",
		"CloudWatch full access":           "arn:aws:iam::aws:policy/CloudWatchFullAccess",
		"Systems Manager Parameter access": "arn:aws:iam::aws:policy/AmazonSSMReadOnlyAccess",
	}

	arns := make([]string, 0, len(requirements))
	for _, req := range requirements {
		// If it's already an ARN, use it directly
		if strings.HasPrefix(req, "arn:") {
			arns = append(arns, req)
		} else if arn, exists := policyMap[req]; exists {
			arns = append(arns, arn)
		}
		// Skip unrecognized requirements
	}

	return arns
}

// IsRoleNotFoundError checks if the error indicates a role was not found
func IsRoleNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "role not found") ||
		strings.Contains(errStr, "NoSuchEntity") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "does not exist")
}

// ExtractPolicyName extracts the policy name from its ARN
func ExtractPolicyName(policyArn string) string {
	parts := strings.Split(policyArn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return policyArn
}

// CreateRoleWithPolicies creates an IAM role and attaches multiple policies with proper error handling
func (s *IAMService) CreateRoleWithPolicies(ctx context.Context, config *RoleConfig, policies []string) (*Role, error) {
	// 1. Create the role first
	role, err := s.CreateRole(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	// 2. Attach all policies with error handling and rollback
	var attachedPolicies []string
	for _, policyArn := range policies {
		if err := s.AttachPolicyWithRetry(ctx, role.Name, policyArn); err != nil {
			// Rollback: detach already attached policies and delete role
			s.rollbackRoleCreation(ctx, role.Name, attachedPolicies)
			return nil, fmt.Errorf("failed to attach policy %s: %w", policyArn, err)
		}
		attachedPolicies = append(attachedPolicies, policyArn)
	}

	// 3. Wait for IAM propagation
	if err := s.waitForRolePropagation(ctx, role.ARN); err != nil {
		return nil, fmt.Errorf("role created but propagation failed: %w", err)
	}

	return role, nil
}

// AttachPolicyWithRetry attaches a policy with retry logic for eventual consistency
func (s *IAMService) AttachPolicyWithRetry(ctx context.Context, roleName, policyArn string) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.AttachPolicy(ctx, roleName, policyArn)
		if err == nil {
			return nil
		}

		// If it's a temporary error, retry
		if attempt < maxRetries-1 && (strings.Contains(err.Error(), "throttling") ||
			strings.Contains(err.Error(), "temporarily unavailable")) {
			waitTime := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(waitTime)
			continue
		}

		return err
	}

	return fmt.Errorf("failed to attach policy after %d attempts", maxRetries)
}

// waitForRolePropagation waits for the role to be propagated across AWS regions
func (s *IAMService) waitForRolePropagation(ctx context.Context, roleArn string) error {
	// Extract role name from ARN
	parts := strings.Split(roleArn, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid role ARN format: %s", roleArn)
	}
	roleName := parts[len(parts)-1]

	maxAttempts := 15
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Try to get the role
		_, err := s.GetRole(ctx, roleName)
		if err == nil {
			// Role is accessible, check if policies are attached
			policies, err := s.ListAttachedPolicies(ctx, roleName)
			if err == nil && len(policies) > 0 {
				// At least some policies are attached, consider it ready
				return nil
			}
		}

		// Wait before retrying with exponential backoff (max 10 seconds)
		waitTime := time.Duration(1<<uint(attempt)) * time.Second
		if waitTime > 10*time.Second {
			waitTime = 10 * time.Second
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("role propagation timeout after %d attempts", maxAttempts)
}

// rollbackRoleCreation cleans up after a failed role creation
func (s *IAMService) rollbackRoleCreation(ctx context.Context, roleName string, attachedPolicies []string) {
	// Best effort cleanup - don't fail on errors during rollback

	// Detach any policies that were attached
	for _, policyArn := range attachedPolicies {
		s.DetachPolicy(ctx, roleName, policyArn)
	}

	// Delete the role
	s.DeleteRole(ctx, roleName)
}

// GetRoleArn gets the ARN for a role by name
func (s *IAMService) GetRoleArn(ctx context.Context, roleName string) (string, error) {
	role, err := s.GetRole(ctx, roleName)
	if err != nil {
		return "", err
	}
	return role.ARN, nil
}

// ValidateRole checks if a role exists and has the expected policies
func (s *IAMService) ValidateRole(ctx context.Context, roleName string, expectedPolicies []string) error {
	// Check if role exists
	_, err := s.GetRole(ctx, roleName)
	if err != nil {
		return fmt.Errorf("role validation failed: %w", err)
	}

	// Check attached policies
	attachedPolicies, err := s.ListAttachedPolicies(ctx, roleName)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	// Convert to map for easier lookup
	attachedMap := make(map[string]bool)
	for _, policy := range attachedPolicies {
		attachedMap[policy.ARN] = true
	}

	// Check if all expected policies are attached
	var missingPolicies []string
	for _, expectedArn := range expectedPolicies {
		if !attachedMap[expectedArn] {
			missingPolicies = append(missingPolicies, expectedArn)
		}
	}

	if len(missingPolicies) > 0 {
		return fmt.Errorf("missing policies: %v", missingPolicies)
	}

	return nil
}
