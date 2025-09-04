# Genesys Provider Architecture

## Core Design: Provider-Agnostic Interface

### Provider Interface Definition

```go
// pkg/provider/interface.go
package provider

import (
    "context"
)

// Provider is the main interface all cloud providers must implement
type Provider interface {
    // Metadata
    Name() string
    Region() string
    Validate() error
    
    // Resource factories
    Compute() ComputeService
    Storage() StorageService
    Network() NetworkService
    Database() DatabaseService
    Serverless() ServerlessService
    
    // State management
    StateBackend() StateBackend
    
    // Authentication
    Authenticate(ctx context.Context) error
}

// ComputeService handles all compute resources (VMs, containers, etc)
type ComputeService interface {
    CreateInstance(ctx context.Context, config *InstanceConfig) (*Instance, error)
    GetInstance(ctx context.Context, id string) (*Instance, error)
    UpdateInstance(ctx context.Context, id string, config *InstanceConfig) error
    DeleteInstance(ctx context.Context, id string) error
    ListInstances(ctx context.Context, filters map[string]string) ([]*Instance, error)
    
    // Adoption/Discovery
    DiscoverInstances(ctx context.Context) ([]*Instance, error)
    AdoptInstance(ctx context.Context, id string) (*Instance, error)
}

// StorageService handles storage resources (object storage, block storage)
type StorageService interface {
    CreateBucket(ctx context.Context, config *BucketConfig) (*Bucket, error)
    GetBucket(ctx context.Context, name string) (*Bucket, error)
    DeleteBucket(ctx context.Context, name string) error
    ListBuckets(ctx context.Context) ([]*Bucket, error)
    
    // Block storage
    CreateVolume(ctx context.Context, config *VolumeConfig) (*Volume, error)
    AttachVolume(ctx context.Context, volumeID, instanceID string) error
    
    // Discovery
    DiscoverBuckets(ctx context.Context) ([]*Bucket, error)
    AdoptBucket(ctx context.Context, name string) (*Bucket, error)
}

// NetworkService handles networking resources
type NetworkService interface {
    CreateNetwork(ctx context.Context, config *NetworkConfig) (*Network, error)
    CreateSubnet(ctx context.Context, networkID string, config *SubnetConfig) (*Subnet, error)
    CreateSecurityGroup(ctx context.Context, config *SecurityGroupConfig) (*SecurityGroup, error)
    CreateLoadBalancer(ctx context.Context, config *LoadBalancerConfig) (*LoadBalancer, error)
    
    // Discovery
    DiscoverNetworks(ctx context.Context) ([]*Network, error)
    AdoptNetwork(ctx context.Context, id string) (*Network, error)
}

// DatabaseService handles database resources
type DatabaseService interface {
    CreateDatabase(ctx context.Context, config *DatabaseConfig) (*Database, error)
    GetDatabase(ctx context.Context, id string) (*Database, error)
    UpdateDatabase(ctx context.Context, id string, config *DatabaseConfig) error
    DeleteDatabase(ctx context.Context, id string) error
    
    // Discovery
    DiscoverDatabases(ctx context.Context) ([]*Database, error)
    AdoptDatabase(ctx context.Context, id string) (*Database, error)
}

// ServerlessService handles serverless resources
type ServerlessService interface {
    CreateFunction(ctx context.Context, config *FunctionConfig) (*Function, error)
    UpdateFunction(ctx context.Context, id string, config *FunctionConfig) error
    DeleteFunction(ctx context.Context, id string) error
    InvokeFunction(ctx context.Context, id string, payload []byte) ([]byte, error)
    
    // Layers
    CreateLayer(ctx context.Context, config *LayerConfig) (*Layer, error)
    
    // Discovery
    DiscoverFunctions(ctx context.Context) ([]*Function, error)
    AdoptFunction(ctx context.Context, id string) (*Function, error)
}
```

### Universal Resource Models

