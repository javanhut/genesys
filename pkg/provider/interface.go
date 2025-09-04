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