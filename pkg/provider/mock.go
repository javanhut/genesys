package provider

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockProvider is a mock implementation for testing
type MockProvider struct {
	name   string
	region string
}

// NewMockProvider creates a new mock provider
func NewMockProvider(name, region string) Provider {
	return &MockProvider{
		name:   name,
		region: region,
	}
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Region() string {
	return m.region
}

func (m *MockProvider) Validate() error {
	return nil
}

func (m *MockProvider) Compute() ComputeService {
	return &MockComputeService{}
}

func (m *MockProvider) Storage() StorageService {
	return &MockStorageService{}
}

func (m *MockProvider) Network() NetworkService {
	return &MockNetworkService{}
}

func (m *MockProvider) Database() DatabaseService {
	return &MockDatabaseService{}
}

func (m *MockProvider) Serverless() ServerlessService {
	return &MockServerlessService{}
}

func (m *MockProvider) StateBackend() StateBackend {
	return &MockStateBackend{}
}

func (m *MockProvider) Authenticate(ctx context.Context) error {
	return nil
}

func (m *MockProvider) Monitoring() MonitoringService {
	return &MockMonitoringService{}
}

func (m *MockProvider) Inspector() InspectorService {
	return &MockInspectorService{}
}

func (m *MockProvider) Logs() LogsService {
	return &MockLogsService{}
}

// MockComputeService mock implementation
type MockComputeService struct{}

func (m *MockComputeService) CreateInstance(ctx context.Context, config *InstanceConfig) (*Instance, error) {
	return &Instance{
		ID:        fmt.Sprintf("i-%d", time.Now().Unix()),
		Name:      config.Name,
		Type:      config.Type,
		State:     "running",
		PrivateIP: "10.0.1.10",
		Tags:      config.Tags,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockComputeService) GetInstance(ctx context.Context, id string) (*Instance, error) {
	return &Instance{
		ID:        id,
		Name:      "mock-instance",
		Type:      InstanceTypeMedium,
		State:     "running",
		PrivateIP: "10.0.1.10",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockComputeService) UpdateInstance(ctx context.Context, id string, config *InstanceConfig) error {
	return nil
}

func (m *MockComputeService) DeleteInstance(ctx context.Context, id string) error {
	return nil
}

func (m *MockComputeService) ListInstances(ctx context.Context, filters map[string]string) ([]*Instance, error) {
	return []*Instance{}, nil
}

func (m *MockComputeService) DiscoverInstances(ctx context.Context) ([]*Instance, error) {
	return []*Instance{
		{
			ID:        "i-existing-1",
			Name:      "existing-instance",
			Type:      InstanceTypeMedium,
			State:     "running",
			PrivateIP: "10.0.1.20",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
	}, nil
}

func (m *MockComputeService) AdoptInstance(ctx context.Context, id string) (*Instance, error) {
	return m.GetInstance(ctx, id)
}

// MockStorageService mock implementation
type MockStorageService struct{}

func (m *MockStorageService) CreateBucket(ctx context.Context, config *BucketConfig) (*Bucket, error) {
	return &Bucket{
		Name:         config.Name,
		Versioning:   config.Versioning,
		Encryption:   config.Encryption,
		PublicAccess: config.PublicAccess,
		Tags:         config.Tags,
		CreatedAt:    time.Now(),
	}, nil
}

func (m *MockStorageService) GetBucket(ctx context.Context, name string) (*Bucket, error) {
	return &Bucket{
		Name:       name,
		Versioning: true,
		Encryption: true,
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MockStorageService) DeleteBucket(ctx context.Context, name string) error {
	return nil
}

func (m *MockStorageService) DeleteBucketWithOptions(ctx context.Context, name string, forceDelete bool) error {
	return nil
}

func (m *MockStorageService) EmptyBucket(ctx context.Context, name string) error {
	return nil
}

func (m *MockStorageService) EmptyBucketWithOptions(ctx context.Context, name string, forceDelete bool) error {
	return nil
}

func (m *MockStorageService) ListBuckets(ctx context.Context) ([]*Bucket, error) {
	return []*Bucket{}, nil
}

func (m *MockStorageService) DiscoverBuckets(ctx context.Context) ([]*Bucket, error) {
	return []*Bucket{
		{
			Name:       "existing-bucket",
			Versioning: false,
			Encryption: true,
			CreatedAt:  time.Now().Add(-30 * 24 * time.Hour),
		},
	}, nil
}

func (m *MockStorageService) AdoptBucket(ctx context.Context, name string) (*Bucket, error) {
	return m.GetBucket(ctx, name)
}

func (m *MockStorageService) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]*S3ObjectInfo, error) {
	return []*S3ObjectInfo{}, nil
}

func (m *MockStorageService) ListObjectsRecursive(ctx context.Context, bucketName, prefix string) ([]*S3ObjectInfo, error) {
	return []*S3ObjectInfo{}, nil
}

func (m *MockStorageService) GetObject(ctx context.Context, bucketName, key string) ([]byte, error) {
	return []byte{}, nil
}

func (m *MockStorageService) GetObjectMetadata(ctx context.Context, bucketName, key string) (*S3ObjectMetadata, error) {
	return &S3ObjectMetadata{Key: key}, nil
}

func (m *MockStorageService) PutObject(ctx context.Context, bucketName, key string, data []byte, contentType string) error {
	return nil
}

func (m *MockStorageService) DeleteObject(ctx context.Context, bucketName, key string) error {
	return nil
}

func (m *MockStorageService) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	return nil
}

func (m *MockStorageService) UploadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *TransferProgress) error {
	return nil
}

func (m *MockStorageService) DownloadFile(ctx context.Context, bucketName, key, localPath string, progress chan<- *TransferProgress) error {
	return nil
}

func (m *MockStorageService) SyncDirectory(ctx context.Context, bucketName, prefix, localPath, direction string) error {
	return nil
}

func (m *MockStorageService) GeneratePresignedURL(ctx context.Context, bucketName, key string, expiresIn int) (string, error) {
	return "https://example.com/presigned", nil
}

func (m *MockStorageService) CopyObjectCrossRegion(ctx context.Context, srcBucket, srcKey, dstRegion, dstBucket, dstKey string) error {
	return nil
}

func (m *MockStorageService) CopyBucketCrossRegion(ctx context.Context, srcBucket, dstRegion, dstBucket, prefix string, progress chan<- *CrossRegionCopyProgress) error {
	if progress != nil {
		progress <- &CrossRegionCopyProgress{
			SourceBucket:    srcBucket,
			DestBucket:      dstBucket,
			DestRegion:      dstRegion,
			Status:          "complete",
			PercentComplete: 100,
		}
	}
	return nil
}

func (m *MockStorageService) ListBucketsInRegion(ctx context.Context, region string) ([]*Bucket, error) {
	return []*Bucket{
		{
			Name:      "mock-bucket-" + region,
			Region:    region,
			CreatedAt: time.Now(),
		},
	}, nil
}

func (m *MockStorageService) GetBucketRegion(ctx context.Context, bucketName string) (string, error) {
	return "us-east-1", nil
}

// MockNetworkService mock implementation
type MockNetworkService struct{}

func (m *MockNetworkService) CreateNetwork(ctx context.Context, config *NetworkConfig) (*Network, error) {
	return &Network{
		ID:        fmt.Sprintf("vpc-%d", time.Now().Unix()),
		Name:      config.Name,
		CIDR:      config.CIDR,
		Tags:      config.Tags,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockNetworkService) GetNetwork(ctx context.Context, id string) (*Network, error) {
	return &Network{
		ID:        id,
		Name:      "mock-network",
		CIDR:      "10.0.0.0/16",
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockNetworkService) CreateSubnet(ctx context.Context, networkID string, config *SubnetConfig) (*Subnet, error) {
	return &Subnet{
		ID:        fmt.Sprintf("subnet-%d", time.Now().Unix()),
		Name:      config.Name,
		CIDR:      config.CIDR,
		NetworkID: networkID,
		Public:    config.Public,
		AZ:        config.AZ,
	}, nil
}

func (m *MockNetworkService) CreateSecurityGroup(ctx context.Context, config *SecurityGroupConfig) (*SecurityGroup, error) {
	return &SecurityGroup{
		ID:          fmt.Sprintf("sg-%d", time.Now().Unix()),
		Name:        config.Name,
		Description: config.Description,
		Rules:       config.Rules,
		Tags:        config.Tags,
	}, nil
}

func (m *MockNetworkService) DiscoverNetworks(ctx context.Context) ([]*Network, error) {
	return []*Network{
		{
			ID:        "vpc-existing",
			Name:      "default-vpc",
			CIDR:      "172.16.0.0/16",
			CreatedAt: time.Now().Add(-90 * 24 * time.Hour),
		},
	}, nil
}

func (m *MockNetworkService) AdoptNetwork(ctx context.Context, id string) (*Network, error) {
	return m.GetNetwork(ctx, id)
}

// MockDatabaseService mock implementation
type MockDatabaseService struct{}

func (m *MockDatabaseService) CreateDatabase(ctx context.Context, config *DatabaseConfig) (*Database, error) {
	return &Database{
		ID:        fmt.Sprintf("db-%d", time.Now().Unix()),
		Name:      config.Name,
		Engine:    config.Engine,
		Version:   config.Version,
		Size:      config.Size,
		Storage:   config.Storage,
		MultiAZ:   config.MultiAZ,
		Endpoint:  fmt.Sprintf("%s.mock.rds.amazonaws.com", config.Name),
		Port:      5432,
		Tags:      config.Tags,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockDatabaseService) GetDatabase(ctx context.Context, id string) (*Database, error) {
	return &Database{
		ID:        id,
		Name:      "mock-database",
		Engine:    "postgres",
		Version:   "14",
		Size:      DatabaseSizeMedium,
		Storage:   100,
		Endpoint:  "mock-db.mock.rds.amazonaws.com",
		Port:      5432,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockDatabaseService) UpdateDatabase(ctx context.Context, id string, config *DatabaseConfig) error {
	return nil
}

func (m *MockDatabaseService) DeleteDatabase(ctx context.Context, id string) error {
	return nil
}

func (m *MockDatabaseService) DiscoverDatabases(ctx context.Context) ([]*Database, error) {
	return []*Database{
		{
			ID:        "db-existing",
			Name:      "production-db",
			Engine:    "postgres",
			Version:   "13",
			Size:      DatabaseSizeLarge,
			Storage:   500,
			Endpoint:  "prod-db.mock.rds.amazonaws.com",
			Port:      5432,
			CreatedAt: time.Now().Add(-180 * 24 * time.Hour),
		},
	}, nil
}

func (m *MockDatabaseService) AdoptDatabase(ctx context.Context, id string) (*Database, error) {
	return m.GetDatabase(ctx, id)
}

// MockServerlessService mock implementation
type MockServerlessService struct{}

func (m *MockServerlessService) CreateFunction(ctx context.Context, config *FunctionConfig) (*Function, error) {
	return &Function{
		ID:          fmt.Sprintf("fn-%d", time.Now().Unix()),
		Name:        config.Name,
		Runtime:     config.Runtime,
		Handler:     config.Handler,
		Memory:      config.Memory,
		Timeout:     config.Timeout,
		Environment: config.Environment,
		URL:         fmt.Sprintf("https://%s.lambda-url.us-east-1.on.aws/", config.Name),
		Tags:        config.Tags,
		CreatedAt:   time.Now(),
	}, nil
}

func (m *MockServerlessService) UpdateFunction(ctx context.Context, id string, config *FunctionConfig) error {
	return nil
}

func (m *MockServerlessService) DeleteFunction(ctx context.Context, id string) error {
	return nil
}

func (m *MockServerlessService) InvokeFunction(ctx context.Context, id string, payload []byte) ([]byte, error) {
	return []byte(`{"status": "success", "message": "Hello from mock function"}`), nil
}

func (m *MockServerlessService) DiscoverFunctions(ctx context.Context) ([]*Function, error) {
	return []*Function{
		{
			ID:        "fn-existing",
			Name:      "api-handler",
			Runtime:   "python3.11",
			Handler:   "main.handler",
			Memory:    512,
			Timeout:   30,
			CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		},
	}, nil
}

func (m *MockServerlessService) AdoptFunction(ctx context.Context, id string) (*Function, error) {
	return &Function{
		ID:        id,
		Name:      "adopted-function",
		Runtime:   "python3.11",
		Handler:   "main.handler",
		Memory:    256,
		Timeout:   60,
		CreatedAt: time.Now(),
	}, nil
}

// MockStateBackend mock implementation
type MockStateBackend struct {
	state map[string]*State
	mu    sync.RWMutex
}

func (m *MockStateBackend) Init(ctx context.Context) error {
	if m.state == nil {
		m.state = make(map[string]*State)
	}
	return nil
}

func (m *MockStateBackend) Lock(ctx context.Context, key string) error {
	m.mu.Lock()
	return nil
}

func (m *MockStateBackend) Unlock(ctx context.Context, key string) error {
	m.mu.Unlock()
	return nil
}

func (m *MockStateBackend) Read(ctx context.Context, key string) (*State, error) {
	if s, ok := m.state[key]; ok {
		return s, nil
	}
	return &State{
		Version:   1,
		Resources: make(map[string]interface{}),
		Outputs:   make(map[string]interface{}),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockStateBackend) Write(ctx context.Context, key string, state *State) error {
	if m.state == nil {
		m.state = make(map[string]*State)
	}
	m.state[key] = state
	return nil
}

// MockMonitoringService mock implementation
type MockMonitoringService struct{}

func (m *MockMonitoringService) GetResourceMetrics(ctx context.Context, resourceType, resourceID string, period string) (*MetricsData, error) {
	return &MetricsData{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Period:       period,
		Datapoints:   []MetricDatapoint{},
	}, nil
}

func (m *MockMonitoringService) GetEC2Metrics(ctx context.Context, instanceID string, period string) (*EC2Metrics, error) {
	return &EC2Metrics{
		InstanceID:     instanceID,
		CPUUtilization: []MetricDatapoint{{Timestamp: time.Now(), Value: 45.2, Unit: "Percent"}},
		NetworkIn:      []MetricDatapoint{{Timestamp: time.Now(), Value: 1024000, Unit: "Bytes"}},
		NetworkOut:     []MetricDatapoint{{Timestamp: time.Now(), Value: 2048000, Unit: "Bytes"}},
	}, nil
}

func (m *MockMonitoringService) GetS3Metrics(ctx context.Context, bucketName string, period string) (*S3Metrics, error) {
	return &S3Metrics{
		BucketName:      bucketName,
		NumberOfObjects: 100,
		BucketSizeBytes: 10485760,
	}, nil
}

func (m *MockMonitoringService) GetLambdaMetrics(ctx context.Context, functionName string, period string) (*LambdaMetrics, error) {
	return &LambdaMetrics{
		FunctionName: functionName,
		Invocations:  []MetricDatapoint{{Timestamp: time.Now(), Value: 1234, Unit: "Count"}},
		Duration:     []MetricDatapoint{{Timestamp: time.Now(), Value: 145.5, Unit: "Milliseconds"}},
	}, nil
}

func (m *MockMonitoringService) GetResourceHealth(ctx context.Context, resourceType, resourceID string) (*ResourceHealth, error) {
	return &ResourceHealth{
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Status:       "healthy",
		LastChecked:  time.Now(),
		Metrics:      map[string]float64{"cpu": 45.2},
	}, nil
}

func (m *MockMonitoringService) GetAllResourcesHealth(ctx context.Context) ([]*ResourceHealth, error) {
	return []*ResourceHealth{}, nil
}

func (m *MockMonitoringService) ListResourceAlarms(ctx context.Context, resourceID string) ([]*CloudWatchAlarm, error) {
	return []*CloudWatchAlarm{}, nil
}

func (m *MockMonitoringService) GetAlarmState(ctx context.Context, alarmName string) (string, error) {
	return "OK", nil
}

// MockInspectorService mock implementation
type MockInspectorService struct{}

func (m *MockInspectorService) InspectEC2Instance(ctx context.Context, instanceID string) (*EC2InspectionResult, error) {
	return &EC2InspectionResult{
		Instance: &Instance{
			ID:    instanceID,
			Name:  "mock-instance",
			State: "running",
		},
	}, nil
}

func (m *MockInspectorService) GetEC2ConsoleOutput(ctx context.Context, instanceID string) (string, error) {
	return "Console output...", nil
}

func (m *MockInspectorService) GetEC2SystemLog(ctx context.Context, instanceID string) (string, error) {
	return "System log...", nil
}

func (m *MockInspectorService) InspectS3Bucket(ctx context.Context, bucketName string) (*S3InspectionResult, error) {
	return &S3InspectionResult{
		Bucket: &Bucket{
			Name: bucketName,
		},
		ObjectCount:    100,
		TotalSizeBytes: 10485760,
	}, nil
}

func (m *MockInspectorService) AnalyzeBucketSize(ctx context.Context, bucketName string) (int64, int64, error) {
	return 100, 10485760, nil
}

func (m *MockInspectorService) GetBucketACL(ctx context.Context, bucketName string) (*BucketACL, error) {
	return &BucketACL{Owner: "mock-owner"}, nil
}

func (m *MockInspectorService) GetBucketCORS(ctx context.Context, bucketName string) (*CORSConfiguration, error) {
	return &CORSConfiguration{}, nil
}

func (m *MockInspectorService) InspectLambdaFunction(ctx context.Context, functionName string) (*LambdaInspectionResult, error) {
	return &LambdaInspectionResult{
		DetailedMetrics: &LambdaMetrics{FunctionName: functionName},
	}, nil
}

func (m *MockInspectorService) GetLambdaConfiguration(ctx context.Context, functionName string) (*LambdaDetailedConfig, error) {
	return &LambdaDetailedConfig{
		State:            "Active",
		LastUpdateStatus: "Successful",
	}, nil
}

// MockLogsService mock implementation
type MockLogsService struct{}

func (m *MockLogsService) ListLogStreams(ctx context.Context, logGroupName string) ([]*LogStream, error) {
	return []*LogStream{}, nil
}

func (m *MockLogsService) ListLogGroups(ctx context.Context) ([]*LogGroup, error) {
	return []*LogGroup{}, nil
}

func (m *MockLogsService) GetLogEvents(ctx context.Context, logGroupName, logStreamName string, startTime, endTime int64) ([]*LogEvent, error) {
	return []*LogEvent{}, nil
}

func (m *MockLogsService) GetLambdaLogs(ctx context.Context, functionName string, startTime, endTime int64, limit int) ([]*LogEvent, error) {
	return []*LogEvent{
		{
			Timestamp:     time.Now(),
			Message:       "Mock log message",
			LogStreamName: "mock-stream",
		},
	}, nil
}

func (m *MockLogsService) TailLambdaLogs(ctx context.Context, functionName string) (<-chan *LogEvent, <-chan error) {
	eventChan := make(chan *LogEvent)
	errChan := make(chan error)
	close(eventChan)
	close(errChan)
	return eventChan, errChan
}