```go
// pkg/provider/resources.go
package provider

// Universal compute instance - works for any provider
type Instance struct {
    // Common fields across all providers
    ID           string
    Name         string
    Type         InstanceType  // Abstracted sizing
    State        string
    PrivateIP    string
    PublicIP     string
    Tags         map[string]string
    
    // Provider-specific metadata stored here
    ProviderData map[string]interface{}
}

type InstanceConfig struct {
    Name         string
    Type         InstanceType  // small|medium|large|xlarge
    Image        string        // Can be alias like "ubuntu-lts" 
    Network      string
    Subnet       string
    SecurityGroups []string
    UserData     string
    KeyPair      string
    Tags         map[string]string
}

// Instance types abstracted from provider-specific names
type InstanceType string

const (
    InstanceTypeSmall   InstanceType = "small"   // 1-2 vCPU, 1-2 GB RAM
    InstanceTypeMedium  InstanceType = "medium"  // 2-4 vCPU, 4-8 GB RAM
    InstanceTypeLarge   InstanceType = "large"   // 4-8 vCPU, 8-16 GB RAM
    InstanceTypeXLarge  InstanceType = "xlarge"  // 8+ vCPU, 16+ GB RAM
)

// Universal storage bucket
type Bucket struct {
    Name         string
    Region       string
    Versioning   bool
    Encryption   EncryptionConfig
    PublicAccess bool
    Tags         map[string]string
    ProviderData map[string]interface{}
}

type BucketConfig struct {
    Name         string
    Versioning   bool
    Encryption   bool
    PublicAccess bool
    Lifecycle    *LifecycleConfig
    Tags         map[string]string
}

// Universal network
type Network struct {
    ID           string
    Name         string
    CIDR         string
    Subnets      []Subnet
    Tags         map[string]string
    ProviderData map[string]interface{}
}

// Universal database
type Database struct {
    ID           string
    Name         string
    Engine       string  // postgres|mysql|mongodb
    Version      string
    Size         DatabaseSize
    Storage      int     // GB
    MultiAZ      bool
    Backups      BackupConfig
    Tags         map[string]string
    ProviderData map[string]interface{}
}

type DatabaseSize string

const (
    DatabaseSizeSmall  DatabaseSize = "small"   // 1-2 vCPU, 1-4 GB RAM
    DatabaseSizeMedium DatabaseSize = "medium"  // 2-4 vCPU, 4-16 GB RAM
    DatabaseSizeLarge  DatabaseSize = "large"   // 4-8 vCPU, 16-64 GB RAM
)

// Universal serverless function
type Function struct {
    ID           string
    Name         string
    Runtime      string  // python3.11|nodejs18|go1.21
    Handler      string
    Memory       int     // MB
    Timeout      int     // seconds
    Environment  map[string]string
    Triggers     []Trigger
    Layers       []string
    Tags         map[string]string
    ProviderData map[string]interface{}
}
```

## Provider Implementations

### AWS Provider Implementation

```go
// pkg/provider/aws/provider.go
package aws

import (
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/rds"
    "github.com/aws/aws-sdk-go-v2/service/lambda"
    "github.com/genesys/pkg/provider"
)

type AWSProvider struct {
    region      string
    ec2Client   *ec2.Client
    s3Client    *s3.Client
    rdsClient   *rds.Client
    lambdaClient *lambda.Client
}

func New(region string) provider.Provider {
    return &AWSProvider{
        region: region,
    }
}

func (p *AWSProvider) Name() string {
    return "aws"
}

func (p *AWSProvider) Compute() provider.ComputeService {
    return &AWSComputeService{
        client: p.ec2Client,
    }
}

// AWSComputeService implements ComputeService for AWS
type AWSComputeService struct {
    client *ec2.Client
}

func (s *AWSComputeService) CreateInstance(ctx context.Context, config *provider.InstanceConfig) (*provider.Instance, error) {
    // Translate generic config to AWS-specific EC2 parameters
    instanceType := s.translateInstanceType(config.Type)
    ami := s.resolveAMI(config.Image)
    
    input := &ec2.RunInstancesInput{
        ImageId:      &ami,
        InstanceType: instanceType,
        MinCount:     aws.Int32(1),
        MaxCount:     aws.Int32(1),
        TagSpecifications: s.buildTags(config.Tags),
    }
    
    result, err := s.client.RunInstances(ctx, input)
    if err != nil {
        return nil, err
    }
    
    // Convert AWS instance to universal Instance
    return s.toUniversalInstance(result.Instances[0]), nil
}

func (s *AWSComputeService) translateInstanceType(t provider.InstanceType) types.InstanceType {
    switch t {
    case provider.InstanceTypeSmall:
        return types.InstanceTypeT3Small
    case provider.InstanceTypeMedium:
        return types.InstanceTypeT3Medium
    case provider.InstanceTypeLarge:
        return types.InstanceTypeT3Large
    case provider.InstanceTypeXLarge:
        return types.InstanceTypeT3XLarge
    default:
        return types.InstanceTypeT3Small
    }
}

func (s *AWSComputeService) resolveAMI(image string) string {
    // Resolve image aliases to actual AMI IDs
    switch image {
    case "ubuntu-lts":
        return s.getLatestUbuntuAMI()
    case "amazon-linux":
        return s.getLatestAmazonLinuxAMI()
    default:
        return image // Assume it's already an AMI ID
    }
}
```

