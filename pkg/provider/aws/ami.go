package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AMIResolver handles dynamic AMI resolution with multiple strategies
type AMIResolver struct {
	provider         AWSProviderInterface
	region           string
	cache            map[string]*AMICacheEntry
	cacheTTL         time.Duration
	strategy         string // "auto", "ssm", "describe", "static"
	disableCache     bool
	fallbackToStatic bool
}

// AWSProviderInterface defines the interface needed by AMI resolver
type AWSProviderInterface interface {
	CreateClient(service string) (*AWSClient, error)
}

// AMIResolverConfig holds configuration options for the AMI resolver
type AMIResolverConfig struct {
	Strategy         string        // "auto", "ssm", "describe", "static"
	CacheTTL         time.Duration // Cache time-to-live
	DisableCache     bool          // Disable caching entirely
	FallbackToStatic bool          // Fall back to static mappings if dynamic lookup fails
}

// AMICacheEntry represents a cached AMI lookup result
type AMICacheEntry struct {
	AMIID     string
	Timestamp time.Time
	Source    string // "ssm", "describe", "static"
}

// NewAMIResolver creates a new AMI resolver with default configuration
func NewAMIResolver(provider AWSProviderInterface, region string) *AMIResolver {
	return NewAMIResolverWithConfig(provider, region, AMIResolverConfig{
		Strategy:         "auto",
		CacheTTL:         24 * time.Hour,
		DisableCache:     false,
		FallbackToStatic: true,
	})
}

// NewAMIResolverWithConfig creates a new AMI resolver with custom configuration
func NewAMIResolverWithConfig(provider AWSProviderInterface, region string, config AMIResolverConfig) *AMIResolver {
	return &AMIResolver{
		provider:         provider,
		region:           region,
		cache:            make(map[string]*AMICacheEntry),
		cacheTTL:         config.CacheTTL,
		strategy:         config.Strategy,
		disableCache:     config.DisableCache,
		fallbackToStatic: config.FallbackToStatic,
	}
}

