# Getting Started with Genesys

Quick start guide to get up and running with Genesys cloud resource management.

## What is Genesys?

Genesys is a simplicity-first Infrastructure as a Service tool that focuses on outcomes rather than resources. It provides an interactive approach to cloud resource management with:

- **Interactive workflows** - Guided prompts for resource creation
- **Multi-cloud support** - AWS, GCP, Azure, Tencent Cloud
- **Configuration-driven** - YAML files for resource lifecycle management
- **Dry-run capability** - Preview changes before deployment
- **Direct API integration** - Fast performance without heavy SDKs

## Installation

### Prerequisites

- Go 1.21 or higher (for building from source)
- Cloud provider account and credentials
- Command line terminal

### From Source

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd genesys
   ```

2. Build the application:
   ```bash
   go build -o genesys ./cmd/genesys
   ```

3. Verify installation:
   ```bash
   ./genesys version
   ```

### Add to PATH (Optional)

For system-wide access:
```bash
# Copy to system PATH
sudo cp genesys /usr/local/bin/

# Or add current directory to PATH
export PATH=$PATH:$(pwd)
```

## Initial Setup

### Step 1: Configure Cloud Provider

Before creating resources, configure your cloud provider credentials:

```bash
genesys config setup
```

This interactive wizard will:
1. **Select Provider** - Choose from AWS, GCP, Azure, or Tencent
2. **Detect Credentials** - Check for existing local credentials
3. **Configure Authentication** - Set up credentials if needed
4. **Select Region** - Choose default region
5. **Validate** - Test credentials work correctly
6. **Set Default** - Optionally set as default provider

### Step 2: Verify Configuration

Check your provider is configured correctly:

```bash
# List all configured providers
genesys config list

# Show details for specific provider
genesys config show aws
```

Expected output:
```
Configured Cloud Providers:

  ✓ AWS *
     Region: us-east-1
     Auth: Local Credentials

* = Default provider
```

## Quick Start: Create Your First S3 Bucket

### Step 1: Start Interactive Mode

```bash
genesys interact
```

### Step 2: Follow the Prompts

1. **Select provider**: Choose `aws`
2. **Select resource**: Choose `S3 Storage Bucket`
3. **Configure bucket**:
   - Name: `my-first-bucket-12345` (must be globally unique)
   - Region: `us-east-1`
   - Versioning: `yes`
   - Encryption: `yes`
   - Public access: `no`
   - Tags: Accept defaults or add custom tags
   - Lifecycle: Skip for now

### Step 3: Review Generated Configuration

The wizard saves a configuration file like `s3-my-first-bucket-1703876543.yaml`:

```bash
cat s3-my-first-bucket-*.yaml
```

### Step 4: Preview Deployment (Dry Run)

```bash
genesys execute s3-my-first-bucket-*.yaml --dry-run
```

This shows what would be created without making actual changes.

### Step 5: Deploy the Bucket

```bash
genesys execute s3-my-first-bucket-*.yaml
```

Success output:
```
Deploying S3 bucket from: s3-my-first-bucket-1703876543.yaml
======================================

✓ S3 bucket 'my-first-bucket-12345' created successfully!
Region: us-east-1

Next steps:
  • View bucket: aws s3 ls s3://my-first-bucket-12345
  • Delete bucket: genesys execute deletion s3-my-first-bucket-1703876543.yaml
```

### Step 6: Verify Resource Creation

```bash
# List all your resources
genesys list resources

# List only storage resources
genesys list resources --service storage
```

### Step 7: Clean Up (When Done)

```bash
# Preview deletion
genesys execute deletion s3-my-first-bucket-*.yaml --dry-run

