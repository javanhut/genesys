# Genesys Architecture

## Division of Responsibilities

### Core Packages (`pkg/`)

#### Provider Layer (`pkg/provider/`)
**Responsibility**: Cloud provider abstraction and implementations
- `interface.go` - Universal provider interface
- `resources.go` - Universal resource models
- `registry.go` - Provider registration and discovery
- `mock.go` - Mock implementation for testing
- `aws/` - AWS provider implementation with direct API calls
  - `client.go` - Direct HTTP client with signature v4 authentication
  - `provider.go` - Main AWS provider implementation
  - `storage.go` - S3 service implementation
  - `compute.go` - EC2 service implementation
  - `database.go` - RDS service implementation
  - `network.go` - VPC service implementation
  - `serverless.go` - Lambda service implementation

#### Configuration Layer (`pkg/config/`)
**Responsibility**: Configuration parsing and validation
- `config.go` - YAML/TOML configuration parsing
- `validation.go` - Configuration validation rules
- `defaults.go` - Default value application
- `interactive.go` - Interactive provider configuration
- `interactive_aws.go` - AWS-specific interactive configuration
- `interactive_gcp.go` - GCP-specific interactive configuration
- `interactive_azure.go` - Azure-specific interactive configuration
- `interactive_tencent.go` - Tencent Cloud interactive configuration
- `interactive_storage.go` - Configuration storage and validation
- `s3_interactive.go` - S3-specific interactive configuration

#### Intent Layer (`pkg/intent/`)
**Responsibility**: User intent parsing and interpretation
- `parser.go` - Command line intent parsing
- `types.go` - Intent type definitions
- `validation.go` - Intent validation

#### Planning Layer (`pkg/planner/`)
**Responsibility**: Execution plan generation
- `planner.go` - Main planner logic
- `plan.go` - Plan data structures and formatting
- `cost.go` - Cost estimation
- `iam.go` - IAM permission forecasting

#### Execution Layer (`pkg/executor/`) - Future
**Responsibility**: Plan execution and state management
- `executor.go` - Plan execution engine
- `state.go` - State management
- `rollback.go` - Rollback functionality

#### Discovery Layer (`pkg/discovery/`) - Future
**Responsibility**: Existing resource discovery and adoption
- `scanner.go` - Resource discovery
- `adopter.go` - Resource adoption
- `analyzer.go` - Resource analysis

### Command Layer (`cmd/genesys/`)

#### Main Entry Point
- `main.go` - CLI setup and command registration

#### Commands (`cmd/genesys/commands/`)
**Responsibility**: CLI command implementations
- `execute.go` - Execute command (plan/apply)
- `interact.go` - Interactive mode
- `discover.go` - Resource discovery
- `version.go` - Version information

### Internal Packages (`internal/`) - Future

#### Utilities (`internal/utils/`)
**Responsibility**: Internal utility functions
- `strings.go` - String manipulation utilities
- `files.go` - File system utilities
- `crypto.go` - Cryptographic utilities

#### Logging (`internal/logger/`)
**Responsibility**: Structured logging
- `logger.go` - Logging interface and implementation

### Configuration and Examples

#### Examples (`examples/`)
**Responsibility**: Example configurations for different use cases
- `simple-website.yaml` - Basic static site
- `serverless-api.yaml` - API with serverless functions
- `web-application.toml` - Full web application stack
- `multi-cloud.yaml` - Multi-cloud deployment example

#### Documentation (`docs/`) - Future
**Responsibility**: Project documentation
- `providers/` - Provider-specific documentation
- `examples/` - Usage examples and tutorials
- `api/` - API documentation

## Data Flow

```
CLI Command → Intent Parser → Configuration Loader → Planner → Provider → Plan Output
     ↓              ↓              ↓              ↓         ↓         ↓
  execute        Parse intent   Load config   Generate   Query     Format
  bucket         into struct    from file     steps      provider  human plan
  my-bucket                                                        
```

## Provider Abstraction

### Universal Resource Model
```go
// Any cloud provider implements these interfaces
Provider -> ComputeService -> Instance (t3.medium on AWS, n1-standard-2 on GCP)
         -> StorageService  -> Bucket (S3 on AWS, Cloud Storage on GCP)
         -> DatabaseService -> Database (RDS on AWS, Cloud SQL on GCP)
```

