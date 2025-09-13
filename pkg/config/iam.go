package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/javanhut/genesys/pkg/provider/aws"
)

// UnifiedIAMConfig represents IAM configuration that can be used across all AWS resources
type UnifiedIAMConfig struct {
	RoleName         string            `toml:"role_name,omitempty"`
	RoleArn          string            `toml:"role_arn,omitempty"`
	AutoManage       bool              `toml:"auto_manage,omitempty"`
	AutoCleanup      bool              `toml:"auto_cleanup,omitempty"`
	RequiredPolicies []string          `toml:"required_policies,omitempty"`
	TrustPolicy      string            `toml:"trust_policy,omitempty"` // "s3", "ec2", "lambda", or custom JSON
	ManagedBy        string            `toml:"managed_by,omitempty"`     // "genesys" or "external" 
	Tags             map[string]string `toml:"tags,omitempty"`
	Description      string            `toml:"description,omitempty"`
}

// IAMRoleManager handles unified IAM role management across all AWS resources
type IAMRoleManager struct {
	provider   *aws.AWSProvider
	iamService *aws.IAMService
}

// NewIAMRoleManager creates a new unified IAM role manager
func NewIAMRoleManager(provider *aws.AWSProvider) *IAMRoleManager {
	return &IAMRoleManager{
		provider:   provider,
		iamService: aws.NewIAMService(provider),
	}
}

// EnsureRole ensures a role exists with required permissions using intelligent management
func (irm *IAMRoleManager) EnsureRole(ctx context.Context, config *UnifiedIAMConfig, resourceType, resourceName string) (string, error) {
	// Apply defaults if not set
	if err := irm.applyDefaults(config, resourceType, resourceName); err != nil {
		return "", fmt.Errorf("failed to apply defaults: %w", err)
	}

	// Check if role exists
	existingRole, err := irm.iamService.GetRole(ctx, config.RoleName)
	if err == nil {
		// Role exists - handle existing role
		return irm.handleExistingRole(ctx, existingRole, config)
	}

	if !aws.IsRoleNotFoundError(err) {
		return "", fmt.Errorf("error checking role: %w", err)
	}

	// Role doesn't exist - create new role
	return irm.createNewRole(ctx, config, resourceType, resourceName)
}

// applyDefaults applies default values to IAM configuration
func (irm *IAMRoleManager) applyDefaults(config *UnifiedIAMConfig, resourceType, resourceName string) error {
	// Set default role name if not provided
	if config.RoleName == "" {
		timestamp := time.Now().Format("20060102-150405")
		config.RoleName = fmt.Sprintf("genesys-%s-%s-%s", resourceType, resourceName, timestamp)
	}

	// Set default auto-manage
	if !config.AutoManage && config.ManagedBy == "" {
		config.AutoManage = true
	}

	// Set default auto-cleanup
	if !config.AutoCleanup && config.ManagedBy == "" {
		config.AutoCleanup = true
	}

	// Set default required policies based on resource type
	if len(config.RequiredPolicies) == 0 {
		config.RequiredPolicies = getDefaultPoliciesForResource(resourceType)
	}

	// Set default trust policy based on resource type
	if config.TrustPolicy == "" {
		config.TrustPolicy = resourceType
	}

	// Set default description
	if config.Description == "" {
		config.Description = fmt.Sprintf("Auto-created by Genesys for %s: %s", resourceType, resourceName)
	}

	// Set default tags
	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
	if config.Tags["ManagedBy"] == "" {
		config.Tags["ManagedBy"] = "genesys"
	}
	if config.Tags["ResourceType"] == "" {
		config.Tags["ResourceType"] = resourceType
	}
	if config.Tags["ResourceName"] == "" {
		config.Tags["ResourceName"] = resourceName
	}

	return nil
}

// handleExistingRole handles cases where a role already exists
func (irm *IAMRoleManager) handleExistingRole(ctx context.Context, role *aws.Role, config *UnifiedIAMConfig) (string, error) {
	// Determine if role is managed by Genesys
	isGenesysManaged := role.Tags["ManagedBy"] == "genesys"
	config.ManagedBy = "external"
	if isGenesysManaged {
		config.ManagedBy = "genesys"
	}

	fmt.Printf("  ✓ Found existing role: %s (%s)\n", config.RoleName, config.ManagedBy)

	// Validate and update permissions if needed
	if err := irm.validateAndUpdatePermissions(ctx, role, config); err != nil {
		return "", fmt.Errorf("failed to validate/update role permissions: %w", err)
	}

	return role.ARN, nil
}