# Actually delete the bucket
genesys execute deletion s3-my-first-bucket-*.yaml
```

## Next Steps

### Learn More Commands

- **[Commands Reference](commands.md)** - Complete command documentation
- **[Interactive Workflow](interactive-workflow.md)** - Detailed interactive guide
- **[Configuration](configuration.md)** - Provider configuration details

### Explore More Resources

Once comfortable with S3 buckets, explore other resource types:
- Compute instances (VMs)
- Databases (managed SQL/NoSQL)
- Functions (serverless compute)
- Networks (VPCs, subnets)

### Advanced Usage

- **Multiple Providers** - Configure AWS, GCP, Azure for multi-cloud
- **Configuration Templates** - Reuse configurations across projects
- **Batch Operations** - Deploy multiple resources together
- **Team Workflows** - Share configurations via version control

## Common Workflows

### Development Environment Setup

1. Configure development provider:
   ```bash
   genesys config setup  # Choose AWS, set region to us-east-1
   ```

2. Create development resources:
   ```bash
   genesys interact  # Create S3 bucket for app data
   genesys interact  # Create compute instance for development
   ```

3. Deploy when ready:
   ```bash
   genesys execute dev-storage.yaml
   genesys execute dev-server.yaml
   ```

### Production Deployment

1. Configure production provider:
   ```bash
   genesys config setup  # Same provider, production region
   genesys config default aws
   ```

2. Copy and modify development configurations:
   ```bash
   cp dev-storage.yaml prod-storage.yaml
   # Edit prod-storage.yaml for production settings
   ```

3. Deploy with dry-run first:
   ```bash
   genesys execute prod-storage.yaml --dry-run
   genesys execute prod-storage.yaml
   ```

### Multi-Cloud Setup

1. Configure multiple providers:
   ```bash
   genesys config setup  # Configure AWS
   genesys config setup  # Configure GCP
   genesys config setup  # Configure Azure
   ```

2. Create resources on different providers:
   ```bash
   genesys interact  # Select AWS for primary storage
   genesys interact  # Select GCP for analytics
   genesys interact  # Select Azure for backups
   ```

## Tips for Success

### Before You Start

- **Have credentials ready** - AWS keys, GCP service account, etc.
- **Know your requirements** - Region, security needs, naming conventions
- **Plan resource names** - Use consistent, descriptive naming
- **Understand costs** - Know pricing for resources you create

### During Development

- **Always dry-run first** - Preview changes before deployment
- **Use meaningful names** - Include environment, purpose, date
- **Add descriptive tags** - Help with organization and billing
- **Keep configurations** - Save YAML files for resource management

### Production Deployment

- **Test in development first** - Validate configurations work
- **Review security settings** - Ensure appropriate access controls
- **Monitor resources** - Check cloud provider dashboards
- **Plan for cleanup** - Know how to delete resources when done

## Getting Help

### Documentation

- **Commands**: `genesys <command> --help`
- **Configuration**: `genesys config list`, `genesys config show <provider>`
- **Status**: `genesys list resources`

### Troubleshooting

Common issues and solutions:

**Provider not configured**:
```bash
genesys config setup
```

**Invalid credentials**:
```bash
genesys config show aws  # Check configuration
genesys config setup     # Reconfigure if needed
```

**Resource already exists**:
- Use different name (S3 buckets must be globally unique)
- Check existing resources: `genesys list resources`

**Permission denied**:
- Verify IAM permissions for your cloud provider
- Check account limits and quotas

### Support

- Check command help: `genesys --help`
- Review documentation in `/docs` folder
- Validate configuration: `genesys config list`
- Test with dry-run: `genesys execute config.yaml --dry-run`

## What's Next?

Now that you have Genesys set up and have created your first resource:

1. **Explore other resource types** - Try compute instances or databases
2. **Set up additional providers** - Configure GCP or Azure for multi-cloud
3. **Create configuration templates** - Build reusable configurations for your team
4. **Integrate with CI/CD** - Automate deployments with your build pipeline
5. **Learn advanced features** - Batch operations, complex configurations

Continue with the [Interactive Workflow Guide](interactive-workflow.md) for detailed usage patterns.