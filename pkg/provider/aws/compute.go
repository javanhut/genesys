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
	provider *AWSProvider
}

// NewComputeService creates a new compute service
func NewComputeService(p *AWSProvider) *ComputeService {
	return &ComputeService{
		provider: p,
	}
}

// EC2 API response structures
type RunInstancesResponse struct {
	XMLName   xml.Name `xml:"RunInstancesResponse"`
	Instances struct {
		Items []EC2Instance `xml:"item"`
	} `xml:"instances"`
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

// CreateInstance creates a new EC2 instance
func (c *ComputeService) CreateInstance(ctx context.Context, config *provider.InstanceConfig) (*provider.Instance, error) {
	client, err := c.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	// Map instance types
	instanceType := c.mapInstanceType(string(config.Type))
	
	// Get AMI ID for the image
	amiId, err := c.getAMIForImage(client, config.Image)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI for image %s: %w", config.Image, err)
	}

	// Build parameters
	params := map[string]string{
		"Action":       "RunInstances",
		"Version":      "2016-11-15",
		"ImageId":      amiId,
		"MinCount":     "1", // Default to 1 instance
		"MaxCount":     "1", // Default to 1 instance  
		"InstanceType": instanceType,
	}

	// Add tags
	tagIndex := 1
	for key, value := range config.Tags {
		params[fmt.Sprintf("TagSpecification.1.ResourceType")] = "instance"
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("TagSpecification.1.Tag.%d.Value", tagIndex)] = value
		tagIndex++
	}

	// Add Name tag
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
		return nil, fmt.Errorf("RunInstances failed with status %d: %s", resp.StatusCode, string(body))
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
		return nil, fmt.Errorf("no instances created")
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
		return nil, fmt.Errorf("DescribeInstances failed with status %d: %s", resp.StatusCode, string(body))
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
		return fmt.Errorf("CreateTags failed with status %d: %s", resp.StatusCode, string(body))
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
		return fmt.Errorf("TerminateInstances failed with status %d: %s", resp.StatusCode, string(body))
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
		return nil, fmt.Errorf("DescribeInstances failed with status %d: %s", resp.StatusCode, string(body))
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
	default:
		return "t3.medium"
	}
}

func (c *ComputeService) getAMIForImage(client *AWSClient, image string) (string, error) {
	// Default AMIs for common images in us-east-1
	switch image {
	case "ubuntu-lts", "ubuntu":
		return "ami-0c02fb55956c7d316", nil // Ubuntu 20.04 LTS
	case "amazon-linux", "amzn2":
		return "ami-0abcdef1234567890", nil // Amazon Linux 2
	case "centos":
		return "ami-0d5eff06f840b45e9", nil // CentOS 7
	default:
		// Return Ubuntu as default
		return "ami-0c02fb55956c7d316", nil
	}
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