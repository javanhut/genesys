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
	PublicIP       bool // Whether to assign a public IP address
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

// Monitoring and Metrics

// MetricsData holds general metrics data for any resource
type MetricsData struct {
	ResourceID   string
	ResourceType string
	Period       string
	StartTime    time.Time
	EndTime      time.Time
	Datapoints   []MetricDatapoint
}

// MetricDatapoint represents a single metric measurement
type MetricDatapoint struct {
	Timestamp time.Time
	Value     float64
	Unit      string
	Statistic string
}

// EC2Metrics contains EC2-specific metrics
type EC2Metrics struct {
	InstanceID                string
	CPUUtilization            []MetricDatapoint
	NetworkIn                 []MetricDatapoint
	NetworkOut                []MetricDatapoint
	DiskReadOps               []MetricDatapoint
	DiskWriteOps              []MetricDatapoint
	DiskReadBytes             []MetricDatapoint
	DiskWriteBytes            []MetricDatapoint
	StatusCheckFailed         []MetricDatapoint
	StatusCheckFailedInstance []MetricDatapoint
	StatusCheckFailedSystem   []MetricDatapoint
}

// S3Metrics contains S3-specific metrics
type S3Metrics struct {
	BucketName      string
	NumberOfObjects int64
	BucketSizeBytes int64
	AllRequests     []MetricDatapoint
	GetRequests     []MetricDatapoint
	PutRequests     []MetricDatapoint
	DeleteRequests  []MetricDatapoint
	ListRequests    []MetricDatapoint
	BytesDownloaded []MetricDatapoint
	BytesUploaded   []MetricDatapoint
}

// LambdaMetrics contains Lambda-specific metrics
type LambdaMetrics struct {
	FunctionName         string
	Invocations          []MetricDatapoint
	Duration             []MetricDatapoint
	Errors               []MetricDatapoint
	Throttles            []MetricDatapoint
	ConcurrentExecutions []MetricDatapoint
	DeadLetterErrors     []MetricDatapoint
	IteratorAge          []MetricDatapoint
}

// ResourceHealth represents the health status of a resource
type ResourceHealth struct {
	ResourceID   string
	ResourceName string
	ResourceType string
	Status       string
	LastChecked  time.Time
	Metrics      map[string]float64
	Alarms       []CloudWatchAlarm
	Issues       []string
}

// CloudWatchAlarm represents a CloudWatch alarm
type CloudWatchAlarm struct {
	AlarmName         string
	AlarmArn          string
	AlarmDescription  string
	State             string
	StateReason       string
	MetricName        string
	Namespace         string
	Threshold         float64
	ComparisonOp      string
	EvaluationPeriods int
	Period            int
	Statistic         string
	ActionsEnabled    bool
	AlarmActions      []string
	UpdatedAt         time.Time
}

// S3 Object Management

// S3ObjectInfo represents information about an S3 object or prefix
type S3ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
	Owner        string
	IsPrefix     bool
}

// S3ObjectMetadata contains detailed metadata about an S3 object
type S3ObjectMetadata struct {
	Key                  string
	Size                 int64
	LastModified         time.Time
	ContentType          string
	ETag                 string
	StorageClass         string
	ServerSideEncryption string
	Metadata             map[string]string
	VersionID            string
	CacheControl         string
	ContentDisposition   string
	ContentEncoding      string
	ContentLanguage      string
	Expires              time.Time
}

// TransferProgress tracks file transfer progress
type TransferProgress struct {
	TotalBytes       int64
	TransferredBytes int64
	PercentComplete  float64
	BytesPerSecond   float64
	ETA              time.Duration
	StartTime        time.Time
	CurrentFile      string
	Status           string
	Error            error
}

// CrossRegionCopyProgress tracks cross-region bucket copy progress
type CrossRegionCopyProgress struct {
	SourceBucket    string
	SourceRegion    string
	DestBucket      string
	DestRegion      string
	TotalObjects    int64
	CopiedObjects   int64
	FailedObjects   int64
	TotalBytes      int64
	CopiedBytes     int64
	CurrentObject   string
	PercentComplete float64
	BytesPerSecond  float64
	StartTime       time.Time
	Status          string // "preparing", "copying", "complete", "failed"
	Error           error
	FailedKeys      []string
}

// CloudWatch Logs

// LogEvent represents a CloudWatch log event
type LogEvent struct {
	Timestamp     time.Time
	Message       string
	LogStreamName string
	EventID       string
	IngestionTime time.Time
}

// LogStream represents a CloudWatch log stream
type LogStream struct {
	LogStreamName       string
	CreationTime        time.Time
	FirstEventTime      time.Time
	LastEventTime       time.Time
	LastIngestionTime   time.Time
	UploadSequenceToken string
	StoredBytes         int64
}

// LogGroup represents a CloudWatch log group
type LogGroup struct {
	LogGroupName      string
	CreationTime      time.Time
	RetentionInDays   int
	MetricFilterCount int
	StoredBytes       int64
	KmsKeyID          string
}

// Resource Inspection

// EC2InspectionResult contains detailed EC2 instance inspection data
type EC2InspectionResult struct {
	Instance          *Instance
	DetailedMetrics   *EC2Metrics
	SecurityGroups    []SecurityGroup
	Volumes           []EBSVolume
	NetworkInterfaces []NetworkInterface
	ConsoleOutput     string
	SystemLog         string
	StatusChecks      *StatusCheckResult
	Tags              map[string]string
	LaunchTime        time.Time
	Platform          string
	Architecture      string
	ImageID           string
	KeyName           string
	Monitoring        string
}

