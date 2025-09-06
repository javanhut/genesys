package provider

import "time"

// InstanceType represents abstracted compute sizing
type InstanceType string

const (
	InstanceTypeSmall  InstanceType = "small"  // 1-2 vCPU, 1-2 GB RAM
	InstanceTypeMedium InstanceType = "medium" // 2-4 vCPU, 4-8 GB RAM
	InstanceTypeLarge  InstanceType = "large"  // 4-8 vCPU, 8-16 GB RAM
	InstanceTypeXLarge InstanceType = "xlarge" // 8+ vCPU, 16+ GB RAM
)

// DatabaseSize represents abstracted database sizing
type DatabaseSize string

const (
	DatabaseSizeSmall  DatabaseSize = "small"  // 1-2 vCPU, 1-4 GB RAM
	DatabaseSizeMedium DatabaseSize = "medium" // 2-4 vCPU, 4-16 GB RAM
	DatabaseSizeLarge  DatabaseSize = "large"  // 4-8 vCPU, 16-64 GB RAM
)

// Instance represents a compute instance (VM)
type Instance struct {
	ID           string
	Name         string
	Type         InstanceType
	State        string
	PrivateIP    string
	PublicIP     string
	Tags         map[string]string
	CreatedAt    time.Time
	ProviderData map[string]interface{} // Provider-specific metadata
}

// InstanceConfig for creating/updating instances
type InstanceConfig struct {
	Name           string
	Type           InstanceType
	Image          string // Can be alias like "ubuntu-lts"
	Network        string
	Subnet         string
	SecurityGroups []string
	UserData       string
	KeyPair        string
	Tags           map[string]string
}

// Bucket represents object storage
type Bucket struct {
	Name         string
	Region       string
	Versioning   bool
	Encryption   bool
	PublicAccess bool
	Tags         map[string]string
	CreatedAt    time.Time
	ProviderData map[string]interface{}
}

// BucketConfig for creating/updating buckets
type BucketConfig struct {
	Name         string
	Versioning   bool
	Encryption   bool
	PublicAccess bool
	Lifecycle    *LifecycleConfig
	Tags         map[string]string
}

// LifecycleConfig for bucket lifecycle rules
type LifecycleConfig struct {
	DeleteAfterDays  int
	ArchiveAfterDays int
}

// Network represents a virtual network
type Network struct {
	ID           string
	Name         string
	CIDR         string
	Subnets      []Subnet
	Tags         map[string]string
	CreatedAt    time.Time
	ProviderData map[string]interface{}
}

// NetworkConfig for creating networks
type NetworkConfig struct {
	Name string
	CIDR string
	Tags map[string]string
}

// Subnet represents a network subnet
type Subnet struct {
	ID           string
	Name         string
	CIDR         string
	NetworkID    string
	Public       bool
	AZ           string // Availability Zone
	ProviderData map[string]interface{}
}

// SubnetConfig for creating subnets
type SubnetConfig struct {
	Name   string
	CIDR   string
	Public bool
	AZ     string
}

// SecurityGroup represents firewall rules
type SecurityGroup struct {
	ID           string
	Name         string
	Description  string
	Rules        []SecurityRule
	Tags         map[string]string
	ProviderData map[string]interface{}
}

// SecurityGroupConfig for creating security groups
type SecurityGroupConfig struct {
	Name        string
	Description string
	Rules       []SecurityRule
	Tags        map[string]string
}

// SecurityRule represents a firewall rule
type SecurityRule struct {
	Direction string // ingress or egress
	Protocol  string // tcp, udp, icmp, all
	FromPort  int
	ToPort    int
	Source    string // CIDR or security group
}

// Database represents a managed database
type Database struct {
	ID           string
	Name         string
	Engine       string // postgres, mysql, mongodb
	Version      string
	Size         DatabaseSize
	Storage      int // GB
	MultiAZ      bool
	Endpoint     string
	Port         int
	Tags         map[string]string
	CreatedAt    time.Time
	ProviderData map[string]interface{}
}

// DatabaseConfig for creating/updating databases
type DatabaseConfig struct {
	Name           string
	Engine         string
	Version        string
	Size           DatabaseSize
	Storage        int
	MultiAZ        bool
	BackupConfig   *BackupConfig
	Tags           map[string]string
	MasterUser     string
	MasterPassword string
}

// BackupConfig for database backups
type BackupConfig struct {
	RetentionDays int
	Window        string // e.g., "03:00-04:00"
	FinalSnapshot bool
}

// Function represents a serverless function
type Function struct {
	ID           string
	Name         string
	Runtime      string // python3.11, nodejs18, go1.21
	Handler      string
	Memory       int // MB
	Timeout      int // seconds
	Environment  map[string]string
	URL          string // Function URL if exposed
	Tags         map[string]string
	CreatedAt    time.Time
	ProviderData map[string]interface{}
}

// FunctionConfig for creating/updating functions
type FunctionConfig struct {
	Name        string
	Runtime     string
	Handler     string
	Memory      int
	Timeout     int
	Environment map[string]string
	Code        FunctionCode
	Triggers    []TriggerConfig
	Tags        map[string]string
	Role        string // IAM role ARN
}

// FunctionCode represents function deployment package
type FunctionCode struct {
	ZipFile   []byte // Direct zip upload
	S3Bucket  string // S3 location
	S3Key     string
	ImageURI  string   // Container image
	LocalPath string   // Local path to ZIP file
	Layers    []string // Layer ARNs to attach
}

// TriggerConfig for function triggers
type TriggerConfig struct {
	Type   string // http, schedule, queue, storage
	Config map[string]interface{}
}

// State represents infrastructure state
type State struct {
	Version   int
	Resources map[string]interface{}
	Outputs   map[string]interface{}
	UpdatedAt time.Time
}

// LambdaLayer represents a Lambda layer
type LambdaLayer struct {
	ID                 string
	Name               string
	Description        string
	Version            int
	CompatibleRuntimes []string
	LayerArn           string
	LayerVersionArn    string
	CreatedAt          time.Time
	ProviderData       map[string]interface{}
}

// LambdaLayerConfig for creating/updating layers
type LambdaLayerConfig struct {
	Name               string
	Description        string
	CompatibleRuntimes []string
	Content            LayerContent
}

// LayerContent represents layer deployment package
type LayerContent struct {
	ZipFile   []byte // Direct zip upload
	S3Bucket  string // S3 location
	S3Key     string
	LocalPath string // Local path to ZIP file
}