### Provider Registration
```go
// Providers register themselves at startup
func init() {
    provider.Register("aws", aws.NewFactory)
    provider.Register("gcp", gcp.NewFactory)
    provider.Register("azure", azure.NewFactory)
}
```

### Configuration Translation
```yaml
# User writes universal config
compute:
  - name: web-server
    type: medium
    image: ubuntu-lts

# Genesys translates per provider:
# AWS: t3.medium with ami-xxxxxx (Ubuntu 22.04)
# GCP: n1-standard-2 with ubuntu-2204-lts
# Azure: Standard_B2s with Ubuntu 22.04-LTS
```

## S3 Interactive Workflow

### Current Implementation

The S3 workflow demonstrates the complete Genesys resource lifecycle:

#### Interactive Configuration Generation
```go
// pkg/config/s3_interactive.go
func (isc *InteractiveS3Config) CreateBucketConfig() (*S3BucketConfig, string, error)
```
- Guided prompts for bucket settings
- Validation of bucket names and regions
- Security-first defaults (encryption, no public access)
- Automatic tagging with Environment, ManagedBy, Purpose
- Lifecycle policy configuration

#### Direct API Implementation
```go
// pkg/provider/aws/storage.go
func (s *StorageService) CreateBucket(ctx context.Context, config *BucketConfig) (*Bucket, error)
```
- Direct HTTP calls to AWS S3 API
- AWS Signature Version 4 authentication
- No heavy SDK dependencies for fast builds
- Complete bucket lifecycle management

#### Configuration-Driven Execution
```go
// cmd/genesys/commands/execute.go
func executeS3Config(ctx context.Context, configPath string) error
```
- YAML configuration file parsing
- Dry-run capability for safe previews
- Resource creation and deletion
- Error handling and validation

### Workflow Steps

1. **Interactive Setup**: `genesys interact`
   - Provider selection (aws, gcp, azure, tencent)
   - Resource type selection (S3 Storage Bucket)
   - Configuration through guided prompts
   - YAML generation (s3-bucketname-timestamp.yaml)

2. **Preview**: `genesys execute config.yaml --dry-run`
   - Parse configuration file
   - Display planned changes
   - No actual resource creation

3. **Deploy**: `genesys execute config.yaml`
   - Validate AWS credentials
   - Create S3 bucket with all settings
   - Apply versioning, encryption, tags
   - Configure lifecycle policies

4. **List**: `genesys list --service storage`
   - Discover existing S3 buckets
   - Show bucket details and metadata

5. **Delete**: `genesys execute deletion config.yaml`
   - Remove bucket and all contents
   - Clean resource cleanup

## Implementation Status

### Completed Features
- **Interactive Configuration**: Full S3 bucket configuration wizard
- **Multi-Cloud Credentials**: AWS, GCP, Azure, Tencent provider setup
- **Direct API Integration**: AWS S3 without SDK dependencies
- **YAML Configuration**: Complete resource specification
- **Dry-Run Capability**: Safe preview before deployment
- **Resource Discovery**: List existing buckets and resources
- **Command Line Integration**: Comprehensive CLI with help text

### Future Extensions

### Phase 1: Additional Resources
- Compute instances (EC2, GCE, Azure VMs)
- Databases (RDS, Cloud SQL, Azure Database)
- Serverless functions (Lambda, Cloud Functions, Azure Functions)
- Networking (VPC, subnets, security groups)

### Phase 2: Multi-Cloud
- Add GCP and Azure providers
- Cross-cloud resource dependencies
- Multi-cloud cost comparison

### Phase 3: Advanced Features
- Resource adoption and import
- Policy enforcement
- Cost optimization recommendations
- Automated testing and validation

## Design Principles

1. **Single Responsibility**: Each package has one clear purpose
2. **Interface Segregation**: Small, focused interfaces
3. **Dependency Inversion**: Depend on abstractions, not concretions
4. **Provider Agnostic**: Core logic independent of any specific provider
5. **Testability**: All components are easily testable in isolation