// createNewRole creates a new IAM role with all required policies
func (irm *IAMRoleManager) createNewRole(ctx context.Context, config *UnifiedIAMConfig, resourceType, resourceName string) (string, error) {
	fmt.Printf("  Creating new role: %s\n", config.RoleName)

	// Get trust policy JSON
	trustPolicyJSON, err := getTrustPolicyForResource(config.TrustPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to get trust policy: %w", err)
	}

	// Create role configuration
	roleConfig := &aws.RoleConfig{
		Name:        config.RoleName,
		Description: config.Description,
		TrustPolicy: trustPolicyJSON,
		Tags:        config.Tags,
	}

	// Convert required policies to ARNs
	policyARNs := aws.ConvertRequirementsToARNs(config.RequiredPolicies)

	// Create role with policies
	role, err := irm.iamService.CreateRoleWithPolicies(ctx, roleConfig, policyARNs)
	if err != nil {
		return "", fmt.Errorf("failed to create role: %w", err)
	}

	// Mark as Genesys-managed
	config.ManagedBy = "genesys"
	config.RoleArn = role.ARN

	fmt.Printf("  ✓ Created role: %s\n", config.RoleName)
	for _, policy := range config.RequiredPolicies {
		fmt.Printf("    ✓ Attached policy: %s\n", policy)
	}

	return role.ARN, nil
}

// validateAndUpdatePermissions ensures role has all required permissions
func (irm *IAMRoleManager) validateAndUpdatePermissions(ctx context.Context, role *aws.Role, config *UnifiedIAMConfig) error {
	// Get currently attached policies
	currentPolicies, err := irm.iamService.ListAttachedPolicies(ctx, role.Name)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	// Convert current policies to map for faster lookup
	currentPolicyMap := make(map[string]bool)
	for _, policy := range currentPolicies {
		currentPolicyMap[policy.ARN] = true
	}

	// Convert required policies to ARNs
	requiredPolicyARNs := aws.ConvertRequirementsToARNs(config.RequiredPolicies)

	// Find missing policies (additive approach - we only add, never remove)
	var missingPolicies []string
	for _, requiredARN := range requiredPolicyARNs {
		if !currentPolicyMap[requiredARN] {
			missingPolicies = append(missingPolicies, requiredARN)
		}
	}

	// Attach missing policies
	if len(missingPolicies) > 0 {
		fmt.Printf("  Adding missing policies to role: %s\n", role.Name)
		for _, policyARN := range missingPolicies {
			policyName := aws.ExtractPolicyName(policyARN)
			fmt.Printf("    Adding: %s\n", policyName)
			
			if err := irm.iamService.AttachPolicy(ctx, role.Name, policyARN); err != nil {
				fmt.Printf("    Warning: Failed to attach %s: %v\n", policyName, err)
				// Continue with other policies - don't fail the entire operation
			} else {
				fmt.Printf("    ✓ Attached: %s\n", policyName)
			}
		}
	}

	return nil
}

