# Genesys

A simplicity-first Infrastructure as a Service tool that focuses on outcomes rather than resources. It provides a discovery-first approach to cloud resource management with human-readable plans.

## Key Features

- **Interactive Workflows** - Guided prompts for resource creation without manual config writing
- **Multi-Cloud Support** - AWS, GCP, Azure, Tencent Cloud with unified interface
- **Configuration-Driven** - YAML-based resource lifecycle management  
- **Dry-Run Capability** - Preview all changes before deployment
- **Direct API Integration** - Fast performance without heavy SDKs
- **Provider-Agnostic** - Write once, deploy anywhere

## Quick Start

### Installation

```bash
# Build from source
git clone <repository-url>
cd genesys
go build -o genesys ./cmd/genesys
```

### Configure Provider

```bash
# Interactive provider setup
genesys config setup
```

### Create Your First Resource

```bash
# Start interactive workflow
genesys interact
# Select provider: aws
# Select resource: S3 Storage Bucket  
# Follow prompts to configure bucket

# Preview deployment
genesys execute s3-mybucket-*.yaml --dry-run

# Deploy the resource
genesys execute s3-mybucket-*.yaml

# List your resources
genesys list resources

# Clean up when done
genesys execute deletion s3-mybucket-*.yaml
```

## Available Commands

### Interactive Mode
```bash
genesys interact                    # Start interactive resource creation
```

### Configuration Management
```bash
genesys config setup                # Configure cloud provider credentials
genesys config list                 # List configured providers
genesys config show aws             # Show provider configuration
genesys config default aws          # Set default provider
```

### Resource Deployment
```bash
genesys execute config.yaml                    # Deploy resources
genesys execute config.yaml --dry-run          # Preview changes
genesys execute deletion config.yaml           # Delete resources
```

### Resource Discovery
```bash
genesys list resources              # List all resources (alias: discover)
genesys list --service storage      # List only storage resources
genesys list --output json          # JSON output format
```

## Supported Resources

### Currently Implemented
- **S3 Storage Buckets** - Complete lifecycle with versioning, encryption, lifecycle policies

### Planned
- **Compute Instances** - Virtual machines across providers
- **Databases** - Managed database services
- **Functions** - Serverless compute
- **Networks** - VPCs, subnets, security groups

## Configuration Example

Generated S3 bucket configuration:

```yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: my-app-storage
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: development
        ManagedBy: Genesys
        Purpose: application-data
      lifecycle:
        archive_after_days: 90
        delete_after_days: 365
policies:
  require_encryption: true
  no_public_buckets: true
  require_tags:
    - Environment
    - ManagedBy
```

## Example Outputs

### Bucket Creation Plan
```
Plan: Deploy S3 Bucket 'my-bucket'
==================================

What will happen:
> 1. Create S3 bucket 'my-bucket'
     -> Store your application data securely
> 2. Enable versioning on bucket
     -> Protect against accidental deletion
> 3. Enable encryption with AWS managed keys
     -> Secure data at rest
> 4. Block all public access
     -> Prevent data exposure

Permissions needed:
- s3:CreateBucket
- s3:PutBucketVersioning
- s3:PutBucketEncryption
- s3:PutBucketPublicAccessBlock

Cost estimate:
- Monthly: $5.00 USD
- Confidence: medium

Time to complete: 30 seconds
```

## Architecture

### Provider Interface
All cloud providers implement the same interface, allowing truly provider-agnostic code:

```go
type Provider interface {
    Compute() ComputeService
    Storage() StorageService
    Network() NetworkService
    Database() DatabaseService
    Serverless() ServerlessService
}
```

### Universal Resource Types
Resources are abstracted across providers:
- `small|medium|large|xlarge` instance types
- `bucket` storage that works on S3, GCS, Azure Blob
- `postgres|mysql` databases that map to RDS, CloudSQL, etc.

### Intent-Driven Architecture
Users express intent, not implementation:
```bash
# User says what they want
genesys execute static-site domain=example.com

# Genesys figures out how (S3 + CloudFront + Route53 on AWS,
# Storage + CDN + DNS on GCP, etc.)
```

## Project Structure

```
genesys/
├── cmd/genesys/           # CLI entry point
│   ├── main.go
│   └── commands/          # Command implementations
├── pkg/                   # Core packages
│   ├── provider/          # Provider interface and implementations  
│   ├── config/            # YAML/TOML configuration with validation
│   ├── intent/            # Intent parsing and interpretation
│   ├── planner/           # Plan generation and formatting
│   ├── executor/          # Plan execution (future)
│   └── discovery/         # Resource discovery (future)
├── internal/              # Internal utilities (future)
├── examples/              # Configuration examples
│   ├── simple-website.yaml
│   ├── serverless-api.yaml
│   ├── web-application.toml
│   └── multi-cloud.yaml
├── ARCHITECTURE.md        # Detailed architecture documentation
└── README.md
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/intent -v
go test ./pkg/planner -v
```

## Next Steps (Beyond MVP)

- **Real provider implementations** (AWS SDK integration)
- **State management** (S3/DynamoDB backend)
- **Plan execution** (apply functionality)
- **Resource adoption** (import existing resources)
- **Interactive improvements** (better UX)
- **Multi-cloud support** (GCP, Azure providers)

## Contributing

This is Phase 0 implementation focusing on the core architecture and user experience. The foundation is built to support the full vision outlined in the design documents.

## Philosophy

Genesys follows these principles:

1. **Simplicity First**: Complex infrastructure should be simple to deploy
2. **Outcome Focused**: Users specify what they want, not how to build it
3. **Discovery Over Creation**: Always check what exists first
4. **Provider Agnostic**: Write once, run anywhere
5. **Safe by Default**: Preview first, apply explicitly
6. **Human Readable**: Plans anyone can understand

---

*"Infrastructure deployment should be as simple as describing what you want, not how to build it."*