### GCP Provider Implementation

```go
// pkg/provider/gcp/provider.go
package gcp

import (
    compute "cloud.google.com/go/compute/apiv1"
    "github.com/genesys/pkg/provider"
)

type GCPProvider struct {
    project      string
    region       string
    computeClient *compute.InstancesClient
}

func (p *GCPProvider) Compute() provider.ComputeService {
    return &GCPComputeService{
        client:  p.computeClient,
        project: p.project,
    }
}

type GCPComputeService struct {
    client  *compute.InstancesClient
    project string
    zone    string
}

func (s *GCPComputeService) CreateInstance(ctx context.Context, config *provider.InstanceConfig) (*provider.Instance, error) {
    // Translate generic config to GCP-specific parameters
    machineType := s.translateMachineType(config.Type)
    image := s.resolveImage(config.Image)
    
    instance := &computepb.Instance{
        Name:        &config.Name,
        MachineType: &machineType,
        Disks:       s.buildBootDisk(image),
        NetworkInterfaces: s.buildNetworkInterfaces(config),
        Labels:      config.Tags,
    }
    
    op, err := s.client.Insert(ctx, &computepb.InsertInstanceRequest{
        Project:  s.project,
        Zone:     s.zone,
        Instance: instance,
    })
    
    if err != nil {
        return nil, err
    }
    
    // Wait for operation and convert to universal Instance
    return s.toUniversalInstance(op), nil
}

func (s *GCPComputeService) translateMachineType(t provider.InstanceType) string {
    switch t {
    case provider.InstanceTypeSmall:
        return "n1-standard-1"
    case provider.InstanceTypeMedium:
        return "n1-standard-2"
    case provider.InstanceTypeLarge:
        return "n1-standard-4"
    case provider.InstanceTypeXLarge:
        return "n1-standard-8"
    default:
        return "n1-standard-1"
    }
}
```

### Azure Provider Implementation

```go
// pkg/provider/azure/provider.go
package azure

import (
    "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
    "github.com/genesys/pkg/provider"
)

type AzureProvider struct {
    subscription string
    resourceGroup string
    computeClient *armcompute.VirtualMachinesClient
}

func (p *AzureProvider) Compute() provider.ComputeService {
    return &AzureComputeService{
        client:        p.computeClient,
        subscription:  p.subscription,
        resourceGroup: p.resourceGroup,
    }
}

type AzureComputeService struct {
    client        *armcompute.VirtualMachinesClient
    subscription  string
    resourceGroup string
}

func (s *AzureComputeService) CreateInstance(ctx context.Context, config *provider.InstanceConfig) (*provider.Instance, error) {
    // Translate generic config to Azure-specific parameters
    vmSize := s.translateVMSize(config.Type)
    image := s.resolveImageReference(config.Image)
    
    parameters := armcompute.VirtualMachine{
        Location: to.Ptr(s.location),
        Properties: &armcompute.VirtualMachineProperties{
            HardwareProfile: &armcompute.HardwareProfile{
                VMSize: &vmSize,
            },
            StorageProfile: &armcompute.StorageProfile{
                ImageReference: image,
            },
            OSProfile: s.buildOSProfile(config),
            NetworkProfile: s.buildNetworkProfile(config),
        },
        Tags: config.Tags,
    }
    
    poller, err := s.client.BeginCreateOrUpdate(ctx, s.resourceGroup, config.Name, parameters, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := poller.PollUntilDone(ctx, nil)
    if err != nil {
        return nil, err
    }
    
    return s.toUniversalInstance(resp.VirtualMachine), nil
}

func (s *AzureComputeService) translateVMSize(t provider.InstanceType) armcompute.VirtualMachineSizeTypes {
    switch t {
    case provider.InstanceTypeSmall:
        return armcompute.VirtualMachineSizeTypesStandardB1s
    case provider.InstanceTypeMedium:
        return armcompute.VirtualMachineSizeTypesStandardB2s
    case provider.InstanceTypeLarge:
        return armcompute.VirtualMachineSizeTypesStandardD2sV3
    case provider.InstanceTypeXLarge:
        return armcompute.VirtualMachineSizeTypesStandardD4sV3
    default:
        return armcompute.VirtualMachineSizeTypesStandardB1s
    }
}
```

