# Genesys Architecture

## Division of Responsibilities

### Core Packages (`pkg/`)

#### Provider Layer (`pkg/provider/`)
**Responsibility**: Cloud provider abstraction and implementations
- `interface.go` - Universal provider interface
- `resources.go` - Universal resource models
- `registry.go` - Provider registration and discovery
- `mock.go` - Mock implementation for testing
- `aws/` - AWS provider implementation (future)
- `gcp/` - GCP provider implementation (future)
- `azure/` - Azure provider implementation (future)

#### Configuration Layer (`pkg/config/`)
**Responsibility**: Configuration parsing and validation
- `config.go` - YAML/TOML configuration parsing
- `validation.go` - Configuration validation rules
- `defaults.go` - Default value application

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

## Future Extensions

### Phase 1: Real Providers
- Implement AWS provider with real SDK calls
- Add state management with S3/DynamoDB backend
- Implement plan execution

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