package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// ComputeService implements AWS EC2 operations using direct API calls
type ComputeService struct {
	provider    *AWSProvider
	amiResolver *AMIResolver
}

// NewComputeService creates a new compute service
func NewComputeService(p *AWSProvider) *ComputeService {
	return &ComputeService{
		provider: p,
	}
}

// initAMIResolver initializes the AMI resolver if not already done
func (c *ComputeService) initAMIResolver() error {
	if c.amiResolver != nil {
		return nil
	}

	c.amiResolver = NewAMIResolver(c.provider, c.provider.region)
	return nil
}

// EC2 API response structures
type RunInstancesResponse struct {
	XMLName   xml.Name `xml:"RunInstancesResponse"`
	Instances struct {
		Items []EC2Instance `xml:"item"`
	} `xml:"instancesSet"`
}

type DescribeInstancesResponse struct {
	XMLName      xml.Name `xml:"DescribeInstancesResponse"`
	Reservations struct {
		Items []struct {
			Instances struct {
				Items []EC2Instance `xml:"item"`
			} `xml:"instances"`
		} `xml:"item"`
	} `xml:"reservationSet"`
}

type EC2Instance struct {
	InstanceId   string `xml:"instanceId"`
	InstanceType string `xml:"instanceType"`
	State        struct {
		Name string `xml:"name"`
	} `xml:"state"`
	PrivateIpAddress string `xml:"privateIpAddress"`
	LaunchTime       string `xml:"launchTime"`
	Tags             struct {
		Items []struct {
			Key   string `xml:"key"`
			Value string `xml:"value"`
		} `xml:"item"`
	} `xml:"tagSet"`
}

// EC2Error represents an EC2 API error response
type EC2Error struct {
	XMLName xml.Name `xml:"Response"`
	Errors  struct {
		Error []struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	} `xml:"Errors"`
	RequestID string `xml:"RequestID"`
}

// parseEC2Error extracts a clean error message from EC2 XML error response
func parseEC2Error(responseBody []byte) string {
	var ec2Err EC2Error
	if err := xml.Unmarshal(responseBody, &ec2Err); err != nil {
		// If we can't parse the XML, return the raw response
		return string(responseBody)
	}
	
	// Return a clean, user-friendly error message
	if len(ec2Err.Errors.Error) > 0 {
		firstError := ec2Err.Errors.Error[0]
		if firstError.Code != "" && firstError.Message != "" {
			return fmt.Sprintf("%s: %s", firstError.Code, firstError.Message)
		}
		if firstError.Message != "" {
			return firstError.Message
		}
	}
	
	// Last resort fallback
	return string(responseBody)
}

// isValidAMIID checks if an AMI ID has the correct format
func isValidAMIID(amiID string) bool {
	// AMI IDs should start with "ami-" and be followed by 17 hex characters (total length 21)
	if len(amiID) != 21 {
		return false
	}
	if !strings.HasPrefix(amiID, "ami-") {
		return false
	}
	// Check if the remaining 17 characters are valid hex
	suffix := amiID[4:]
	for _, char := range suffix {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

// CreateInstance creates a new EC2 instance
func (c *ComputeService) CreateInstance(ctx context.Context, config *provider.InstanceConfig) (*provider.Instance, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	// Map instance types
	instanceType := c.mapInstanceType(string(config.Type))
	
	// Initialize AMI resolver
	if err := c.initAMIResolver(); err != nil {
		return nil, fmt.Errorf("failed to initialize AMI resolver: %w", err)
	}

	// Get AMI ID using dynamic resolution
	amiId, err := c.amiResolver.ResolveAMI(ctx, config.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve AMI for image %s: %w", config.Image, err)
	}

	// Build parameters (default to 1 instance for now)
	params := map[string]string{
		"Action":       "RunInstances",
		"Version":      "2016-11-15",
		"ImageId":      amiId,
		"MinCount":     "1",
		"MaxCount":     "1",
		"InstanceType": instanceType,
	}

	// Add tags (always include Name tag, so we always have at least one tag)
	params[fmt.Sprintf("TagSpecification.1.ResourceType")] = "instance"
	tagIndex := 1
	
	// Add user-defined tags
	for key, value := range config.Tags {
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIndex)] = value
		tagIndex++
	}

	// Add Name tag (always present)
	params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIndex)] = "Name"
	params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIndex)] = config.Name

	// Make the request
	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to run instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("RunInstances failed: %s", cleanError)
	}

	// Parse response
	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var runResp RunInstancesResponse
	if err := xml.Unmarshal(body, &runResp); err != nil {
		// Debug: print the raw response to understand the structure
		fmt.Printf("DEBUG: Failed to parse RunInstances response. Raw response:\n%s\n", string(body))
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(runResp.Instances.Items) == 0 {
		// Debug: print what we actually got
		fmt.Printf("DEBUG: RunInstances response parsed but no instances found. Response structure: %+v\n", runResp)
		return nil, fmt.Errorf("no instances created - response parsed but instances list is empty")
	}

	// Convert to provider instance
	ec2Instance := runResp.Instances.Items[0]
	return c.convertToProviderInstance(ec2Instance), nil
}