// ResolveAMI resolves an image name/type to an actual AMI ID using multiple strategies
func (r *AMIResolver) ResolveAMI(ctx context.Context, image string) (string, error) {
	// If it's already an AMI ID, validate and return
	if strings.HasPrefix(image, "ami-") && len(image) == 21 {
		if isValidAMIID(image) {
			return image, nil
		}
		return "", fmt.Errorf("invalid AMI ID format: %s", image)
	}

	// Check cache first (unless disabled)
	cacheKey := fmt.Sprintf("%s-%s", r.region, image)
	if !r.disableCache {
		if entry, exists := r.cache[cacheKey]; exists {
			if time.Since(entry.Timestamp) < r.cacheTTL {
				return entry.AMIID, nil
			}
			// Cache expired, remove entry
			delete(r.cache, cacheKey)
		}
	}

	// Apply resolution strategy
	switch r.strategy {
	case "ssm":
		// Only try SSM Parameter Store
		if amiID, err := r.lookupFromSSM(ctx, image); err == nil && amiID != "" {
			r.cacheAMI(cacheKey, amiID, "ssm")
			return amiID, nil
		}
	case "describe":
		// Only try DescribeImages API
		if amiID, err := r.lookupFromDescribeImages(ctx, image); err == nil && amiID != "" {
			r.cacheAMI(cacheKey, amiID, "describe")
			return amiID, nil
		}
	case "static":
		// Only use static mappings
		if amiID := r.getStaticAMI(image); amiID != "" {
			r.cacheAMI(cacheKey, amiID, "static")
			return amiID, nil
		}
	default: // "auto" or any other value
		// Try all strategies in order

		// Strategy 1: Try AWS Systems Manager Parameter Store
		if amiID, err := r.lookupFromSSM(ctx, image); err == nil && amiID != "" {
			r.cacheAMI(cacheKey, amiID, "ssm")
			return amiID, nil
		}

		// Strategy 2: Try DescribeImages API
		if amiID, err := r.lookupFromDescribeImages(ctx, image); err == nil && amiID != "" {
			r.cacheAMI(cacheKey, amiID, "describe")
			return amiID, nil
		}

		// Strategy 3: Fall back to static mappings (if enabled)
		if r.fallbackToStatic {
			if amiID := r.getStaticAMI(image); amiID != "" {
				r.cacheAMI(cacheKey, amiID, "static")
				return amiID, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve AMI for image: %s in region: %s", image, r.region)
}

// lookupFromSSM attempts to get AMI ID from AWS Systems Manager Parameter Store
func (r *AMIResolver) lookupFromSSM(ctx context.Context, image string) (string, error) {
	// AWS publishes current AMI IDs in SSM Parameter Store
	parameterMap := map[string]string{
		"ubuntu-lts":        "/aws/service/canonical/ubuntu/server/20.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"ubuntu":            "/aws/service/canonical/ubuntu/server/20.04/stable/current/amd64/hvm/ebs-gp2/ami-id",
		"amazon-linux":      "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-6.1-x86_64",
		"amzn2":             "/aws/service/ami-amazon-linux-latest/amzn2-ami-hvm-x86_64-gp2",
		"amazon-linux-2023": "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-6.1-x86_64",
	}

	parameterName, exists := parameterMap[strings.ToLower(image)]
	if !exists {
		return "", fmt.Errorf("no SSM parameter mapping for image: %s", image)
	}

	// Call SSM GetParameter
	params := map[string]string{
		"Action": "GetParameter",
		"Name":   parameterName,
	}

	// Create SSM client (different service endpoint)
	ssmClient, err := r.provider.CreateClient("ssm")
	if err != nil {
		return "", fmt.Errorf("failed to create SSM client: %w", err)
	}

	resp, err := ssmClient.Request("POST", "/", params, nil)
	if err != nil {
		return "", fmt.Errorf("SSM GetParameter failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// SSM not available or parameter not found - this is not an error, just means we try next strategy
		return "", fmt.Errorf("SSM parameter not found: %s", parameterName)
	}

	// Parse SSM response
	body, err := ReadResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read SSM response: %w", err)
	}

	var ssmResp SSMGetParameterResponse
	if err := xml.Unmarshal(body, &ssmResp); err != nil {
		return "", fmt.Errorf("failed to parse SSM response: %w", err)
	}

	amiID := ssmResp.Parameter.Value
	if !isValidAMIID(amiID) {
		return "", fmt.Errorf("invalid AMI ID from SSM: %s", amiID)
	}

	return amiID, nil
}

// lookupFromDescribeImages attempts to find the latest AMI using DescribeImages API
func (r *AMIResolver) lookupFromDescribeImages(ctx context.Context, image string) (string, error) {
	// Define search filters for different image types
	filterMap := map[string]map[string]string{
		"ubuntu-lts": {
			"name":         "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*",
			"owner-id":     "099720109477", // Canonical
			"architecture": "x86_64",
		},
		"ubuntu": {
			"name":         "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*",
			"owner-id":     "099720109477", // Canonical
			"architecture": "x86_64",
		},
		"amazon-linux": {
			"name":         "amzn2-ami-hvm-*-x86_64-gp2",
			"owner-id":     "137112412989", // Amazon
			"architecture": "x86_64",
		},
		"amzn2": {
			"name":         "amzn2-ami-hvm-*-x86_64-gp2",
			"owner-id":     "137112412989", // Amazon
			"architecture": "x86_64",
		},
	}

	filters, exists := filterMap[strings.ToLower(image)]
	if !exists {
		return "", fmt.Errorf("no DescribeImages filters for image: %s", image)
	}

	// Build DescribeImages request
	params := map[string]string{
		"Action": "DescribeImages",
	}

	// Add filters
	filterIndex := 1
	for key, value := range filters {
		params[fmt.Sprintf("Filter.%d.Name", filterIndex)] = key
		params[fmt.Sprintf("Filter.%d.Value.1", filterIndex)] = value
		filterIndex++
	}

	// Add state filter to only get available images
	params[fmt.Sprintf("Filter.%d.Name", filterIndex)] = "state"
	params[fmt.Sprintf("Filter.%d.Value.1", filterIndex)] = "available"

	// Create EC2 client for this request
	ec2Client, err := r.provider.CreateClient("ec2")
	if err != nil {
		return "", fmt.Errorf("failed to create EC2 client: %w", err)
	}

	resp, err := ec2Client.Request("POST", "/", params, nil)
	if err != nil {
		return "", fmt.Errorf("DescribeImages failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return "", fmt.Errorf("DescribeImages API error: %s", parseEC2Error(body))
	}

	// Parse DescribeImages response
	body, err := ReadResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read DescribeImages response: %w", err)
	}

	var descResp DescribeImagesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return "", fmt.Errorf("failed to parse DescribeImages response: %w", err)
	}

	if len(descResp.Images.Items) == 0 {
		return "", fmt.Errorf("no AMIs found for image: %s", image)
	}

	// Sort by creation date (newest first)
	sort.Slice(descResp.Images.Items, func(i, j int) bool {
		return descResp.Images.Items[i].CreationDate > descResp.Images.Items[j].CreationDate
	})

	// Return the newest AMI
	newestAMI := descResp.Images.Items[0]
	if !isValidAMIID(newestAMI.ImageId) {
		return "", fmt.Errorf("invalid AMI ID from DescribeImages: %s", newestAMI.ImageId)
	}

	return newestAMI.ImageId, nil
}

// getStaticAMI returns a static AMI mapping as last resort
func (r *AMIResolver) getStaticAMI(image string) string {
	// These should be updated regularly - consider them emergency fallbacks only
	staticMappings := map[string]map[string]string{
		"ubuntu-lts": {
			"us-east-1":      "ami-0e2c8caa4b6378d8c",
			"us-east-2":      "ami-0ea3c35c5c3284d82",
			"us-west-1":      "ami-0827b3b7dcd39002a",
			"us-west-2":      "ami-0aff18ec83b712f05",
			"eu-west-1":      "ami-0694d931cee176e7d",
			"eu-central-1":   "ami-04e601abe3e1a910f",
			"ap-southeast-1": "ami-0df7a207adb9748c7",
			"ap-northeast-1": "ami-0f36dcfcc94112ea1",
		},
		"amazon-linux": {
			"us-east-1":      "ami-0e2c8caa4b6378d8c",
			"us-east-2":      "ami-0ea3c35c5c3284d82",
			"us-west-1":      "ami-0827b3b7dcd39002a",
			"us-west-2":      "ami-0aff18ec83b712f05",
			"eu-west-1":      "ami-0694d931cee176e7d",
			"eu-central-1":   "ami-04e601abe3e1a910f",
			"ap-southeast-1": "ami-0df7a207adb9748c7",
			"ap-northeast-1": "ami-0f36dcfcc94112ea1",
		},
	}

	// Normalize image name
	normalizedImage := strings.ToLower(image)
	switch normalizedImage {
	case "ubuntu", "ubuntu-lts":
		normalizedImage = "ubuntu-lts"
	case "amazon-linux", "amzn2":
		normalizedImage = "amazon-linux"
	default:
		normalizedImage = "ubuntu-lts" // Default fallback
	}

	if regionMap, exists := staticMappings[normalizedImage]; exists {
		if amiID, exists := regionMap[r.region]; exists {
			return amiID
		}
		// Fall back to us-east-1
		if amiID, exists := regionMap["us-east-1"]; exists {
			return amiID
		}
	}

	// Ultimate fallback
	return "ami-0c02fb55956c7d316"
}

// cacheAMI stores an AMI ID in the cache (if caching is enabled)
func (r *AMIResolver) cacheAMI(key, amiID, source string) {
	if !r.disableCache {
		r.cache[key] = &AMICacheEntry{
			AMIID:     amiID,
			Timestamp: time.Now(),
			Source:    source,
		}
	}
}

// ClearCache clears the AMI cache (useful for testing or forced refresh)
func (r *AMIResolver) ClearCache() {
	r.cache = make(map[string]*AMICacheEntry)
}

// RefreshCache clears expired entries from the cache
func (r *AMIResolver) RefreshCache() {
	for key, entry := range r.cache {
		if time.Since(entry.Timestamp) > r.cacheTTL {
			delete(r.cache, key)
		}
	}
}

// GetCacheStats returns cache statistics for debugging
func (r *AMIResolver) GetCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["entries"] = len(r.cache)
	stats["ttl_hours"] = r.cacheTTL.Hours()

	sources := make(map[string]int)
	expired := 0
	for _, entry := range r.cache {
		sources[entry.Source]++
		if time.Since(entry.Timestamp) > r.cacheTTL {
			expired++
		}
	}
	stats["sources"] = sources
	stats["expired"] = expired

	return stats
}

// GetAMIDetails retrieves details about an AMI by its ID
func (r *AMIResolver) GetAMIDetails(ctx context.Context, amiID string) (*AMIImage, error) {
	if !isValidAMIID(amiID) {
		return nil, fmt.Errorf("invalid AMI ID: %s", amiID)
	}

	ec2Client, err := r.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":    "DescribeImages",
		"ImageId.1": amiID,
	}

	resp, err := ec2Client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("DescribeImages failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeImages API error: %s", parseEC2Error(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read DescribeImages response: %w", err)
	}

	var descResp DescribeImagesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse DescribeImages response: %w", err)
	}

	if len(descResp.Images.Items) == 0 {
		return nil, fmt.Errorf("AMI not found: %s", amiID)
	}

	return &descResp.Images.Items[0], nil
}