## Configuration Files (YAML/TOML)

### YAML Configuration Example

```yaml
# genesys.yaml
provider: aws  # Can be aws|gcp|azure|alibaba|tencent
region: us-east-1

# Outcome-based configuration
outcomes:
  static-site:
    domain: example.com
    enable_cdn: true
    enable_https: true
    
  api:
    name: my-api
    runtime: python3.11
    memory: 512
    timeout: 30

# Resource-based configuration (advanced users)
resources:
  compute:
    - name: web-server
      type: medium  # Translates to t3.medium on AWS, n1-standard-2 on GCP, etc
      image: ubuntu-lts
      count: 3
      network: main-vpc
      security_groups:
        - web-sg
      tags:
        Environment: production
        Team: platform

  storage:
    - name: app-data
      type: bucket
      versioning: true
      encryption: true
      lifecycle:
        delete_after_days: 90
      tags:
        Purpose: application-data

  network:
    - name: main-vpc
      cidr: 10.0.0.0/16
      subnets:
        - name: public
          cidr: 10.0.1.0/24
          public: true
        - name: private
          cidr: 10.0.2.0/24
          public: false

  database:
    - name: app-db
      engine: postgres
      version: "14"
      size: medium  # Translates appropriately per provider
      storage: 100  # GB
      multi_az: true
      backup:
        retention_days: 7
        window: "03:00-04:00"

  serverless:
    - name: api-handler
      runtime: python3.11
      handler: main.handler
      memory: 512
      timeout: 30
      environment:
        DB_HOST: ${database.app-db.endpoint}
      triggers:
        - type: http
          path: /api/*
          methods: [GET, POST]

# State configuration (optional, auto-configured if not specified)
state:
  backend: s3  # or gcs, azureblob, local
  bucket: genesys-state-${account_id}
  lock_table: genesys-locks
  encrypt: true

# Policy guardrails
policies:
  - no_public_buckets: true
  - require_encryption: true
  - require_tags: [Environment, Team]
  - max_cost_per_month: 1000
```

### TOML Configuration Example

```toml
# genesys.toml
provider = "gcp"
project = "my-project"
region = "us-central1"

[outcomes.static-site]
domain = "example.com"
enable_cdn = true
enable_https = true

[outcomes.api]
name = "my-api"
runtime = "nodejs18"
memory = 256
timeout = 60

[[resources.compute]]
name = "web-server"
type = "medium"
image = "ubuntu-lts"
count = 3
network = "main-vpc"
security_groups = ["web-sg"]

[resources.compute.tags]
Environment = "production"
Team = "platform"

[[resources.storage]]
name = "app-data"
type = "bucket"
versioning = true
encryption = true

[resources.storage.lifecycle]
delete_after_days = 90

[[resources.network]]
name = "main-vpc"
cidr = "10.0.0.0/16"

[[resources.network.subnets]]
name = "public"
cidr = "10.0.1.0/24"
public = true

[[resources.network.subnets]]
name = "private"
cidr = "10.0.2.0/24"
public = false

[[resources.database]]
name = "app-db"
engine = "postgres"
version = "14"
size = "medium"
storage = 100
multi_az = true

[resources.database.backup]
retention_days = 7
window = "03:00-04:00"

[[resources.serverless]]
name = "api-handler"
runtime = "python3.11"
handler = "main.handler"
memory = 512
timeout = 30

[resources.serverless.environment]
DB_HOST = "${database.app-db.endpoint}"

[[resources.serverless.triggers]]
type = "http"
path = "/api/*"
methods = ["GET", "POST"]

[state]
backend = "gcs"
bucket = "genesys-state-${project_id}"
encrypt = true

[policies]
no_public_buckets = true
require_encryption = true
require_tags = ["Environment", "Team"]
max_cost_per_month = 1000
```

