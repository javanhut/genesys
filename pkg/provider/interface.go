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

	// Monitoring and Management
	Monitoring() MonitoringService
	Inspector() InspectorService
	Logs() LogsService

	// Authentication
	Authenticate(ctx context.Context) error
}

// ComputeService handles all compute resources
type ComputeService interface {
	CreateInstance(ctx context.Context, config *InstanceConfig) (*Instance, error)
	GetInstance(ctx context.Context, id string) (*Instance, error)
	UpdateInstance(ctx context.Context, id string, config *InstanceConfig) error
	DeleteInstance(ctx context.Context, id string) error
	ListInstances(ctx context.Context, filters map[string]string) ([]*Instance, error)

	// Discovery
	DiscoverInstances(ctx context.Context) ([]*Instance, error)
	AdoptInstance(ctx context.Context, id string) (*Instance, error)
}

// StorageService handles storage resources
type StorageService interface {
	CreateBucket(ctx context.Context, config *BucketConfig) (*Bucket, error)
	GetBucket(ctx context.Context, name string) (*Bucket, error)
	DeleteBucket(ctx context.Context, name string) error
	DeleteBucketWithOptions(ctx context.Context, name string, forceDelete bool) error
	EmptyBucket(ctx context.Context, name string) error
	EmptyBucketWithOptions(ctx context.Context, name string, forceDelete bool) error
	ListBuckets(ctx context.Context) ([]*Bucket, error)

	// Discovery
	DiscoverBuckets(ctx context.Context) ([]*Bucket, error)
	AdoptBucket(ctx context.Context, name string) (*Bucket, error)

	// Object Operations
	ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]*S3ObjectInfo, error)
	ListObjectsRecursive(ctx context.Context, bucketName, prefix string) ([]*S3ObjectInfo, error)
	GetObject(ctx context.Context, bucketName, key string) ([]byte, error)
	GetObjectMetadata(ctx context.Context, bucketName, key string) (*S3ObjectMetadata, error)
	PutObject(ctx context.Context, bucketName, key string, data []byte, contentType string) error
	DeleteObject(ctx context.Context, bucketName, key string) error
	CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error

	// File Transfer
	UploadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *TransferProgress) error
	DownloadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *TransferProgress) error
	SyncDirectory(ctx context.Context, bucketName, prefix, localPath, direction string) error

	// Advanced
	GeneratePresignedURL(ctx context.Context, bucketName, key string, expiresIn int) (string, error)
}

// NetworkService handles networking resources
type NetworkService interface {
	CreateNetwork(ctx context.Context, config *NetworkConfig) (*Network, error)
	GetNetwork(ctx context.Context, id string) (*Network, error)
	CreateSubnet(ctx context.Context, networkID string, config *SubnetConfig) (*Subnet, error)
	CreateSecurityGroup(ctx context.Context, config *SecurityGroupConfig) (*SecurityGroup, error)

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

	// Discovery
	DiscoverFunctions(ctx context.Context) ([]*Function, error)
	AdoptFunction(ctx context.Context, id string) (*Function, error)
}

// StateBackend handles state storage
type StateBackend interface {
	Init(ctx context.Context) error
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
	Read(ctx context.Context, key string) (*State, error)
	Write(ctx context.Context, key string, state *State) error
}

// MonitoringService handles resource monitoring and metrics
type MonitoringService interface {
	// Metrics Collection
	GetResourceMetrics(ctx context.Context, resourceType, resourceID string, period string) (*MetricsData, error)
	GetEC2Metrics(ctx context.Context, instanceID string, period string) (*EC2Metrics, error)
	GetS3Metrics(ctx context.Context, bucketName string, period string) (*S3Metrics, error)
	GetLambdaMetrics(ctx context.Context, functionName string, period string) (*LambdaMetrics, error)

	// Resource Health
	GetResourceHealth(ctx context.Context, resourceType, resourceID string) (*ResourceHealth, error)
	GetAllResourcesHealth(ctx context.Context) ([]*ResourceHealth, error)

	// Alarms
	ListResourceAlarms(ctx context.Context, resourceID string) ([]*CloudWatchAlarm, error)
	GetAlarmState(ctx context.Context, alarmName string) (string, error)
}

// InspectorService handles deep resource inspection
type InspectorService interface {
	// EC2 Inspection
	InspectEC2Instance(ctx context.Context, instanceID string) (*EC2InspectionResult, error)
	GetEC2ConsoleOutput(ctx context.Context, instanceID string) (string, error)
	GetEC2SystemLog(ctx context.Context, instanceID string) (string, error)

	// S3 Inspection
	InspectS3Bucket(ctx context.Context, bucketName string) (*S3InspectionResult, error)
	AnalyzeBucketSize(ctx context.Context, bucketName string) (int64, int64, error)
	GetBucketACL(ctx context.Context, bucketName string) (*BucketACL, error)
	GetBucketCORS(ctx context.Context, bucketName string) (*CORSConfiguration, error)

	// Lambda Inspection
	InspectLambdaFunction(ctx context.Context, functionName string) (*LambdaInspectionResult, error)
	GetLambdaConfiguration(ctx context.Context, functionName string) (*LambdaDetailedConfig, error)
}

// LogsService handles CloudWatch Logs
type LogsService interface {
	// Log Streams
	ListLogStreams(ctx context.Context, logGroupName string) ([]*LogStream, error)
	ListLogGroups(ctx context.Context) ([]*LogGroup, error)

	// Log Events
	GetLogEvents(ctx context.Context, logGroupName, logStreamName string, startTime, endTime int64) ([]*LogEvent, error)
	GetLambdaLogs(ctx context.Context, functionName string, startTime, endTime int64, limit int) ([]*LogEvent, error)

	// Real-time Streaming
	TailLambdaLogs(ctx context.Context, functionName string) (<-chan *LogEvent, <-chan error)
}
