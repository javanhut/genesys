# Configuration Examples

Real-world examples of Genesys configurations for common use cases.

## S3 Bucket Configurations

### Basic S3 Bucket

Simple bucket with encryption and versioning:

```yaml
# s3-basic-bucket.yaml
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
policies:
  require_encryption: true
  no_public_buckets: true
  require_tags:
    - Environment
    - ManagedBy
```

**Usage**:
```bash
# Deploy
genesys execute s3-basic-bucket.yaml

# Delete
genesys execute deletion s3-basic-bucket.yaml
```

### S3 Bucket with Lifecycle Policies

Bucket with automatic archiving and deletion:

```yaml
# s3-lifecycle-bucket.yaml
provider: aws
region: us-west-2
resources:
  storage:
    - name: log-storage-bucket
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: log-storage
        Project: web-application
        CostCenter: engineering
      lifecycle:
        archive_after_days: 30
        delete_after_days: 365
policies:
  require_encryption: true
  no_public_buckets: true
  require_tags:
    - Environment
    - ManagedBy
    - Project
```

**Use Case**: Log storage with automatic cost optimization

### Public S3 Bucket for Static Website

Public bucket for hosting static website:

```yaml
# s3-static-website.yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: my-website-static-content
      type: bucket
      versioning: false
      encryption: false
      public_access: true
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: static-website
        Domain: mywebsite.com
policies:
  require_encryption: false
  no_public_buckets: false
```

**Use Case**: Static website hosting
**Note**: Only enable public access when specifically needed

### Development Environment Bucket

Development bucket with relaxed settings:

```yaml
# s3-dev-bucket.yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: myapp-dev-storage-2024
      type: bucket
      versioning: false
      encryption: true
      public_access: false
      tags:
        Environment: development
        ManagedBy: Genesys
        Purpose: development-testing
        Developer: john-doe
        Temporary: true
      lifecycle:
        delete_after_days: 90
policies:
  require_encryption: true
  no_public_buckets: true
  require_tags:
    - Environment
    - Developer
```

**Use Case**: Short-term development and testing

### Multi-Environment Buckets

Production bucket with strict security:

```yaml
# s3-production-bucket.yaml
provider: aws
region: us-west-2
resources:
  storage:
    - name: myapp-prod-data-secure
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: application-data
        Compliance: required
        BackupSchedule: daily
        DataClassification: sensitive
      lifecycle:
        archive_after_days: 90
        delete_after_days: 2555  # 7 years
policies:
  require_encryption: true
  no_public_buckets: true
  require_tags:
    - Environment
    - Compliance
    - DataClassification
```

**Use Case**: Production data with compliance requirements

## Multi-Region Configurations

### Cross-Region Bucket Setup

Buckets in different regions for global application:

**Primary Region (US East)**:
```yaml
# s3-primary-us-east.yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: myapp-primary-us-east
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: primary-data
        Region: us-east-1
        ReplicaRegion: eu-west-1
policies:
  require_encryption: true
  no_public_buckets: true
```

**Secondary Region (EU West)**:
```yaml
# s3-secondary-eu-west.yaml
provider: aws
region: eu-west-1
resources:
  storage:
    - name: myapp-secondary-eu-west
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: replica-data
        Region: eu-west-1
        PrimaryRegion: us-east-1
policies:
  require_encryption: true
  no_public_buckets: true
```

## Command Examples

### Complete S3 Workflow

```bash
# 1. Configure AWS
genesys config setup

# 2. Create configuration interactively
genesys interact
# Select: aws -> S3 Storage Bucket -> configure settings

# 3. Review generated configuration
cat s3-mybucket-*.yaml

# 4. Preview deployment
genesys execute s3-mybucket-*.yaml --dry-run

# 5. Deploy bucket
genesys execute s3-mybucket-*.yaml

# 6. List resources to verify
genesys list resources --service storage

# 7. Clean up when done
genesys execute deletion s3-mybucket-*.yaml --dry-run
genesys execute deletion s3-mybucket-*.yaml
```

### Multi-Provider Setup

```bash
# Configure multiple providers
genesys config setup    # Configure AWS
genesys config setup    # Configure GCP  
genesys config setup    # Configure Azure

# List all providers
genesys config list

# Set default provider
genesys config default aws

# Create resources on different providers
genesys interact        # AWS S3 bucket
genesys interact        # GCP storage bucket
genesys interact        # Azure blob storage

# Deploy to specific providers
genesys execute aws-storage.yaml
genesys execute gcp-storage.yaml  
genesys execute azure-storage.yaml
```

### Development vs Production

```bash
# Development workflow
genesys config setup                    # Configure dev AWS account
genesys interact                        # Create dev resources
genesys execute dev-storage.yaml        # Deploy to development

# Production workflow  
genesys config setup                    # Configure prod AWS account
cp dev-storage.yaml prod-storage.yaml   # Copy and modify config
# Edit prod-storage.yaml for production settings
genesys execute prod-storage.yaml --dry-run  # Preview changes
genesys execute prod-storage.yaml            # Deploy to production
```