// StatusCheckResult contains EC2 status check information
type StatusCheckResult struct {
	SystemStatus    string
	InstanceStatus  string
	SystemDetails   []StatusCheckDetail
	InstanceDetails []StatusCheckDetail
	LastUpdated     time.Time
}

// StatusCheckDetail contains specific status check details
type StatusCheckDetail struct {
	Name    string
	Status  string
	Details string
}

// EBSVolume represents an EBS volume attached to an EC2 instance
type EBSVolume struct {
	VolumeID            string
	Size                int
	VolumeType          string
	IOPS                int
	Throughput          int
	State               string
	Encrypted           bool
	KmsKeyID            string
	SnapshotID          string
	AvailabilityZone    string
	AttachmentState     string
	AttachTime          time.Time
	Device              string
	DeleteOnTermination bool
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	InterfaceID     string
	PrivateIP       string
	PublicIP        string
	PrivateDNS      string
	PublicDNS       string
	SubnetID        string
	VpcID           string
	SecurityGroups  []string
	MACAddress      string
	SourceDestCheck bool
	Status          string
}

// S3InspectionResult contains detailed S3 bucket inspection data
type S3InspectionResult struct {
	Bucket            *Bucket
	Location          string
	ObjectCount       int64
	TotalSizeBytes    int64
	Versioning        bool
	Encryption        bool
	EncryptionType    string
	Replication       bool
	Lifecycle         bool
	Logging           bool
	ACL               *BucketACL
	CORS              *CORSConfiguration
	Website           *WebsiteConfiguration
	PublicAccessBlock *PublicAccessBlockConfiguration
	ObjectLockEnabled bool
	CreatedDate       time.Time
}

// BucketACL represents S3 bucket ACL
type BucketACL struct {
	Owner  string
	Grants []ACLGrant
}

// ACLGrant represents an ACL grant
type ACLGrant struct {
	Grantee     string
	GranteeType string
	Permission  string
}

// CORSConfiguration represents S3 CORS configuration
type CORSConfiguration struct {
	Rules []CORSRule
}

// CORSRule represents a single CORS rule
type CORSRule struct {
	ID             string
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	ExposeHeaders  []string
	MaxAgeSeconds  int
}

// WebsiteConfiguration represents S3 website configuration
type WebsiteConfiguration struct {
	IndexDocument string
	ErrorDocument string
	RedirectRules []RedirectRule
}

// RedirectRule represents a website redirect rule
type RedirectRule struct {
	Condition Condition
	Redirect  Redirect
}

// Condition for redirect rule
type Condition struct {
	KeyPrefixEquals             string
	HTTPErrorCodeReturnedEquals string
}

// Redirect configuration
type Redirect struct {
	Protocol             string
	HostName             string
	ReplaceKeyPrefixWith string
	ReplaceKeyWith       string
	HTTPRedirectCode     string
}

// PublicAccessBlockConfiguration represents S3 public access block settings
type PublicAccessBlockConfiguration struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

// LambdaInspectionResult contains detailed Lambda function inspection data
type LambdaInspectionResult struct {
	Function            *Function
	DetailedMetrics     *LambdaMetrics
	RecentLogs          []*LogEvent
	Configuration       *LambdaDetailedConfig
	Concurrency         *ConcurrencyConfig
	EventSourceMappings []EventSourceMapping
	Aliases             []FunctionAlias
	Versions            []FunctionVersion
}

// LambdaDetailedConfig contains detailed Lambda configuration
type LambdaDetailedConfig struct {
	CodeSize          int64
	CodeSHA256        string
	LastModified      time.Time
	LastUpdateStatus  string
	State             string
	StateReason       string
	Layers            []LayerInfo
	VpcConfig         *VPCConfig
	DeadLetterConfig  *DeadLetterConfig
	FileSystemConfigs []FileSystemConfig
	TracingConfig     string
	RevisionID        string
	PackageType       string
	Architectures     []string
}

// LayerInfo contains information about a Lambda layer
type LayerInfo struct {
	Arn                      string
	CodeSize                 int64
	SigningProfileVersionArn string
	SigningJobArn            string
}

// VPCConfig contains Lambda VPC configuration
type VPCConfig struct {
	SubnetIDs        []string
	SecurityGroupIDs []string
	VpcID            string
}

// DeadLetterConfig contains Lambda dead letter queue configuration
type DeadLetterConfig struct {
	TargetArn string
}

// FileSystemConfig contains Lambda EFS configuration
type FileSystemConfig struct {
	Arn            string
	LocalMountPath string
}

// ConcurrencyConfig contains Lambda concurrency settings
type ConcurrencyConfig struct {
	ReservedConcurrentExecutions   int
	UnreservedConcurrentExecutions int
	ProvisionedConcurrency         int
}

// EventSourceMapping represents a Lambda event source mapping
type EventSourceMapping struct {
	UUID                           string
	EventSourceArn                 string
	FunctionArn                    string
	LastModified                   time.Time
	LastProcessingResult           string
	State                          string
	StateTransitionReason          string
	BatchSize                      int
	MaximumBatchingWindowInSeconds int
}

// FunctionAlias represents a Lambda function alias
type FunctionAlias struct {
	AliasArn        string
	Name            string
	FunctionVersion string
	Description     string
	RoutingConfig   map[string]float64
	RevisionID      string
}

// FunctionVersion represents a Lambda function version
type FunctionVersion struct {
	Version      string
	Description  string
	CodeSize     int64
	CodeSHA256   string
	LastModified time.Time
}
