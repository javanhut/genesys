# Genesys

A simplicity-first Infrastructure as a Service tool that focuses on outcomes rather than resources.

## Overview

Genesys is a Go-based tool designed to replace the complexity of traditional IaC tools with a simple, outcome-focused approach. Instead of defining low-level resources, users specify what they want to achieve (e.g., 'static-site', 'database', 'function').

## Key Features

- **Outcome-based deployment**: Ask for what you want, not how to build it
- **Discovery-first**: Always check for existing resources before creating new ones
- **Human-readable plans**: Plain English descriptions with IAM forecasting
- **Provider-agnostic**: Same code works across AWS, GCP, Azure, and more
- **State-less for users**: State automatically managed in cloud provider
- **Preview by default**: Safe by default, explicit apply

## Phase 0 MVP Implementation

This implementation includes:

- **CLI Scaffold**: Single `genesys` command with `execute` and `interact` modes  
- **Intent Parser**: Supports `bucket`, `network`, `function`, `static-site`, `database`, `api`, `webapp`  
- **Human Plans**: English descriptions with cost estimates and IAM permissions  
- **Provider Interface**: Pluggable architecture for any cloud provider  
- **Configuration Support**: YAML and TOML configuration files  
- **Mock Provider**: For testing and development  

## Quick Start

### Build from source
```bash
git clone https://github.com/javanhut/genesys
cd genesys
go build -o genesys ./cmd/genesys
```

### Basic Usage

```bash
# Preview what would happen (default behavior)
./genesys execute bucket my-data-bucket

# Deploy a static website
./genesys execute static-site domain=example.com --apply

# Create a database with parameters
./genesys execute database mydb engine=postgres size=large

# Interactive mode
./genesys interact

# Discover existing resources
./genesys discover

# JSON output for automation
./genesys execute function my-api runtime=python3.11 --output json

# Execute from configuration file
./genesys execute --config examples/simple-website.yaml
./genesys execute --config examples/serverless-api.yaml --apply
```

### Parameter Syntax

Parameters can be specified as `key=value` pairs:

```bash
# Function with parameters
./genesys execute function my-api runtime=python3.11 memory=512 trigger=http

# Static site with custom domain
./genesys execute static-site domain=my-site.com cdn=true

# Database with specific configuration
./genesys execute database prod-db engine=postgres size=large storage=500
```

### Configuration Files

Create a `genesys.yaml` file for complex infrastructure:

```yaml
provider: aws
region: us-east-1

resources:
  storage:
    - name: app-data
      type: bucket
      versioning: true
      encryption: true
      
  database:
    - name: user-db
      engine: postgres
      size: medium
      multi_az: true
      
  serverless:
    - name: api-handler
      runtime: python3.11
      memory: 512
      triggers:
        - type: http
```

Then execute with:
```bash
./genesys execute --config genesys.yaml
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