// CleanupRole removes a Genesys-managed role if auto-cleanup is enabled
func (irm *IAMRoleManager) CleanupRole(ctx context.Context, config *UnifiedIAMConfig) error {
	// Only cleanup roles that are managed by Genesys and have auto-cleanup enabled
	if config.ManagedBy != "genesys" || !config.AutoCleanup {
		fmt.Printf("  Info: Role cleanup skipped: %s (managed_by=%s, auto_cleanup=%v)\n", 
			config.RoleName, config.ManagedBy, config.AutoCleanup)
		return nil
	}

	// Get role to verify it exists and is managed by Genesys
	role, err := irm.iamService.GetRole(ctx, config.RoleName)
	if err != nil {
		if aws.IsRoleNotFoundError(err) {
			// Role doesn't exist, nothing to clean up
			return nil
		}
		return fmt.Errorf("failed to get role for cleanup: %w", err)
	}

	// Double-check that it's managed by Genesys
	if role.Tags["ManagedBy"] != "genesys" {
		return fmt.Errorf("role is not managed by Genesys, skipping cleanup")
	}

	fmt.Printf("  Cleaning up Genesys-managed role: %s\n", config.RoleName)

	// Detach all managed policies
	attachedPolicies, err := irm.iamService.ListAttachedPolicies(ctx, config.RoleName)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	for _, policy := range attachedPolicies {
		policyName := aws.ExtractPolicyName(policy.ARN)
		if err := irm.iamService.DetachPolicy(ctx, config.RoleName, policy.ARN); err != nil {
			fmt.Printf("    Warning: Failed to detach policy %s: %v\n", policyName, err)
		} else {
			fmt.Printf("    ✓ Detached policy: %s\n", policyName)
		}
	}

	// Delete the role
	if err := irm.iamService.DeleteRole(ctx, config.RoleName); err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	fmt.Printf("  ✓ Role cleaned up: %s\n", config.RoleName)
	return nil
}

// getDefaultPoliciesForResource returns default required policies for each resource type
func getDefaultPoliciesForResource(resourceType string) []string {
	switch strings.ToLower(resourceType) {
	case "s3":
		return []string{
			"S3 full access",
			"CloudWatch full access",
		}
	case "ec2":
		return []string{
			"Systems Manager Parameter access",
			"CloudWatch full access",
		}
	case "lambda":
		return []string{
			"Basic CloudWatch Logs access",
		}
	case "rds":
		return []string{
			"CloudWatch full access",
		}
	default:
		return []string{
			"CloudWatch full access", // Default minimal access
		}
	}
}

// getTrustPolicyForResource returns the appropriate trust policy JSON for a resource type
func getTrustPolicyForResource(trustPolicyType string) (string, error) {
	switch strings.ToLower(trustPolicyType) {
	case "lambda":
		return aws.GetLambdaTrustPolicy(), nil
	case "s3":
		return getS3TrustPolicy(), nil
	case "ec2":
		return getEC2TrustPolicy(), nil
	case "rds":
		return getRDSTrustPolicy(), nil
	default:
		// If it looks like JSON, return it directly
		if strings.Contains(trustPolicyType, "{") {
			return trustPolicyType, nil
		}
		// Default to EC2 trust policy for unknown types
		return getEC2TrustPolicy(), nil
	}
}

// getS3TrustPolicy returns trust policy for S3-related roles
func getS3TrustPolicy() string {
	return `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": ["s3.amazonaws.com", "ec2.amazonaws.com"]
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`
}

// getEC2TrustPolicy returns trust policy for EC2 instance profiles
func getEC2TrustPolicy() string {
	return `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`
}

// getRDSTrustPolicy returns trust policy for RDS-related roles
func getRDSTrustPolicy() string {
	return `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "rds.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`
}

// ValidateIAMConfig validates IAM configuration
func ValidateIAMConfig(config *UnifiedIAMConfig) error {
	if config == nil {
		return nil // IAM config is optional
	}

	// If RoleArn is provided, it should be a valid ARN format
	if config.RoleArn != "" && !strings.HasPrefix(config.RoleArn, "arn:aws:iam::") {
		return fmt.Errorf("invalid role ARN format: %s", config.RoleArn)
	}

	// If RoleName is provided, it should follow AWS naming rules
	if config.RoleName != "" {
		if len(config.RoleName) > 64 {
			return fmt.Errorf("role name too long (max 64 characters): %s", config.RoleName)
		}
		if strings.Contains(config.RoleName, " ") {
			return fmt.Errorf("role name cannot contain spaces: %s", config.RoleName)
		}
	}

	return nil
}

// FormatRoleName creates a standardized role name
func FormatRoleName(resourceType, resourceName string) string {
	timestamp := time.Now().Format("20060102-150405")
	// Clean the resource name to be AWS-compatible
	cleanName := strings.ReplaceAll(resourceName, "_", "-")
	cleanName = strings.ReplaceAll(cleanName, " ", "-")
	cleanName = strings.ToLower(cleanName)
	return fmt.Sprintf("genesys-%s-%s-%s", resourceType, cleanName, timestamp)
}