// GetAMIDetailsInRegion retrieves details about an AMI in a specific region
func GetAMIDetailsInRegion(ctx context.Context, amiID, region string) (*AMIImage, error) {
	if !isValidAMIID(amiID) {
		return nil, fmt.Errorf("invalid AMI ID: %s", amiID)
	}

	ec2Client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":    "DescribeImages",
		"ImageId.1": amiID,
	}

	resp, err := ec2Client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("DescribeImages failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeImages API error: %s", parseEC2Error(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read DescribeImages response: %w", err)
	}

	var descResp DescribeImagesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse DescribeImages response: %w", err)
	}

	if len(descResp.Images.Items) == 0 {
		return nil, fmt.Errorf("AMI not found: %s", amiID)
	}

	return &descResp.Images.Items[0], nil
}

// AWS API Response structures

// SSMGetParameterResponse represents SSM GetParameter API response
type SSMGetParameterResponse struct {
	XMLName   xml.Name `xml:"GetParameterResponse"`
	Parameter struct {
		Name  string `xml:"Name"`
		Value string `xml:"Value"`
		Type  string `xml:"Type"`
	} `xml:"GetParameterResult>Parameter"`
}

// DescribeImagesResponse represents DescribeImages API response
type DescribeImagesResponse struct {
	XMLName xml.Name `xml:"DescribeImagesResponse"`
	Images  struct {
		Items []AMIImage `xml:"item"`
	} `xml:"imagesSet"`
}

// AMIImage represents an AMI in DescribeImages response
type AMIImage struct {
	ImageId      string `xml:"imageId"`
	Name         string `xml:"name"`
	Description  string `xml:"description"`
	CreationDate string `xml:"creationDate"`
	State        string `xml:"state"`
	Architecture string `xml:"architecture"`
	OwnerId      string `xml:"imageOwnerId"`
}
