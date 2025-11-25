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
		"Action":  "DescribeVpcs",
		"Version": "2016-11-15",
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

// Security Group Management

// SecurityGroupDetail contains detailed security group information
type SecurityGroupDetail struct {
	GroupId      string
	GroupName    string
	Description  string
	VpcId        string
	IngressRules []SecurityGroupRule
	EgressRules  []SecurityGroupRule
}

// SecurityGroupRule represents an inbound or outbound rule
type SecurityGroupRule struct {
	Protocol   string
	FromPort   int
	ToPort     int
	CidrBlocks []string
}

// DescribeSecurityGroupsResponse represents the EC2 DescribeSecurityGroups API response
type DescribeSecurityGroupsResponse struct {
	XMLName        xml.Name `xml:"DescribeSecurityGroupsResponse"`
	SecurityGroups struct {
		Items []struct {
			GroupId             string          `xml:"groupId"`
			GroupName           string          `xml:"groupName"`
			Description         string          `xml:"groupDescription"`
			VpcId               string          `xml:"vpcId"`
			IpPermissions       IpPermissionSet `xml:"ipPermissions"`
			IpPermissionsEgress IpPermissionSet `xml:"ipPermissionsEgress"`
		} `xml:"item"`
	} `xml:"securityGroupInfo"`
}

// IpPermissionSet represents a set of IP permissions
type IpPermissionSet struct {
	Items []struct {
		IpProtocol string `xml:"ipProtocol"`
		FromPort   int    `xml:"fromPort"`
		ToPort     int    `xml:"toPort"`
		IpRanges   struct {
			Items []struct {
				CidrIp string `xml:"cidrIp"`
			} `xml:"item"`
		} `xml:"ipRanges"`
	} `xml:"item"`
}

// DescribeSecurityGroup retrieves details about a security group
func (n *NetworkService) DescribeSecurityGroup(ctx context.Context, groupId string) (*SecurityGroupDetail, error) {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":    "DescribeSecurityGroups",
		"Version":   "2016-11-15",
		"GroupId.1": groupId,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe security groups: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeSecurityGroups failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeSecurityGroupsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(descResp.SecurityGroups.Items) == 0 {
		return nil, fmt.Errorf("security group %s not found", groupId)
	}

	sg := descResp.SecurityGroups.Items[0]
	detail := &SecurityGroupDetail{
		GroupId:     sg.GroupId,
		GroupName:   sg.GroupName,
		Description: sg.Description,
		VpcId:       sg.VpcId,
	}

	// Parse ingress rules
	for _, perm := range sg.IpPermissions.Items {
		rule := SecurityGroupRule{
			Protocol: perm.IpProtocol,
			FromPort: perm.FromPort,
			ToPort:   perm.ToPort,
		}
		for _, cidr := range perm.IpRanges.Items {
			rule.CidrBlocks = append(rule.CidrBlocks, cidr.CidrIp)
		}
		detail.IngressRules = append(detail.IngressRules, rule)
	}

	// Parse egress rules
	for _, perm := range sg.IpPermissionsEgress.Items {
		rule := SecurityGroupRule{
			Protocol: perm.IpProtocol,
			FromPort: perm.FromPort,
			ToPort:   perm.ToPort,
		}
		for _, cidr := range perm.IpRanges.Items {
			rule.CidrBlocks = append(rule.CidrBlocks, cidr.CidrIp)
		}
		detail.EgressRules = append(detail.EgressRules, rule)
	}

	return detail, nil
}