## Provider Registry and Loading

```go
// pkg/provider/registry.go
package provider

import (
    "fmt"
    "sync"
)

type ProviderFactory func(config map[string]string) (Provider, error)

type Registry struct {
    mu        sync.RWMutex
    providers map[string]ProviderFactory
}

var globalRegistry = &Registry{
    providers: make(map[string]ProviderFactory),
}

// Register a new provider
func Register(name string, factory ProviderFactory) {
    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()
    globalRegistry.providers[name] = factory
}

// Get a provider by name
func Get(name string, config map[string]string) (Provider, error) {
    globalRegistry.mu.RLock()
    factory, exists := globalRegistry.providers[name]
    globalRegistry.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("provider %s not registered", name)
    }
    
    return factory(config)
}

// Initialize all providers
func init() {
    // Register built-in providers
    Register("aws", aws.NewFactory)
    Register("gcp", gcp.NewFactory)
    Register("azure", azure.NewFactory)
    Register("alibaba", alibaba.NewFactory)
    Register("tencent", tencent.NewFactory)
}
```

## Configuration Parser

```go
// pkg/config/parser.go
package config

import (
    "github.com/BurntSushi/toml"
    "gopkg.in/yaml.v3"
)

type Config struct {
    Provider  string                 `yaml:"provider" toml:"provider"`
    Region    string                 `yaml:"region" toml:"region"`
    Outcomes  map[string]Outcome     `yaml:"outcomes" toml:"outcomes"`
    Resources Resources              `yaml:"resources" toml:"resources"`
    State     StateConfig            `yaml:"state" toml:"state"`
    Policies  Policies               `yaml:"policies" toml:"policies"`
}

type Resources struct {
    Compute    []ComputeResource    `yaml:"compute" toml:"compute"`
    Storage    []StorageResource    `yaml:"storage" toml:"storage"`
    Network    []NetworkResource    `yaml:"network" toml:"network"`
    Database   []DatabaseResource   `yaml:"database" toml:"database"`
    Serverless []ServerlessResource `yaml:"serverless" toml:"serverless"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var config Config
    
    // Detect format by extension
    switch filepath.Ext(path) {
    case ".yaml", ".yml":
        err = yaml.Unmarshal(data, &config)
    case ".toml":
        err = toml.Unmarshal(data, &config)
    default:
        // Try to auto-detect format
        if err = yaml.Unmarshal(data, &config); err != nil {
            err = toml.Unmarshal(data, &config)
        }
    }
    
    if err != nil {
        return nil, err
    }
    
    // Validate and apply defaults
    config.applyDefaults()
    config.validate()
    
    return &config, nil
}
```

## Usage Example

```go
// cmd/genesys/main.go
package main

import (
    "github.com/genesys/pkg/config"
    "github.com/genesys/pkg/provider"
    "github.com/genesys/pkg/planner"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig("genesys.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get the appropriate provider
    p, err := provider.Get(cfg.Provider, map[string]string{
        "region": cfg.Region,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Create planner with provider
    plnr := planner.New(p)
    
    // Generate plan from configuration
    plan, err := plnr.Plan(cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Display human-readable plan
    fmt.Println(plan.ToHumanReadable())
    
    // Apply if requested
    if applyFlag {
        executor := executor.New(p)
        result, err := executor.Execute(plan)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Successfully deployed: %s\n", result.Summary())
    }
}
```

This architecture ensures:
1. **Provider agnostic**: Same configuration works across all providers
2. **Clean abstraction**: Provider interface hides implementation details
3. **Easy extension**: New providers just implement the interface
4. **Type safety**: Go interfaces ensure all providers implement required methods
5. **Configuration flexibility**: Support for both YAML and TOML
6. **Resource normalization**: Universal resource types mapped to provider-specific implementations