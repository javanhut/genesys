package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AWSNamingRules defines naming rules and formatting for AWS resources
type AWSNamingRules struct {
	Resource     string
	MinLength    int
	MaxLength    int
	Pattern      *regexp.Regexp
	Description  string
	AllowedChars string
	Formatter    func(string) string
	Examples     []string
}

// AWSResourceRules contains naming rules for all AWS resource types
var AWSResourceRules = map[string]AWSNamingRules{
	"lambda": {
		Resource:     "Lambda Function",
		MinLength:    1,
		MaxLength:    64,
		Pattern:      regexp.MustCompile(`^[a-zA-Z0-9-_]+$`),
		Description:  "Letters, numbers, hyphens, and underscores only",
		AllowedChars: "a-z, A-Z, 0-9, -, _",
		Formatter:    formatLambdaName,
		Examples:     []string{"my-function", "dataProcessor", "user_handler"},
	},
	"s3": {
		Resource:     "S3 Bucket",
		MinLength:    3,
		MaxLength:    63,
		Pattern:      regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`),
		Description:  "Lowercase letters, numbers, dots, and hyphens only",
		AllowedChars: "a-z, 0-9, ., -",
		Formatter:    formatS3Name,
		Examples:     []string{"my-bucket", "data.backup", "logs-2024"},
	},
	"ec2": {
		Resource:     "EC2 Instance",
		MinLength:    1,
		MaxLength:    255,
		Pattern:      regexp.MustCompile(`^[a-zA-Z0-9 ._-]+$`),
		Description:  "Letters, numbers, spaces, dots, underscores, and hyphens",
		AllowedChars: "a-z, A-Z, 0-9, space, ., _, -",
		Formatter:    formatEC2Name,
		Examples:     []string{"Web Server", "db-instance", "worker.node"},
	},
	"iam-role": {
		Resource:     "IAM Role",
		MinLength:    1,
		MaxLength:    64,
		Pattern:      regexp.MustCompile(`^[a-zA-Z0-9+=,.@_-]+$`),
		Description:  "Letters, numbers, and specific symbols: +=,.@_-",
		AllowedChars: "a-z, A-Z, 0-9, +, =, ,, ., @, _, -",
		Formatter:    formatIAMRoleName,
		Examples:     []string{"lambda-role", "EC2-Instance-Role", "service@role"},
	},
	"iam-policy": {
		Resource:     "IAM Policy",
		MinLength:    1,
		MaxLength:    128,
		Pattern:      regexp.MustCompile(`^[a-zA-Z0-9+=,.@_-]+$`),
		Description:  "Letters, numbers, and specific symbols: +=,.@_-",
		AllowedChars: "a-z, A-Z, 0-9, +, =, ,, ., @, _, -",
		Formatter:    formatIAMPolicyName,
		Examples:     []string{"S3-Access-Policy", "lambda.execution", "custom@policy"},
	},
}

// ValidateAndFormatName validates and formats a resource name according to AWS rules
func ValidateAndFormatName(resourceType, name string) (string, error) {
	rules, exists := AWSResourceRules[resourceType]
	if !exists {
		return name, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if name == "" {
		return "", fmt.Errorf("%s name cannot be empty", rules.Resource)
	}

	// Apply formatter
	formattedName := rules.Formatter(name)

	// Validate formatted name
	if len(formattedName) < rules.MinLength {
		return "", fmt.Errorf("%s name must be at least %d characters (got %d)",
			rules.Resource, rules.MinLength, len(formattedName))
	}

	if len(formattedName) > rules.MaxLength {
		return "", fmt.Errorf("%s name must be at most %d characters (got %d)",
			rules.Resource, rules.MaxLength, len(formattedName))
	}

	if !rules.Pattern.MatchString(formattedName) {
		return "", fmt.Errorf("%s name contains invalid characters. %s. Allowed: %s",
			rules.Resource, rules.Description, rules.AllowedChars)
	}

	return formattedName, nil
}

// AutoGenerateName generates a valid resource name with the given prefix
func AutoGenerateName(resourceType, prefix string) string {
	if prefix == "" {
		prefix = "genesys"
	}

	timestamp := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("%s-%s", prefix, timestamp)

	// Format according to resource rules
	if rules, exists := AWSResourceRules[resourceType]; exists {
		return rules.Formatter(name)
	}

	return name
}

// GetNamingRules returns the naming rules for a resource type
func GetNamingRules(resourceType string) (AWSNamingRules, error) {
	rules, exists := AWSResourceRules[resourceType]
	if !exists {
		return AWSNamingRules{}, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return rules, nil
}

// formatLambdaName formats a name for Lambda functions
func formatLambdaName(input string) string {
	// Remove invalid characters
	name := regexp.MustCompile(`[^a-zA-Z0-9-_]`).ReplaceAllString(input, "-")

	// Remove multiple consecutive dashes/underscores
	name = regexp.MustCompile(`[-_]{2,}`).ReplaceAllString(name, "-")

	// Trim dashes/underscores from ends
	name = strings.Trim(name, "-_")

	// Ensure starts with letter or number
	if matched, _ := regexp.MatchString(`^[^a-zA-Z0-9]`, name); matched {
		name = "lambda-" + name
	}

	// Truncate to max length
	if len(name) > 64 {
		name = name[:64]
		// Ensure doesn't end with dash/underscore after truncation
		name = strings.TrimRight(name, "-_")
	}

	// Ensure minimum length
	if len(name) == 0 {
		name = "lambda-function"
	}

	return name
}

// formatS3Name formats a name for S3 buckets
func formatS3Name(input string) string {
	// Convert to lowercase
	name := strings.ToLower(input)

	// Replace invalid characters with dashes
	name = regexp.MustCompile(`[^a-z0-9.-]`).ReplaceAllString(name, "-")

	// Remove consecutive dots/dashes
	name = regexp.MustCompile(`[-]{2,}`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`[.]{2,}`).ReplaceAllString(name, ".")

	// Remove dot-dash and dash-dot combinations
	name = regexp.MustCompile(`\.-|-\.`).ReplaceAllString(name, "-")

	// Ensure valid start/end (no dots or dashes)
	name = strings.Trim(name, "-.")

	// Ensure starts and ends with alphanumeric
	if matched, _ := regexp.MatchString(`^[^a-z0-9]`, name); matched {
		name = "bucket-" + name
	}
	if matched, _ := regexp.MatchString(`[^a-z0-9]$`, name); matched {
		name = name + "-bucket"
	}

	// Ensure minimum length
	if len(name) < 3 {
		name = "genesys-" + name
		if len(name) < 3 {
			name = "genesys-bucket"
		}
	}

	// Truncate to max length
	if len(name) > 63 {
		name = name[:63]
		// Ensure doesn't end with dash/dot after truncation
		name = strings.TrimRight(name, "-.")
	}

	return name
}

// formatEC2Name formats a name for EC2 instances
func formatEC2Name(input string) string {
	// Remove invalid characters but keep spaces
	name := regexp.MustCompile(`[^a-zA-Z0-9 ._-]`).ReplaceAllString(input, "-")

	// Clean up multiple spaces and special characters
	name = regexp.MustCompile(`\s{2,}`).ReplaceAllString(name, " ")
	name = regexp.MustCompile(`[-_.]{2,}`).ReplaceAllString(name, "-")

	// Trim spaces and special characters from ends
	name = strings.Trim(name, " -_.")

	// Ensure minimum length
	if len(name) == 0 {
		name = "EC2-Instance"
	}

	// Truncate to max length
	if len(name) > 255 {
		name = name[:255]
		name = strings.TrimRight(name, " -_.")
	}

	return name
}

// formatIAMRoleName formats a name for IAM roles
func formatIAMRoleName(input string) string {
	// Remove invalid characters
	name := regexp.MustCompile(`[^a-zA-Z0-9+=,.@_-]`).ReplaceAllString(input, "-")

	// Clean up multiple dashes
	name = regexp.MustCompile(`[-]{2,}`).ReplaceAllString(name, "-")

	// Trim dashes from ends
	name = strings.Trim(name, "-")

	// Ensure starts with letter or number
	if matched, _ := regexp.MatchString(`^[^a-zA-Z0-9]`, name); matched {
		name = "genesys-" + name
	}

	// Ensure minimum length
	if len(name) == 0 {
		name = "genesys-role"
	}

	// Truncate to max length
	if len(name) > 64 {
		name = name[:64]
		name = strings.TrimRight(name, "-")
	}

	return name
}

// formatIAMPolicyName formats a name for IAM policies
func formatIAMPolicyName(input string) string {
	// Remove invalid characters
	name := regexp.MustCompile(`[^a-zA-Z0-9+=,.@_-]`).ReplaceAllString(input, "-")

	// Clean up multiple dashes
	name = regexp.MustCompile(`[-]{2,}`).ReplaceAllString(name, "-")

	// Trim dashes from ends
	name = strings.Trim(name, "-")

	// Ensure starts with letter or number
	if matched, _ := regexp.MatchString(`^[^a-zA-Z0-9]`, name); matched {
		name = "genesys-" + name
	}

	// Ensure minimum length
	if len(name) == 0 {
		name = "genesys-policy"
	}

	// Truncate to max length
	if len(name) > 128 {
		name = name[:128]
		name = strings.TrimRight(name, "-")
	}

	return name
}

// IsValidName checks if a name is valid for the given resource type without formatting
func IsValidName(resourceType, name string) error {
	rules, exists := AWSResourceRules[resourceType]
	if !exists {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	if name == "" {
		return fmt.Errorf("%s name cannot be empty", rules.Resource)
	}

	if len(name) < rules.MinLength {
		return fmt.Errorf("%s name must be at least %d characters", rules.Resource, rules.MinLength)
	}

	if len(name) > rules.MaxLength {
		return fmt.Errorf("%s name must be at most %d characters", rules.Resource, rules.MaxLength)
	}

	if !rules.Pattern.MatchString(name) {
		return fmt.Errorf("%s name contains invalid characters. %s", rules.Resource, rules.Description)
	}

	return nil
}

// S3BucketNameError represents a detailed error for S3 bucket name validation
type S3BucketNameError struct {
	BucketName string
	Reason     string
	Suggestion string
}

func (e *S3BucketNameError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s (suggestion: %s)", e.Reason, e.Suggestion)
	}
	return e.Reason
}

// ValidateS3BucketName validates an S3 bucket name with detailed error messages and suggestions
func ValidateS3BucketName(name string) error {
	if name == "" {
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot be empty",
			Suggestion: "genesys-bucket-" + time.Now().Format("20060102"),
		}
	}

	// Check length constraints
	if len(name) < 3 {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name must be at least 3 characters",
			Suggestion: suggestion,
		}
	}

	if len(name) > 63 {
		suggestion := formatS3Name(name[:63])
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name must be at most 63 characters",
			Suggestion: suggestion,
		}
	}

	// Check if name contains uppercase letters
	if name != strings.ToLower(name) {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name must be lowercase",
			Suggestion: suggestion,
		}
	}

	// Check if starts or ends with valid characters
	if matched, _ := regexp.MatchString(`^[^a-z0-9]`, name); matched {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name must start with a lowercase letter or number",
			Suggestion: suggestion,
		}
	}

	if matched, _ := regexp.MatchString(`[^a-z0-9]$`, name); matched {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name must end with a lowercase letter or number",
			Suggestion: suggestion,
		}
	}

	// Check for IP address format (not allowed)
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+\.\d+$`, name); matched {
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot be formatted as an IP address",
			Suggestion: "genesys-" + strings.ReplaceAll(name, ".", "-"),
		}
	}

	// Check for invalid characters
	if matched, _ := regexp.MatchString(`[^a-z0-9.-]`, name); matched {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name can only contain lowercase letters, numbers, dots, and hyphens",
			Suggestion: suggestion,
		}
	}

	// Check for consecutive dots or dot-dash combinations
	if strings.Contains(name, "..") || strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		suggestion := formatS3Name(name)
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot contain consecutive dots or dot-dash combinations",
			Suggestion: suggestion,
		}
	}

	// Check for reserved prefixes
	if strings.HasPrefix(name, "xn--") {
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot start with 'xn--' prefix",
			Suggestion: "genesys-" + name,
		}
	}

	if strings.HasPrefix(name, "sthree-") {
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot start with 'sthree-' prefix",
			Suggestion: "genesys-" + name,
		}
	}

	// Check for reserved suffixes
	if strings.HasSuffix(name, "-s3alias") || strings.HasSuffix(name, "--ol-s3") {
		return &S3BucketNameError{
			BucketName: name,
			Reason:     "bucket name cannot end with reserved suffixes like '-s3alias' or '--ol-s3'",
			Suggestion: strings.TrimSuffix(strings.TrimSuffix(name, "-s3alias"), "--ol-s3") + "-bucket",
		}
	}

	return nil
}
