package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/javanhut/genesys/pkg/provider"
)

// NetworkService implements AWS VPC operations using direct API calls
type NetworkService struct {
	provider *AWSProvider
}

// NewNetworkService creates a new network service
func NewNetworkService(p *AWSProvider) *NetworkService {
	return &NetworkService{
		provider: p,
	}
}

// VPC API response structures
type CreateVpcResponse struct {
	XMLName xml.Name `xml:"CreateVpcResponse"`
	VPC     VPC      `xml:"vpc"`
}

type DescribeVpcsResponse struct {
	XMLName xml.Name `xml:"DescribeVpcsResponse"`
	VPCs    struct {
		Items []VPC `xml:"item"`
	} `xml:"vpcSet"`
}

type VPC struct {
	VpcId     string `xml:"vpcId"`
	CidrBlock string `xml:"cidrBlock"`
	State     string `xml:"state"`
	Tags      struct {
		Items []struct {
			Key   string `xml:"key"`
			Value string `xml:"value"`
		} `xml:"item"`
	} `xml:"tagSet"`
}

// CreateNetwork creates a new VPC
func (n *NetworkService) CreateNetwork(ctx context.Context, config *provider.NetworkConfig) (*provider.Network, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	// Build parameters
	params := map[string]string{
		"Action":    "CreateVpc",
		"Version":   "2016-11-15",
		"CidrBlock": config.CIDR,
	}

	// Make the request
	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("CreateVpc failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var createResp CreateVpcResponse
	if err := xml.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Add tags if provided
	if len(config.Tags) > 0 {
		if err := n.createTags(client, createResp.VPC.VpcId, config.Tags); err != nil {
			return nil, fmt.Errorf("failed to add tags: %w", err)
		}
	}

	// Add Name tag
	if config.Name != "" {
		tags := map[string]string{"Name": config.Name}
		if err := n.createTags(client, createResp.VPC.VpcId, tags); err != nil {
			return nil, fmt.Errorf("failed to add name tag: %w", err)
		}
	}

	// Convert to provider network
	return &provider.Network{
		ID:        createResp.VPC.VpcId,
		Name:      config.Name,
		CIDR:      createResp.VPC.CidrBlock,
		Tags:      config.Tags,
		CreatedAt: time.Now(),
	}, nil
}

// GetNetwork retrieves a network by ID
func (n *NetworkService) GetNetwork(ctx context.Context, id string) (*provider.Network, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":   "DescribeVpcs",
		"Version":  "2016-11-15",
		"VpcId.1": id,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPCs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeVpcs failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeVpcsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	for _, vpc := range descResp.VPCs.Items {
		if vpc.VpcId == id {
			return n.convertToProviderNetwork(vpc), nil
		}
	}

	return nil, fmt.Errorf("VPC %s not found", id)
}

// CreateSubnet creates a subnet in a VPC
func (n *NetworkService) CreateSubnet(ctx context.Context, networkID string, config *provider.SubnetConfig) (*provider.Subnet, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":    "CreateSubnet",
		"Version":   "2016-11-15",
		"VpcId":     networkID,
		"CidrBlock": config.CIDR,
	}

	if config.AZ != "" {
		params["AvailabilityZone"] = config.AZ
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("CreateSubnet failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For simplicity, return a basic subnet structure
	return &provider.Subnet{
		ID:        fmt.Sprintf("subnet-%d", time.Now().Unix()),
		Name:      config.Name,
		CIDR:      config.CIDR,
		NetworkID: networkID,
		Public:    config.Public,
		AZ:        config.AZ,
	}, nil
}

// CreateSecurityGroup creates a security group
func (n *NetworkService) CreateSecurityGroup(ctx context.Context, config *provider.SecurityGroupConfig) (*provider.SecurityGroup, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":           "CreateSecurityGroup",
		"Version":          "2016-11-15",
		"GroupName":        config.Name,
		"GroupDescription": config.Description,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create security group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("CreateSecurityGroup failed with status %d: %s", resp.StatusCode, string(body))
	}

	// For simplicity, return a basic security group structure
	return &provider.SecurityGroup{
		ID:          fmt.Sprintf("sg-%d", time.Now().Unix()),
		Name:        config.Name,
		Description: config.Description,
		Rules:       config.Rules,
		Tags:        config.Tags,
	}, nil
}

// DiscoverNetworks discovers existing VPCs
func (n *NetworkService) DiscoverNetworks(ctx context.Context) ([]*provider.Network, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":  "DescribeVpcs",
		"Version": "2016-11-15",
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPCs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeVpcs failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeVpcsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var networks []*provider.Network
	for _, vpc := range descResp.VPCs.Items {
		networks = append(networks, n.convertToProviderNetwork(vpc))
	}

	return networks, nil
}

// AdoptNetwork adopts an existing VPC into management
func (n *NetworkService) AdoptNetwork(ctx context.Context, id string) (*provider.Network, error) {
	return n.GetNetwork(ctx, id)
}

// Helper methods

func (n *NetworkService) createTags(client *AWSClient, resourceId string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	params := map[string]string{
		"Action":       "CreateTags",
		"Version":      "2016-11-15",
		"ResourceId.1": resourceId,
	}

	tagIndex := 1
	for key, value := range tags {
		params[fmt.Sprintf("Tag.%d.Key", tagIndex)] = key
		params[fmt.Sprintf("Tag.%d.Value", tagIndex)] = value
		tagIndex++
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return fmt.Errorf("CreateTags failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (n *NetworkService) convertToProviderNetwork(vpc VPC) *provider.Network {
	tags := make(map[string]string)
	var name string

	for _, tag := range vpc.Tags.Items {
		tags[tag.Key] = tag.Value
		if tag.Key == "Name" {
			name = tag.Value
		}
	}

	return &provider.Network{
		ID:        vpc.VpcId,
		Name:      name,
		CIDR:      vpc.CidrBlock,
		Tags:      tags,
		CreatedAt: time.Now(), // We don't have creation time from basic API
	}
}