### Batch Operations

```bash
# Create multiple configurations
genesys interact  # Create storage bucket config
genesys interact  # Create compute instance config
genesys interact  # Create database config

# Deploy all resources
for config in *.yaml; do
    echo "Deploying $config..."
    genesys execute "$config" --dry-run
    genesys execute "$config"
done

# List all deployed resources
genesys list resources

# Clean up all resources
for config in *.yaml; do
    echo "Deleting resources from $config..."
    genesys execute deletion "$config"
done
```

## Use Case Scenarios

### Scenario 1: Web Application Storage

**Requirements**:
- S3 bucket for user uploads
- Encryption required
- Automatic cleanup of old files
- Development and production environments

**Development Configuration**:
```yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: webapp-uploads-dev-2024
      type: bucket
      versioning: false
      encryption: true
      public_access: false
      tags:
        Environment: development
        ManagedBy: Genesys
        Purpose: user-uploads
        Application: web-app
      lifecycle:
        delete_after_days: 30
policies:
  require_encryption: true
  no_public_buckets: true
```

**Production Configuration**:
```yaml
provider: aws
region: us-west-2
resources:
  storage:
    - name: webapp-uploads-prod-2024
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: user-uploads
        Application: web-app
        Compliance: gdpr
      lifecycle:
        archive_after_days: 90
        delete_after_days: 2555
policies:
  require_encryption: true
  no_public_buckets: true
```

### Scenario 2: Data Analytics Pipeline

**Requirements**:
- Raw data ingestion bucket
- Processed data bucket  
- Archive bucket for long-term storage
- Cross-region replication

**Raw Data Bucket**:
```yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: analytics-raw-data-2024
      type: bucket
      versioning: false
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: raw-data-ingestion
        Pipeline: analytics
        Stage: ingestion
      lifecycle:
        archive_after_days: 7
        delete_after_days: 30
policies:
  require_encryption: true
  no_public_buckets: true
```

**Processed Data Bucket**:
```yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: analytics-processed-data-2024
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: processed-data
        Pipeline: analytics
        Stage: processing
      lifecycle:
        archive_after_days: 30
        delete_after_days: 365
policies:
  require_encryption: true
  no_public_buckets: true
```

### Scenario 3: Backup and Disaster Recovery

**Primary Backup Bucket**:
```yaml
provider: aws
region: us-east-1
resources:
  storage:
    - name: company-backups-primary-2024
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: primary-backups
        DataRetention: 7-years
        Compliance: sox
      lifecycle:
        archive_after_days: 90
        delete_after_days: 2555
policies:
  require_encryption: true
  no_public_buckets: true
```

**Disaster Recovery Bucket**:
```yaml
provider: aws
region: us-west-2
resources:
  storage:
    - name: company-backups-dr-2024
      type: bucket
      versioning: true
      encryption: true
      public_access: false
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: disaster-recovery
        PrimaryRegion: us-east-1
        DataRetention: 7-years
policies:
  require_encryption: true
  no_public_buckets: true
```

## Best Practices Examples

### Naming Conventions

```yaml
# Good naming examples
name: mycompany-prod-logs-2024        # Company-environment-purpose-year
name: webapp-dev-uploads-us-east      # Application-environment-purpose-region
name: analytics-raw-data-pipeline     # System-type-purpose-component

# Avoid these patterns
name: bucket1                         # Too generic
name: My-Bucket-Name                  # Mixed case (invalid)
name: bucket_with_underscores         # Underscores not recommended
```

### Tagging Standards

```yaml
# Comprehensive tagging
tags:
  # Required tags
  Environment: production
  ManagedBy: Genesys
  Purpose: application-data
  
  # Business tags
  Project: web-application
  CostCenter: engineering
  Owner: platform-team
  
  # Technical tags  
  Region: us-east-1
  DataClassification: internal
  BackupSchedule: daily
  
  # Operational tags
  CreatedDate: "2024-01-15"
  Version: "1.0"
  Monitoring: enabled
```

### Security-First Configuration

```yaml
# Security-focused S3 bucket
provider: aws
region: us-west-2
resources:
  storage:
    - name: secure-app-data-prod-2024
      type: bucket
      versioning: true          # Enable versioning for data protection
      encryption: true          # Always encrypt at rest
      public_access: false      # Never allow public access by default
      tags:
        Environment: production
        ManagedBy: Genesys
        Purpose: sensitive-data
        DataClassification: confidential
        Compliance: hipaa
      lifecycle:
        archive_after_days: 90   # Cost optimization
        delete_after_days: 2555  # 7 years retention
policies:
  require_encryption: true      # Enforce encryption
  no_public_buckets: true       # Block public access
  require_tags:                 # Enforce required tags
    - Environment
    - DataClassification
    - Compliance
```