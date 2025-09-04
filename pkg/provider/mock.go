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