# Interactive Workflow Guide

Comprehensive guide to using Genesys interactive mode for cloud resource creation.

## Overview

Genesys interactive mode (`genesys interact`) provides a guided workflow for creating cloud resources through step-by-step prompts. This eliminates the need to write configuration files manually and ensures all required settings are properly configured.

## Starting Interactive Mode

```bash
genesys interact
```

## Workflow Steps

### Step 1: Provider Selection

Choose your cloud provider from available options:

- **aws** - Amazon Web Services
- **gcp** - Google Cloud Platform  
- **azure** - Microsoft Azure
- **tencent** - Tencent Cloud

The selected provider must be pre-configured with credentials. If not configured, you'll see an error message prompting you to run `genesys config setup`.

### Step 2: Resource Type Selection

Choose the type of resource to create:

- **S3 Storage Bucket** - Object storage bucket (AWS S3)
- **Compute Instance** - Virtual machine/server instance
- **Database** - Managed database service
- **Function** - Serverless function
- **Network** - Virtual network infrastructure

Currently, S3 Storage Bucket is fully implemented. Other resource types show placeholder messages.

### Step 3: Resource Configuration

Based on your selections, you'll be guided through resource-specific configuration prompts.

## S3 Storage Bucket Configuration

When selecting S3 Storage Bucket, you'll go through the following configuration steps:

### Basic Settings

**Bucket Name**:
- Must be globally unique across all AWS accounts
- 3-63 characters long
- DNS-compliant (lowercase, numbers, hyphens only)
- No spaces or special characters
- Example: `my-app-data-bucket-2024`

**Region Selection**:
- Choose from common AWS regions
- Each region has description (e.g., "US East (N. Virginia)")
- Default: `us-east-1`
- Affects latency and compliance requirements

### Security Configuration

**Enable Versioning**:
- Keeps multiple versions of objects in bucket
- Protects against accidental deletion or modification
- Default: `yes` (recommended)
- Adds small storage cost but provides data protection

**Enable Encryption**:
- Server-side encryption using AES256
- Encrypts all objects at rest
- Default: `yes` (recommended)
- No additional cost, provides data protection

**Allow Public Access**:
- Controls whether bucket can be accessed publicly
- Default: `no` (recommended for security)
- Only enable if hosting static websites or public assets
- Security warning shown when selecting yes

### Tagging

**Default Tags** (automatically added):
- `Environment: development` - Deployment environment
- `ManagedBy: Genesys` - Tool that manages this resource
- `Purpose: demo` - General purpose description

**Custom Tags** (optional):
- Add additional key-value pairs
- Useful for:
  - Cost allocation and billing
  - Access control policies
  - Automation and monitoring
  - Organizational requirements

Example custom tags:
```
Project: web-application
Owner: engineering-team
CostCenter: 1001
```

### Lifecycle Policies (Optional)

Configure automatic object management:

**Archive Objects**:
- Move objects to Glacier (cheaper storage) after specified days
- Common setting: 90 days
- Reduces storage costs for infrequently accessed data
- Objects remain accessible but with retrieval delay

**Delete Objects**:
- Permanently delete objects after specified days  
- Common setting: 365 days (1 year)
- Helps with compliance and cost management
- Cannot be undone after deletion

### Configuration File Generation

After completing all prompts, the interactive workflow:

1. **Generates YAML configuration** with all specified settings
2. **Saves to file** with format: `s3-<bucket-name>-<timestamp>.yaml`
3. **Shows next steps** for deployment and management

Example generated filename: `s3-my-app-data-bucket-1703876543.yaml`

## Next Steps After Interactive Configuration

### Review Configuration

```bash
cat s3-my-app-data-bucket-1703876543.yaml
```

Verify all settings are correct before deployment.

### Preview Deployment (Dry Run)

```bash
genesys execute s3-my-app-data-bucket-1703876543.yaml --dry-run
```

Shows exactly what will be created without making changes.

### Deploy Resources

```bash
genesys execute s3-my-app-data-bucket-1703876543.yaml
```

Creates the actual S3 bucket with all configured settings.

### Manage Resources

```bash
# List all resources
genesys list resources

# List only storage resources  
genesys list --service storage
```

### Clean Up Resources

```bash
# Preview deletion
genesys execute deletion s3-my-app-data-bucket-1703876543.yaml --dry-run

# Delete bucket
genesys execute deletion s3-my-app-data-bucket-1703876543.yaml
```

## Prerequisites

### Provider Configuration

Before using interactive mode, ensure your chosen cloud provider is configured:

```bash
genesys config setup
```

This guides you through:
- Provider selection
- Credential configuration (local or manual)
- Region and other settings
- Credential validation

### Verify Configuration

Check your provider is properly configured:

```bash
# List all configured providers
genesys config list

# Show specific provider details
genesys config show aws
```

## Best Practices

### Naming Conventions

- Use consistent naming patterns across resources
- Include environment, project, or purpose in names
- Follow cloud provider naming requirements
- Examples:
  - `myapp-prod-data-bucket`
  - `project-dev-storage-2024`
  - `company-logging-bucket`

### Security Settings

- **Always enable encryption** unless there's a specific reason not to
- **Disable public access** unless hosting public content
- **Enable versioning** for important data
- **Use meaningful tags** for access control and billing

### Configuration Management

- **Keep configuration files** for documentation and deletion
- **Use descriptive file names** that identify the resource
- **Store configurations in version control** for team sharing
- **Review configurations** before deployment

### Cost Management

- **Use lifecycle policies** to manage storage costs
- **Choose appropriate regions** considering latency vs cost
- **Monitor resource usage** through cloud provider billing
- **Clean up unused resources** promptly

## Error Handling

### Common Scenarios

**Provider Not Configured**:
```
Provider 'aws' not configured. Run 'genesys config setup' first.
```
Solution: Configure the provider with credentials.

**Invalid Input**:
```
Bucket name must be at least 3 characters
```
Solution: Follow the validation requirements shown.

**Credential Issues**:
```
AWS credentials validation failed
```
Solution: Check credentials with `genesys config show aws` and reconfigure if needed.

### Getting Help

- Use `--help` flag with any command for detailed usage
- Check provider configuration: `genesys config list`
- Validate credentials: `genesys config show <provider>`
- Use dry-run mode to preview changes safely

## Advanced Usage

### Multiple Providers

Configure multiple cloud providers and switch between them:

```bash
# Configure multiple providers
genesys config setup  # Configure AWS
genesys config setup  # Configure GCP  
genesys config setup  # Configure Azure

# List all providers
genesys config list

# Set default provider
genesys config default aws
```

### Batch Operations

Create multiple resources by running interactive mode multiple times:

```bash
# Create first resource
genesys interact  # Create S3 bucket

# Create second resource  
genesys interact  # Create compute instance

# Deploy all resources
genesys execute bucket-config.yaml
genesys execute instance-config.yaml
```

### Configuration Templates

Save interactive configurations as templates for future use:

```bash
# Create base configuration
genesys interact  # Save as template-s3-basic.yaml

# Copy and modify for new projects
cp template-s3-basic.yaml project-a-storage.yaml
# Edit project-a-storage.yaml as needed
genesys execute project-a-storage.yaml
```