// GetInstance retrieves an instance by ID
func (c *ComputeService) GetInstance(ctx context.Context, id string) (*provider.Instance, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":      "DescribeInstances",
		"Version":     "2016-11-15",
		"InstanceId.1": id,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeInstances failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeInstancesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for _, reservation := range descResp.Reservations.Items {
		for _, instance := range reservation.Instances.Items {
			if instance.InstanceId == id {
				return c.convertToProviderInstance(instance), nil
			}
		}
	}

	return nil, fmt.Errorf("instance %s not found", id)
}

// UpdateInstance updates an instance configuration
func (c *ComputeService) UpdateInstance(ctx context.Context, id string, config *provider.InstanceConfig) error {
	// For now, we only support updating tags
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":       "CreateTags",
		"Version":      "2016-11-15",
		"ResourceId.1": id,
	}

	tagIndex := 1
	for key, value := range config.Tags {
		params[fmt.Sprintf("Tag.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("Tag.%d.Value", tagIndex)] = value
		tagIndex++
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to create tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return fmt.Errorf("CreateTags failed: %s", cleanError)
	}

	return nil
}

// DeleteInstance terminates an instance
func (c *ComputeService) DeleteInstance(ctx context.Context, id string) error {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":      "TerminateInstances",
		"Version":     "2016-11-15",
		"InstanceId.1": id,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to terminate instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return fmt.Errorf("TerminateInstances failed: %s", cleanError)
	}

	return nil
}

// ListInstances lists instances with optional filters
func (c *ComputeService) ListInstances(ctx context.Context, filters map[string]string) ([]*provider.Instance, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeInstances",
		"Version": "2016-11-15",
	}

	// Add filters
	filterIndex := 1
	for key, value := range filters {
		params[fmt.Sprintf("Filter.%d.Name", filterIndex)] = key
		params[fmt.Sprintf("Filter.%d.Value.1", filterIndex)] = value
		filterIndex++
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeInstances failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeInstancesResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var instances []*provider.Instance
	for _, reservation := range descResp.Reservations.Items {
		for _, instance := range reservation.Instances.Items {
			instances = append(instances, c.convertToProviderInstance(instance))
		}
	}

	return instances, nil
}

// DiscoverInstances discovers existing instances
func (c *ComputeService) DiscoverInstances(ctx context.Context) ([]*provider.Instance, error) {
	return c.ListInstances(ctx, map[string]string{
		"instance-state-name": "running",
	})
}

// AdoptInstance adopts an existing instance into management
func (c *ComputeService) AdoptInstance(ctx context.Context, id string) (*provider.Instance, error) {
	return c.GetInstance(ctx, id)
}

// Helper methods

func (c *ComputeService) mapInstanceType(instanceType string) string {
	switch instanceType {
	case "small":
		return "t3.small"
	case "medium":
		return "t3.medium"
	case "large":
		return "t3.large"
	case "xlarge":
		return "t3.xlarge"
	// Free Tier eligible instance types
	case "t2.micro":
		return "t2.micro"
	case "t2.small":
		return "t2.small"
	case "t3.micro":
		return "t3.micro"
	case "t3.nano":
		return "t3.nano"
	case "c7i-flex.large":
		return "c7i-flex.large"
	case "m7i-flex.large":
		return "m7i-flex.large"
	default:
		return "t3.micro" // Default to Free Tier
	}
}

func (c *ComputeService) getAMIForImage(client *AWSClient, image string) (string, error) {
	// Initialize AMI resolver if not already done
	if err := c.initAMIResolver(); err != nil {
		return "", fmt.Errorf("failed to initialize AMI resolver: %w", err)
	}

	// Use the AMI resolver for dynamic lookup
	ctx := context.Background()
	amiID, err := c.amiResolver.ResolveAMI(ctx, image)
	if err != nil {
		return "", fmt.Errorf("failed to resolve AMI for image '%s': %w", image, err)
	}

	return amiID, nil
}

func (c *ComputeService) convertToProviderInstance(ec2Instance EC2Instance) *provider.Instance {
	tags := make(map[string]string)
	var name string
	
	for _, tag := range ec2Instance.Tags.Items {
		tags[tag.Key] = tag.Value
		if tag.Key == "Name" {
			name = tag.Value
		}
	}

	createdAt := time.Now()
	if ec2Instance.LaunchTime != "" {
		if t, err := time.Parse(time.RFC3339, ec2Instance.LaunchTime); err == nil {
			createdAt = t
		}
	}

	return &provider.Instance{
		ID:        ec2Instance.InstanceId,
		Name:      name,
		Type:      provider.InstanceType(c.reverseMapInstanceType(ec2Instance.InstanceType)),
		State:     ec2Instance.State.Name,
		PrivateIP: ec2Instance.PrivateIpAddress,
		Tags:      tags,
		CreatedAt: createdAt,
	}
}

func (c *ComputeService) reverseMapInstanceType(awsType string) string {
	switch awsType {
	case "t3.small":
		return "small"
	case "t3.medium":
		return "medium"
	case "t3.large":
		return "large"
	case "t3.xlarge":
		return "xlarge"
	default:
		return strings.TrimPrefix(awsType, "t3.")
	}
}