// DescribeSecurityGroupInRegion retrieves details about a security group in a specific region
func (n *NetworkService) DescribeSecurityGroupInRegion(ctx context.Context, groupId, region string) (*SecurityGroupDetail, error) {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":    "DescribeSecurityGroups",
		"Version":   "2016-11-15",
		"GroupId.1": groupId,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to describe security groups: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return nil, fmt.Errorf("DescribeSecurityGroups failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := ReadResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var descResp DescribeSecurityGroupsResponse
	if err := xml.Unmarshal(body, &descResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(descResp.SecurityGroups.Items) == 0 {
		return nil, fmt.Errorf("security group %s not found", groupId)
	}

	sg := descResp.SecurityGroups.Items[0]
	detail := &SecurityGroupDetail{
		GroupId:     sg.GroupId,
		GroupName:   sg.GroupName,
		Description: sg.Description,
		VpcId:       sg.VpcId,
	}

	// Parse ingress rules
	for _, perm := range sg.IpPermissions.Items {
		rule := SecurityGroupRule{
			Protocol: perm.IpProtocol,
			FromPort: perm.FromPort,
			ToPort:   perm.ToPort,
		}
		for _, cidr := range perm.IpRanges.Items {
			rule.CidrBlocks = append(rule.CidrBlocks, cidr.CidrIp)
		}
		detail.IngressRules = append(detail.IngressRules, rule)
	}

	// Parse egress rules
	for _, perm := range sg.IpPermissionsEgress.Items {
		rule := SecurityGroupRule{
			Protocol: perm.IpProtocol,
			FromPort: perm.FromPort,
			ToPort:   perm.ToPort,
		}
		for _, cidr := range perm.IpRanges.Items {
			rule.CidrBlocks = append(rule.CidrBlocks, cidr.CidrIp)
		}
		detail.EgressRules = append(detail.EgressRules, rule)
	}

	return detail, nil
}

// AuthorizeSecurityGroupIngress adds an inbound rule to a security group
func (n *NetworkService) AuthorizeSecurityGroupIngress(ctx context.Context, groupId, protocol string, fromPort, toPort int, cidrIp string) error {
	client, err := n.provider.CreateClient("ec2")
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":                            "AuthorizeSecurityGroupIngress",
		"Version":                           "2016-11-15",
		"GroupId":                           groupId,
		"IpPermissions.1.IpProtocol":        protocol,
		"IpPermissions.1.FromPort":          fmt.Sprintf("%d", fromPort),
		"IpPermissions.1.ToPort":            fmt.Sprintf("%d", toPort),
		"IpPermissions.1.IpRanges.1.CidrIp": cidrIp,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to authorize security group ingress: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return fmt.Errorf("AuthorizeSecurityGroupIngress failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AuthorizeSecurityGroupIngressInRegion adds an inbound rule to a security group in a specific region
func (n *NetworkService) AuthorizeSecurityGroupIngressInRegion(ctx context.Context, groupId, region, protocol string, fromPort, toPort int, cidrIp string) error {
	client, err := NewAWSClient(region, "ec2")
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	params := map[string]string{
		"Action":                            "AuthorizeSecurityGroupIngress",
		"Version":                           "2016-11-15",
		"GroupId":                           groupId,
		"IpPermissions.1.IpProtocol":        protocol,
		"IpPermissions.1.FromPort":          fmt.Sprintf("%d", fromPort),
		"IpPermissions.1.ToPort":            fmt.Sprintf("%d", toPort),
		"IpPermissions.1.IpRanges.1.CidrIp": cidrIp,
	}

	resp, err := client.Request("POST", "/", params, nil)
	if err != nil {
		return fmt.Errorf("failed to authorize security group ingress: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ReadResponse(resp)
		return fmt.Errorf("AuthorizeSecurityGroupIngress failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddSSHRule adds an SSH inbound rule (port 22) to a security group
func (n *NetworkService) AddSSHRule(ctx context.Context, groupId, cidrIp string) error {
	return n.AuthorizeSecurityGroupIngress(ctx, groupId, "tcp", 22, 22, cidrIp)
}

// AddSSHRuleInRegion adds an SSH inbound rule (port 22) to a security group in a specific region
func (n *NetworkService) AddSSHRuleInRegion(ctx context.Context, groupId, region, cidrIp string) error {
	return n.AuthorizeSecurityGroupIngressInRegion(ctx, groupId, region, "tcp", 22, 22, cidrIp)
}

// HasSSHRule checks if a security group has an SSH rule (port 22 TCP inbound) with actual CIDR blocks
func (sg *SecurityGroupDetail) HasSSHRule() bool {
	for _, rule := range sg.IngressRules {
		// Must have at least one CIDR block to actually allow traffic
		if len(rule.CidrBlocks) == 0 {
			continue
		}
		// Check for TCP port 22
		if rule.Protocol == "tcp" && rule.FromPort <= 22 && rule.ToPort >= 22 {
			return true
		}
		// Check for all protocols (-1)
		if rule.Protocol == "-1" {
			return true
		}
	}
	return false
}

// GetSSHCidrs returns the CIDR blocks that allow SSH access
func (sg *SecurityGroupDetail) GetSSHCidrs() []string {
	var cidrs []string
	for _, rule := range sg.IngressRules {
		if (rule.Protocol == "tcp" && rule.FromPort <= 22 && rule.ToPort >= 22) || rule.Protocol == "-1" {
			cidrs = append(cidrs, rule.CidrBlocks...)
		}
	}
	return cidrs
}
