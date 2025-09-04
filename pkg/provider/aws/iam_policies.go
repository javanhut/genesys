package aws

import (
	"encoding/json"
	"fmt"
)

// GetLambdaDeploymentPolicy returns an inline policy for Lambda deployment permissions
func GetLambdaDeploymentPolicy(functionName string) (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":    "LambdaFunctionManagement",
				"Effect": "Allow",
				"Action": []string{
					"lambda:CreateFunction",
					"lambda:UpdateFunctionCode",
					"lambda:UpdateFunctionConfiguration",
					"lambda:GetFunction",
					"lambda:GetFunctionConfiguration",
					"lambda:DeleteFunction",
					"lambda:TagResource",
					"lambda:UntagResource",
					"lambda:ListTags",
					"lambda:InvokeFunction",
					"lambda:GetFunctionCodeSigningConfig",
				},
				"Resource": fmt.Sprintf("arn:aws:lambda:*:*:function:%s*", functionName),
			},
			{
				"Sid":    "LambdaLayerManagement",
				"Effect": "Allow",
				"Action": []string{
					"lambda:PublishLayerVersion",
					"lambda:DeleteLayerVersion",
					"lambda:GetLayerVersion",
					"lambda:GetLayerVersionPolicy",
					"lambda:ListLayerVersions",
				},
				"Resource": "arn:aws:lambda:*:*:layer:*",
			},
			{
				"Sid":    "LambdaFunctionURLManagement",
				"Effect": "Allow",
				"Action": []string{
					"lambda:CreateFunctionUrlConfig",
					"lambda:UpdateFunctionUrlConfig",
					"lambda:GetFunctionUrlConfig",
					"lambda:DeleteFunctionUrlConfig",
					"lambda:AddPermission",
					"lambda:RemovePermission",
				},
				"Resource": fmt.Sprintf("arn:aws:lambda:*:*:function:%s*", functionName),
			},
			{
				"Sid":      "IAMPassRole",
				"Effect":   "Allow",
				"Action":   "iam:PassRole",
				"Resource": "*",
				"Condition": map[string]interface{}{
					"StringEquals": map[string]string{
						"iam:PassedToService": "lambda.amazonaws.com",
					},
				},
			},
			{
				"Sid":    "CloudWatchLogsAccess",
				"Effect": "Allow",
				"Action": []string{
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
					"logs:DescribeLogGroups",
					"logs:DescribeLogStreams",
				},
				"Resource": fmt.Sprintf("arn:aws:logs:*:*:log-group:/aws/lambda/%s*", functionName),
			},
		},
	}

	policyJSON, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal policy: %w", err)
	}

	return string(policyJSON), nil
}

// GetMinimalLambdaExecutionPolicy returns the minimal policy for Lambda execution
func GetMinimalLambdaExecutionPolicy(functionName string) (string, error) {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
				},
				"Resource": fmt.Sprintf("arn:aws:logs:*:*:log-group:/aws/lambda/%s:*", functionName),
			},
		},
	}

	policyJSON, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal policy: %w", err)
	}

	return string(policyJSON), nil
}

// PolicyTemplate represents a reusable policy template
type PolicyTemplate struct {
	Name        string
	Description string
	Type        string                       // "managed" or "inline"
	ManagedARN  string                       // For managed policies
	InlineFunc  func(string) (string, error) // For inline policies, takes resource name
}

// GetAvailablePolicyTemplates returns all available policy templates
func GetAvailablePolicyTemplates() []PolicyTemplate {
	return []PolicyTemplate{
		// Managed policies
		{
			Name:        "Basic CloudWatch Logs access",
			Description: "Allows Lambda to write logs to CloudWatch",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
		},
		{
			Name:        "VPC access",
			Description: "Allows Lambda to access resources in a VPC",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole",
		},

		// Inline policies for deployment
		{
			Name:        "Lambda deployment permissions",
			Description: "Permissions to deploy Lambda functions and layers",
			Type:        "inline",
			InlineFunc:  GetLambdaDeploymentPolicy,
		},
		{
			Name:        "Minimal Lambda execution",
			Description: "Minimal permissions for Lambda execution (logs only)",
			Type:        "inline",
			InlineFunc:  GetMinimalLambdaExecutionPolicy,
		},

		// Service access policies
		{
			Name:        "S3 read access",
			Description: "Read objects from S3 buckets",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
		},
		{
			Name:        "S3 full access",
			Description: "Read and write S3 buckets",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/AmazonS3FullAccess",
		},
		{
			Name:        "DynamoDB read access",
			Description: "Read from DynamoDB tables",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/AmazonDynamoDBReadOnlyAccess",
		},
		{
			Name:        "DynamoDB read/write access",
			Description: "Full access to DynamoDB tables",
			Type:        "managed",
			ManagedARN:  "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
		},
	}
}
