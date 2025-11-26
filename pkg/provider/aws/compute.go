package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// AllAWSRegions contains all standard AWS regions for EC2 discovery
var AllAWSRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"af-south-1",
	"ap-east-1", "ap-south-1", "ap-south-2",
	"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
	"ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4",
	"ca-central-1",
	"eu-central-1", "eu-central-2",
	"eu-west-1", "eu-west-2", "eu-west-3",
	"eu-north-1", "eu-south-1", "eu-south-2",
	"il-central-1",
	"me-south-1", "me-central-1",
	"sa-east-1",
}

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
			} `xml:"instancesSet"`
		} `xml:"item"`
	} `xml:"reservationSet"`
}

type EC2Instance struct {
	InstanceId       string `xml:"instanceId"`
	InstanceType     string `xml:"instanceType"`
	ImageId          string `xml:"imageId"`
	KeyName          string `xml:"keyName"`
	Platform         string `xml:"platform"`
	PrivateIpAddress string `xml:"privateIpAddress"`
	PublicIpAddress  string `xml:"ipAddress"`
	State            struct {
		Name string `xml:"name"`
	} `xml:"instanceState"`
	LaunchTime string `xml:"launchTime"`
	Tags       struct {
		Items []struct {
			Key   string `xml:"key"`
			Value string `xml:"value"`
		} `xml:"item"`
	} `xml:"tagSet"`
	SecurityGroups struct {
		Items []struct {
			GroupId   string `xml:"groupId"`
			GroupName string `xml:"groupName"`
		} `xml:"item"`
	} `xml:"groupSet"`
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

	// Add key pair if specified (required for SSH access)
	if config.KeyPair != "" {
		params["KeyName"] = config.KeyPair
	}

	// Configure network interface for public IP and security groups
	// Using NetworkInterfaces allows us to control public IP assignment
	// Note: When using NetworkInterfaces, SecurityGroupId cannot be specified at top level
	params["NetworkInterface.1.DeviceIndex"] = "0"

	// Associate public IP if requested (critical for SSH access)
	if config.PublicIP {
		params["NetworkInterface.1.AssociatePublicIpAddress"] = "true"
	} else {
		params["NetworkInterface.1.AssociatePublicIpAddress"] = "false"
	}

	// Add subnet if specified
	if config.Subnet != "" {
		params["NetworkInterface.1.SubnetId"] = config.Subnet
	}

	// Add security groups to network interface (not at instance level when using NetworkInterfaces)
	for i, sg := range config.SecurityGroups {
		params[fmt.Sprintf("NetworkInterface.1.SecurityGroupId.%d", i+1)] = sg
	}

	// Add tags (always include Name tag, so we always have at least one tag)
	params["TagSpecification.1.ResourceType"] = "instance"
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
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(runResp.Instances.Items) == 0 {
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
		"Action":       "DescribeInstances",
		"Version":      "2016-11-15",
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
		"Action":       "TerminateInstances",
		"Version":      "2016-11-15",
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

// DiscoverInstances discovers existing instances across all AWS regions
func (c *ComputeService) DiscoverInstances(ctx context.Context) ([]*provider.Instance, error) {
	return c.DiscoverInstancesAllRegions(ctx)
}

// DiscoverInstancesAllRegions queries all AWS regions in parallel to find EC2 instances
func (c *ComputeService) DiscoverInstancesAllRegions(ctx context.Context) ([]*provider.Instance, error) {
	var allInstances []*provider.Instance
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Channel to collect errors (we'll log them but continue)
	errChan := make(chan error, len(AllAWSRegions))

	for _, region := range AllAWSRegions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()

			instances, err := c.listInstancesInRegion(ctx, r)
			if err != nil {
				// Send error to channel but don't fail entire operation
				errChan <- fmt.Errorf("region %s: %w", r, err)
				return
			}

			if len(instances) > 0 {
				mu.Lock()
				allInstances = append(allInstances, instances...)
				mu.Unlock()
			}
		}(region)
	}

	wg.Wait()
	close(errChan)

	// Drain error channel (errors are logged but don't fail the operation)
	// This allows discovery to succeed even if some regions are not enabled
	for range errChan {
		// Silently ignore region errors (e.g., region not enabled for account)
	}

	return allInstances, nil
}

// listInstancesInRegion lists instances in a specific region
func (c *ComputeService) listInstancesInRegion(ctx context.Context, region string) ([]*provider.Instance, error) {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeInstances",
		"Version": "2016-11-15",
	}

	// Filter out terminated instances
	params["Filter.1.Name"] = "instance-state-name"
	params["Filter.1.Value.1"] = "pending"
	params["Filter.1.Value.2"] = "running"
	params["Filter.1.Value.3"] = "stopping"
	params["Filter.1.Value.4"] = "stopped"

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		// Check for AuthFailure which indicates region is not enabled
		if strings.Contains(string(body), "AuthFailure") || strings.Contains(string(body), "OptInRequired") {
			// Region not enabled for this account, return empty list
			return nil, nil
		}
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
			inst := c.convertToProviderInstanceWithRegion(instance, region)
			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// convertToProviderInstanceWithRegion converts EC2 instance to provider instance with region info
func (c *ComputeService) convertToProviderInstanceWithRegion(ec2Instance EC2Instance, region string) *provider.Instance {
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

	// Build ProviderData with SSH-relevant information and region
	providerData := make(map[string]interface{})
	providerData["Region"] = region
	if ec2Instance.KeyName != "" {
		providerData["KeyName"] = ec2Instance.KeyName
	}
	if ec2Instance.ImageId != "" {
		providerData["ImageId"] = ec2Instance.ImageId
	}
	if ec2Instance.Platform != "" {
		providerData["Platform"] = ec2Instance.Platform
	}

	// Add security groups
	if len(ec2Instance.SecurityGroups.Items) > 0 {
		var sgIds []string
		var sgNames []string
		for _, sg := range ec2Instance.SecurityGroups.Items {
			sgIds = append(sgIds, sg.GroupId)
			sgNames = append(sgNames, sg.GroupName)
		}
		providerData["SecurityGroupIds"] = sgIds
		providerData["SecurityGroupNames"] = sgNames
	}

	return &provider.Instance{
		ID:           ec2Instance.InstanceId,
		Name:         name,
		Type:         provider.InstanceType(c.reverseMapInstanceType(ec2Instance.InstanceType)),
		State:        ec2Instance.State.Name,
		PrivateIP:    ec2Instance.PrivateIpAddress,
		PublicIP:     ec2Instance.PublicIpAddress,
		Tags:         tags,
		CreatedAt:    createdAt,
		ProviderData: providerData,
	}
}

// AdoptInstance adopts an existing instance into management
func (c *ComputeService) AdoptInstance(ctx context.Context, id string) (*provider.Instance, error) {
	return c.GetInstance(ctx, id)
}

// Key Pair Management

// KeyPair represents an EC2 key pair
type KeyPair struct {
	KeyName        string
	KeyFingerprint string
	KeyPairId      string
	KeyMaterial    string // Only populated on creation
	Region         string
}

// CreateKeyPairResponse represents the EC2 CreateKeyPair API response
type CreateKeyPairResponse struct {
	XMLName        xml.Name `xml:"CreateKeyPairResponse"`
	KeyName        string   `xml:"keyName"`
	KeyFingerprint string   `xml:"keyFingerprint"`
	KeyPairId      string   `xml:"keyPairId"`
	KeyMaterial    string   `xml:"keyMaterial"`
}

// DescribeKeyPairsResponse represents the EC2 DescribeKeyPairs API response
type DescribeKeyPairsResponse struct {
	XMLName  xml.Name `xml:"DescribeKeyPairsResponse"`
	KeyPairs struct {
		Items []struct {
			KeyName        string `xml:"keyName"`
			KeyFingerprint string `xml:"keyFingerprint"`
			KeyPairId      string `xml:"keyPairId"`
		} `xml:"item"`
	} `xml:"keySet"`
}

// CreateKeyPair creates a new EC2 key pair and returns the private key material
func (c *ComputeService) CreateKeyPair(ctx context.Context, keyName string) (*KeyPair, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "CreateKeyPair",
		"Version": "2016-11-15",
		"KeyName": keyName,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("CreateKeyPair failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var createResp CreateKeyPairResponse
	if err := xml.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &KeyPair{
		KeyName:        createResp.KeyName,
		KeyFingerprint: createResp.KeyFingerprint,
		KeyPairId:      createResp.KeyPairId,
		KeyMaterial:    createResp.KeyMaterial,
		Region:         c.provider.region,
	}, nil
}

// CreateKeyPairInRegion creates a new EC2 key pair in a specific region
func (c *ComputeService) CreateKeyPairInRegion(ctx context.Context, keyName, region string) (*KeyPair, error) {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "CreateKeyPair",
		"Version": "2016-11-15",
		"KeyName": keyName,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("CreateKeyPair failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var createResp CreateKeyPairResponse
	if err := xml.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &KeyPair{
		KeyName:        createResp.KeyName,
		KeyFingerprint: createResp.KeyFingerprint,
		KeyPairId:      createResp.KeyPairId,
		KeyMaterial:    createResp.KeyMaterial,
		Region:         region,
	}, nil
}

// ListKeyPairs lists all key pairs in the default region
func (c *ComputeService) ListKeyPairs(ctx context.Context) ([]*KeyPair, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeKeyPairs",
		"Version": "2016-11-15",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe key pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeKeyPairs failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeKeyPairsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var keyPairs []*KeyPair
	for _, kp := range descResp.KeyPairs.Items {
		keyPairs = append(keyPairs, &KeyPair{
			KeyName:        kp.KeyName,
			KeyFingerprint: kp.KeyFingerprint,
			KeyPairId:      kp.KeyPairId,
			Region:         c.provider.region,
		})
	}

	return keyPairs, nil
}

// ListKeyPairsInRegion lists all key pairs in a specific region
func (c *ComputeService) ListKeyPairsInRegion(ctx context.Context, region string) ([]*KeyPair, error) {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeKeyPairs",
		"Version": "2016-11-15",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe key pairs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeKeyPairs failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeKeyPairsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var keyPairs []*KeyPair
	for _, kp := range descResp.KeyPairs.Items {
		keyPairs = append(keyPairs, &KeyPair{
			KeyName:        kp.KeyName,
			KeyFingerprint: kp.KeyFingerprint,
			KeyPairId:      kp.KeyPairId,
			Region:         region,
		})
	}

	return keyPairs, nil
}

// DeleteKeyPair deletes a key pair
func (c *ComputeService) DeleteKeyPair(ctx context.Context, keyName string) error {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DeleteKeyPair",
		"Version": "2016-11-15",
		"KeyName": keyName,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to delete key pair: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return fmt.Errorf("DeleteKeyPair failed: %s", cleanError)
	}

	return nil
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

	// Build ProviderData with SSH-relevant information
	providerData := make(map[string]interface{})
	if ec2Instance.KeyName != "" {
		providerData["KeyName"] = ec2Instance.KeyName
	}
	if ec2Instance.ImageId != "" {
		providerData["ImageId"] = ec2Instance.ImageId
	}
	if ec2Instance.Platform != "" {
		providerData["Platform"] = ec2Instance.Platform
	}

	// Add security groups
	if len(ec2Instance.SecurityGroups.Items) > 0 {
		var sgIds []string
		var sgNames []string
		for _, sg := range ec2Instance.SecurityGroups.Items {
			sgIds = append(sgIds, sg.GroupId)
			sgNames = append(sgNames, sg.GroupName)
		}
		providerData["SecurityGroupIds"] = sgIds
		providerData["SecurityGroupNames"] = sgNames
	}

	return &provider.Instance{
		ID:           ec2Instance.InstanceId,
		Name:         name,
		Type:         provider.InstanceType(c.reverseMapInstanceType(ec2Instance.InstanceType)),
		State:        ec2Instance.State.Name,
		PrivateIP:    ec2Instance.PrivateIpAddress,
		PublicIP:     ec2Instance.PublicIpAddress,
		Tags:         tags,
		CreatedAt:    createdAt,
		ProviderData: providerData,
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

// InstanceStatusCheckResponse represents the EC2 DescribeInstanceStatus API response
type InstanceStatusCheckResponse struct {
	XMLName          xml.Name `xml:"DescribeInstanceStatusResponse"`
	InstanceStatuses struct {
		Items []struct {
			InstanceId    string `xml:"instanceId"`
			InstanceState struct {
				Name string `xml:"name"`
			} `xml:"instanceState"`
			SystemStatus struct {
				Status string `xml:"status"`
			} `xml:"systemStatus"`
			InstanceStatus struct {
				Status string `xml:"status"`
			} `xml:"instanceStatus"`
		} `xml:"item"`
	} `xml:"instanceStatusSet"`
}

// InstanceStatusResult contains the status check results for an instance
type InstanceStatusResult struct {
	InstanceId     string
	InstanceState  string
	SystemStatus   string
	InstanceStatus string
	IsReady        bool
}

// GetInstanceStatus retrieves the status checks for an instance
func (c *ComputeService) GetInstanceStatus(ctx context.Context, instanceId string) (*InstanceStatusResult, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":       "DescribeInstanceStatus",
		"Version":      "2016-11-15",
		"InstanceId.1": instanceId,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeInstanceStatus failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var statusResp InstanceStatusCheckResponse
	if err := xml.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If no status returned, instance may not be running yet
	if len(statusResp.InstanceStatuses.Items) == 0 {
		return &InstanceStatusResult{
			InstanceId:     instanceId,
			InstanceState:  "pending",
			SystemStatus:   "initializing",
			InstanceStatus: "initializing",
			IsReady:        false,
		}, nil
	}

	status := statusResp.InstanceStatuses.Items[0]
	result := &InstanceStatusResult{
		InstanceId:     status.InstanceId,
		InstanceState:  status.InstanceState.Name,
		SystemStatus:   status.SystemStatus.Status,
		InstanceStatus: status.InstanceStatus.Status,
		IsReady:        status.InstanceState.Name == "running" && status.SystemStatus.Status == "ok" && status.InstanceStatus.Status == "ok",
	}

	return result, nil
}

// GetInstanceStatusInRegion retrieves the status checks for an instance in a specific region
func (c *ComputeService) GetInstanceStatusInRegion(ctx context.Context, instanceId, region string) (*InstanceStatusResult, error) {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":       "DescribeInstanceStatus",
		"Version":      "2016-11-15",
		"InstanceId.1": instanceId,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		cleanError := parseEC2Error(body)
		return nil, fmt.Errorf("DescribeInstanceStatus failed: %s", cleanError)
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var statusResp InstanceStatusCheckResponse
	if err := xml.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If no status returned, instance may not be running yet
	if len(statusResp.InstanceStatuses.Items) == 0 {
		return &InstanceStatusResult{
			InstanceId:     instanceId,
			InstanceState:  "pending",
			SystemStatus:   "initializing",
			InstanceStatus: "initializing",
			IsReady:        false,
		}, nil
	}

	status := statusResp.InstanceStatuses.Items[0]
	result := &InstanceStatusResult{
		InstanceId:     status.InstanceId,
		InstanceState:  status.InstanceState.Name,
		SystemStatus:   status.SystemStatus.Status,
		InstanceStatus: status.InstanceStatus.Status,
		IsReady:        status.InstanceState.Name == "running" && status.SystemStatus.Status == "ok" && status.InstanceStatus.Status == "ok",
	}

	return result, nil
}

// WaitForInstanceReady waits for an instance to pass status checks with a timeout
func (c *ComputeService) WaitForInstanceReady(ctx context.Context, instanceId string, timeoutSeconds int) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for instance %s to be ready", instanceId)
		case <-ticker.C:
			status, err := c.GetInstanceStatus(ctx, instanceId)
			if err != nil {
				// Continue waiting on transient errors
				continue
			}
			if status.IsReady {
				return nil
			}
		}
	}
}

// WaitForInstanceReadyInRegion waits for an instance in a specific region to pass status checks
func (c *ComputeService) WaitForInstanceReadyInRegion(ctx context.Context, instanceId, region string, timeoutSeconds int) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for instance %s to be ready", instanceId)
		case <-ticker.C:
			status, err := c.GetInstanceStatusInRegion(ctx, instanceId, region)
			if err != nil {
				// Continue waiting on transient errors
				continue
			}
			if status.IsReady {
				return nil
			}
		}
	}
}
