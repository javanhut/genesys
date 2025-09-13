package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// S3BucketNameError represents a bucket name validation error
type S3BucketNameError struct {
	BucketName string
	Reason     string
	Suggestion string
}

func (e *S3BucketNameError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("invalid S3 bucket name '%s': %s. Suggestion: %s", e.BucketName, e.Reason, e.Suggestion)
	}
	return fmt.Sprintf("invalid S3 bucket name '%s': %s", e.BucketName, e.Reason)
}

// ValidateS3BucketName validates an S3 bucket name according to AWS rules
func ValidateS3BucketName(bucketName string) error {
	if bucketName == "" {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot be empty",
			Suggestion: "provide a valid bucket name",
		}
	}

	// Rule 1: Length must be 3-63 characters
	if len(bucketName) < 3 {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name must be at least 3 characters",
			Suggestion: fmt.Sprintf("try '%s-bucket'", bucketName),
		}
	}
	if len(bucketName) > 63 {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name must be 63 characters or less",
			Suggestion: bucketName[:60] + "...",
		}
	}

	// Rule 2: Only lowercase letters, numbers, hyphens, and periods
	validChars := regexp.MustCompile(`^[a-z0-9.-]+$`)
	if !validChars.MatchString(bucketName) {
		suggestion := strings.ToLower(regexp.MustCompile(`[^a-z0-9.-]`).ReplaceAllString(bucketName, "-"))
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name can only contain lowercase letters, numbers, hyphens, and periods",
			Suggestion: suggestion,
		}
	}

	// Rule 3: Must start and end with letter or number
	if !regexp.MustCompile(`^[a-z0-9]`).MatchString(bucketName) {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name must start with a lowercase letter or number",
			Suggestion: "a" + bucketName,
		}
	}
	if !regexp.MustCompile(`[a-z0-9]$`).MatchString(bucketName) {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name must end with a lowercase letter or number",
			Suggestion: bucketName + "1",
		}
	}

	// Rule 4: No consecutive periods
	if strings.Contains(bucketName, "..") {
		suggestion := regexp.MustCompile(`\.\.+`).ReplaceAllString(bucketName, ".")
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot contain consecutive periods",
			Suggestion: suggestion,
		}
	}

	// Rule 5: No period-hyphen combinations
	if strings.Contains(bucketName, ".-") || strings.Contains(bucketName, "-.") {
		suggestion := strings.ReplaceAll(strings.ReplaceAll(bucketName, ".-", "-"), "-.", "-")
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot contain '.-' or '-.' patterns",
			Suggestion: suggestion,
		}
	}

	// Rule 6: Cannot look like an IP address
	if net.ParseIP(bucketName) != nil {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot be formatted as an IP address",
			Suggestion: fmt.Sprintf("bucket-%s", bucketName),
		}
	}

	// Rule 7: Cannot start with 'xn--' (reserved for internationalized domain names)
	if strings.HasPrefix(bucketName, "xn--") {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot start with 'xn--' (reserved prefix)",
			Suggestion: strings.Replace(bucketName, "xn--", "", 1),
		}
	}

	// Rule 8: Cannot end with '-s3alias' (reserved suffix)
	if strings.HasSuffix(bucketName, "-s3alias") {
		return &S3BucketNameError{
			BucketName: bucketName,
			Reason:     "bucket name cannot end with '-s3alias' (reserved suffix)",
			Suggestion: strings.Replace(bucketName, "-s3alias", "", 1),
		}
	}

	return nil
}

// ValidateAndSuggestS3BucketName validates a bucket name and provides suggestions if invalid
func ValidateAndSuggestS3BucketName(bucketName string) (string, error) {
	err := ValidateS3BucketName(bucketName)
	if err != nil {
		if s3Err, ok := err.(*S3BucketNameError); ok && s3Err.Suggestion != "" {
			// Return the suggestion as a valid alternative
			if ValidateS3BucketName(s3Err.Suggestion) == nil {
				return s3Err.Suggestion, err
			}
		}
		return bucketName, err
	}
	return bucketName, nil
}

// GenerateUniqueBucketName generates a unique bucket name with timestamp
func GenerateUniqueBucketName(baseName string) (string, error) {
	// Clean the base name
	suggestion, err := ValidateAndSuggestS3BucketName(baseName)
	if err != nil {
		if s3Err, ok := err.(*S3BucketNameError); ok && s3Err.Suggestion != "" {
			baseName = s3Err.Suggestion
		} else {
			// If no suggestion, create a safe base name
			baseName = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(strings.ToLower(baseName), "-")
			if baseName == "" {
				baseName = "bucket"
			}
		}
	} else {
		baseName = suggestion
	}

	// Add timestamp to make it unique
	timestamp := time.Now().Unix()
	uniqueName := fmt.Sprintf("%s-%d", baseName, timestamp)

	// Ensure it's still valid after adding timestamp
	if len(uniqueName) > 63 {
		// Trim base name to fit
		maxBaseLength := 63 - len(fmt.Sprintf("-%d", timestamp))
		if maxBaseLength < 1 {
			baseName = "bucket"
		} else {
			baseName = baseName[:maxBaseLength]
		}
		uniqueName = fmt.Sprintf("%s-%d", baseName, timestamp)
	}

	return uniqueName, ValidateS3BucketName(uniqueName)
}

// IsS3BucketNameAvailable checks if a bucket name is likely available (basic heuristics)
func IsS3BucketNameAvailable(bucketName string) bool {
	// Basic heuristics for common unavailable names
	commonNames := []string{
		"test", "test-bucket", "my-bucket", "bucket", "demo", "example",
		"sample", "temp", "tmp", "data", "backup", "storage", "files",
		"uploads", "downloads", "images", "photos", "videos", "docs",
		"www", "app", "api", "web", "site", "admin", "user", "users",
	}

	bucketLower := strings.ToLower(bucketName)
	for _, common := range commonNames {
		if bucketLower == common || bucketLower == common+"-bucket" {
			return false
		}
	}

	// Names that are too generic are likely taken
	if len(bucketName) < 8 && !strings.Contains(bucketName, "-") {
		return false
	}

	return true
}

// SuggestUniqueBucketName suggests a unique bucket name based on user input
func SuggestUniqueBucketName(baseName string) string {
	// If the name is likely available, return it
	if IsS3BucketNameAvailable(baseName) {
		validName, err := ValidateAndSuggestS3BucketName(baseName)
		if err == nil {
			return validName
		}
	}

	// Generate unique name with timestamp
	uniqueName, err := GenerateUniqueBucketName(baseName)
	if err == nil {
		return uniqueName
	}

	// Fallback to a completely safe name
	timestamp := time.Now().Unix()
	return fmt.Sprintf("genesys-bucket-%d", timestamp)
}