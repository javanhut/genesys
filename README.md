# Genesys - Cloud Infrastructure Made Simple

Genesys is a simplicity-first Infrastructure as Code tool that focuses on outcomes rather than implementation details. Create, manage, and deploy cloud resources through interactive workflows without complex configuration files.

## Why Genesys?

**Traditional cloud tools are complex**:
- Hundreds of parameters to understand
- Provider-specific knowledge required
- Manual configuration file creation
- Easy to misconfigure security settings

**Genesys makes it simple**:
- Interactive guided workflows
- Secure defaults built-in
- Real-time cost estimation
- Human-readable deployment plans
- Multi-cloud support with unified interface

## Quick Start Tutorial

### Step 1: Installation

```bash
# Clone and build
git clone https://github.com/javanhut/genesys.git
cd genesys
go build -o genesys ./cmd/genesys

# Verify installation
./genesys version
```

### Step 2: Configure Your First Provider

```bash
# Start interactive setup
./genesys config setup
```

Follow the prompts to configure AWS (or your preferred provider):
1. Choose provider: **aws**
2. Region: **us-east-1** (or your preferred region)
3. Credentials: Uses your existing AWS credentials (AWS CLI, environment variables, or IAM role)

### Step 3: Create Your First Storage Bucket

```bash
# Start interactive mode
./genesys interact
```

Follow this workflow:
1. **Provider**: Select `aws`
2. **Resource Type**: Select `S3 Storage Bucket`
3. **Configuration**:
   - Bucket name: `my-tutorial-bucket-123` (must be globally unique)
   - Versioning: `yes` (recommended)
   - Encryption: `yes` (secure by default)
   - Public access: `no` (secure by default)
   - Add tags as desired

The interactive wizard will generate a configuration file like `s3-my-tutorial-bucket-1234567890.yaml`

### Step 4: Preview and Deploy

```bash
# Preview what will be created (safe - no changes made)
./genesys execute s3-my-tutorial-bucket-*.yaml --dry-run

# Deploy the bucket
./genesys execute s3-my-tutorial-bucket-*.yaml
```

Success! Your S3 bucket is now created with secure defaults.

### Step 5: Create Your First EC2 Instance

```bash
# Start interactive mode again
./genesys interact
```

Follow this workflow:
1. **Provider**: Select `aws`  
2. **Resource Type**: Select `Compute Instance`
3. **Configuration**:
   - Instance name: `my-dev-server` (must be unique)
   - Instance type: `t3.micro` (Free Tier eligible)
   - Operating system: `ubuntu-lts`
   - Storage: `8 GB` (default)
   - Add environment tags

The wizard will generate `ec2-my-dev-server-1234567890.toml` with cost estimates.

```bash
# Preview the instance creation
./genesys execute ec2-my-dev-server-*.toml --dry-run

# Deploy the instance
./genesys execute ec2-my-dev-server-*.toml
```

### Step 6: Deploy Your First Lambda Function

```bash
# Create a simple Python function directory
mkdir -p MyLambdaCode
cd MyLambdaCode

# Create a simple Lambda function
cat > lambda_function.py << 'EOF'
def lambda_handler(event, context):
    return {
        'statusCode': 200,
        'body': 'Hello from Genesys Lambda!'
    }
EOF

# Create requirements.txt (can be empty)
touch requirements.txt

cd ..

# Start interactive Lambda creation
./genesys interact
```

Follow this workflow:
1. **Provider**: Select `aws`
2. **Resource Type**: Select `Lambda Function`
3. **Configuration**:
   - Function name: `my-first-function`
   - Runtime: `python3.12`
   - Source path: `MyLambdaCode`
   - Function URL: `yes` (creates HTTPS endpoint)
   - Memory: `512 MB` (default)
   - Timeout: `30 seconds` (default)

The wizard will generate `lambda-my-first-function-*.toml` and automatically:
- Create IAM role with proper permissions
- Build your code and dependencies
- Deploy the Lambda function
- Set up the Function URL

```bash
# Preview the Lambda deployment
./genesys execute lambda-my-first-function-*.toml --dry-run

# Deploy the Lambda function
./genesys execute lambda-my-first-function-*.toml --apply
```

Success! Your Lambda is deployed with a public HTTPS URL you can test immediately.

### Step 7: Manage Your Resources

```bash
# List all resources you've created
./genesys list resources

# List only storage resources
./genesys list resources --service storage

# List only compute instances  
./genesys list resources --service compute
```

### Step 8: Clean Up (When Done)

```bash
# Delete the Lambda function (includes IAM role cleanup)
./genesys execute deletion lambda-my-first-function-*.toml

# Delete the EC2 instance
./genesys execute deletion ec2-my-dev-server-*.toml

# Delete the S3 bucket
./genesys execute deletion s3-my-tutorial-bucket-*.yaml
```

## What Can You Create?

### Currently Supported Resources

#### AWS Storage (S3)
- **S3 Buckets**: Secure object storage with versioning, encryption, and lifecycle policies
- **Interactive Creation**: Guided setup with cost estimation
- **Policy Enforcement**: Secure defaults prevent public access
- **Full Lifecycle**: Create, update, and delete with configuration files

#### AWS Compute (EC2)
- **EC2 Instances**: Virtual machines with automatic AMI resolution
- **Free Tier Support**: t3.micro, t3.small, c7i-flex.large, m7i-flex.large options clearly marked
- **Cost Estimation**: Real-time pricing with regional rates
- **Unique Names**: Built-in validation prevents duplicate instance names
- **Storage Options**: Configurable EBS volumes with encryption

#### AWS Serverless (Lambda)
- **Lambda Functions**: Serverless compute with automatic dependency management
- **Layer Support**: Automatic creation of dependency layers from requirements.txt
- **IAM Integration**: Automatic role creation with least-privilege permissions
- **Function URLs**: Direct HTTPS endpoints for Lambda functions
- **Full Lifecycle**: Deploy code, dependencies, and infrastructure; clean up everything with one command

### Coming Soon
- **Databases** - RDS instances with automated backups
- **Networks** - VPCs, subnets, security groups
- **Multi-Cloud** - GCP, Azure, Tencent Cloud support

## Key Features

### Interactive Workflows
- **Guided Prompts**: Step-by-step configuration with help text
- **Smart Defaults**: Secure, cost-effective settings pre-selected
- **Real-Time Validation**: Immediate feedback on configuration choices
- **Cost Awareness**: See estimated costs before deployment

### Security First
- **Secure Defaults**: Encryption enabled, public access blocked by default
- **Policy Enforcement**: Built-in policies prevent insecure configurations
- **Unique Validation**: Prevents resource name conflicts
- **Permission Checking**: Validates required cloud permissions before deployment

### Developer Experience
- **Dry-Run Everything**: Preview all changes before making them
- **Human-Readable Plans**: Understand exactly what will happen
- **Configuration Files**: Generated TOML/YAML for version control
- **Direct API Integration**: Fast performance without heavy SDKs

### Interactive Terminal UI (New!)
- **TUI Dashboard**: Browse resources in an interactive terminal interface
- **Keyboard Navigation**: Fast, mouse-free resource management
- **S3 Browser**: Navigate buckets and files like a file manager
- **Resource Lists**: View EC2, S3, Lambda resources in real-time
- **No AWS CLI Required**: Direct API integration for all operations

```bash
# Launch interactive TUI
genesys tui

# Browse S3 bucket in TUI
genesys manage s3 my-bucket --tui

# Monitor resources in TUI
genesys monitor --tui
```

See [TUI Documentation](docs/tui.md) for complete guide.

## Complete Command Reference

### Interactive Workflows
```bash
genesys interact                    # Start interactive resource creation wizard
```

### Configuration Management  
```bash
genesys config setup                # Configure cloud provider credentials
genesys config list                 # List configured providers
genesys config show aws             # Show provider configuration details
genesys config default aws          # Set default provider
```

### Resource Deployment
```bash
# Deploy resources
genesys execute config.yaml                    
genesys execute config.toml

# Deploy Lambda functions (includes --apply flag requirement)
genesys execute lambda-function.toml --apply

# Preview changes (safe - no actual deployment)
genesys execute config.yaml --dry-run          
genesys execute lambda-function.toml --dry-run

# Delete resources (includes complete cleanup)
genesys execute deletion config.yaml           
genesys execute deletion lambda-function.toml  # Removes function, layer, and IAM role
```

### Resource Discovery
```bash
genesys list resources              # List all your resources
genesys list resources --service storage      # Filter by service type
genesys list resources --output json          # JSON output format
```

### Resource Monitoring (NEW!)
```bash
# Monitor all resources
genesys monitor resources
genesys monitor resources --watch   # Live monitoring with refresh

# Monitor specific resources
genesys monitor ec2 i-1234567890abcdef0 --period 24h
genesys monitor s3 my-bucket
genesys monitor lambda my-function --logs
genesys monitor lambda my-function --tail  # Stream logs in real-time
```

### Resource Management (NEW!)
```bash
# S3 File Operations (like scp for S3)
genesys manage s3 my-bucket ls              # List objects
genesys manage s3 my-bucket ls /path/       # List with prefix
genesys manage s3 my-bucket get file.txt ./local.txt
genesys manage s3 my-bucket put ./local.txt remote.txt
genesys manage s3 my-bucket rm old-file.txt
genesys manage s3 my-bucket sync ./backups/ /backups/2024-11/

# EC2 Management
genesys manage ec2 i-1234567890abcdef0 describe

# Lambda Management
genesys manage lambda my-function logs
genesys manage lambda my-function invoke '{"key":"value"}'
```

### Resource Inspection (NEW!)
```bash
# Deep resource inspection
genesys inspect ec2 i-1234567890abcdef0
genesys inspect s3 my-bucket --analyze
genesys inspect lambda my-function
genesys inspect s3 my-bucket --output json
```

### Help and Information
```bash
genesys --help                      # Show all available commands
genesys version                     # Show version information
genesys <command> --help            # Get help for specific command
```

## Configuration Examples

### S3 Bucket Configuration (Auto-Generated)
```yaml
provider: aws
region: us-east-1

resources:
  storage:
    - name: my-app-data-bucket
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

### EC2 Instance Configuration (Auto-Generated)
```toml
provider = "aws"
region = "us-east-1"

[[resources.compute]]
name = "web-server-dev"
type = "t3.micro"           # Free Tier eligible
image = "ubuntu-lts"        # Resolves to latest Ubuntu LTS AMI
count = 1

[resources.compute.tags]
Environment = "development"
ManagedBy = "Genesys"
Purpose = "web-server"

[policies]
require_encryption = false
no_public_instances = true
require_tags = ["Environment", "ManagedBy", "Purpose"]
```

### Lambda Function Configuration (Auto-Generated)
```toml
[metadata]
  name = "my-api-function"
  runtime = "python3.12"
  handler = "lambda_function.lambda_handler"
  description = "Lambda function my-api-function created with Genesys"

[build]
  source_path = "/path/to/MyLambdaCode"
  build_method = "podman"
  layer_auto = true
  requirements_file = "requirements.txt"

[function]
  memory_mb = 512
  timeout_seconds = 30

[deployment]
  function_url = true
  cors_enabled = true
  auth_type = "AWS_IAM"
  architecture = "x86_64"

[layer]
  name = "my-api-function-deps"
  description = "Dependencies for my-api-function Lambda function (x86_64)"
  compatible_runtimes = ["python3.12"]

[iam]
  role_name = "genesys-lambda-my-api-function"
  required_policies = [
    "Basic CloudWatch Logs access",
    "Lambda full access"
  ]
  auto_manage = true
  auto_cleanup = true
```

## Real-World Examples

### Development Environment Setup

**Goal**: Create a development environment with storage and compute

```bash
# Step 1: Configure AWS
genesys config setup
# Choose: aws, us-east-1, use existing credentials

# Step 2: Create application data bucket
genesys interact
# Provider: aws
# Resource: S3 Storage Bucket
# Name: myapp-dev-data-bucket
# Enable versioning and encryption

# Step 3: Create development server
genesys interact  
# Provider: aws
# Resource: Compute Instance
# Name: myapp-dev-server
# Type: t3.micro (Free Tier)
# OS: ubuntu-lts

# Step 4: Deploy everything
genesys execute s3-myapp-dev-data-bucket-*.yaml --dry-run
genesys execute s3-myapp-dev-data-bucket-*.yaml

genesys execute ec2-myapp-dev-server-*.toml --dry-run  
genesys execute ec2-myapp-dev-server-*.toml

# Step 5: Verify deployment
genesys list resources
```

### Production Deployment Workflow

**Goal**: Deploy production resources with proper validation

```bash
# Step 1: Configure production region
genesys config setup
# Choose: aws, us-west-2 (production region)

# Step 2: Create production bucket with strict settings
genesys interact
# Provider: aws
# Resource: S3 Storage Bucket  
# Name: myapp-prod-storage
# Versioning: yes
# Encryption: yes
# Lifecycle: 30 days archive, 365 days delete

# Step 3: Always dry-run first in production
genesys execute s3-myapp-prod-storage-*.yaml --dry-run

# Step 4: Review the plan, then deploy
genesys execute s3-myapp-prod-storage-*.yaml

# Step 5: Monitor and verify
genesys list resources
aws s3 ls s3://myapp-prod-storage
```

### Serverless API Development

**Goal**: Deploy a Python API with Lambda functions and storage

```bash
# Step 1: Create your API code
mkdir -p MyAPI
cd MyAPI

# Create main API handler
cat > lambda_function.py << 'EOF'
import json
import boto3

def lambda_handler(event, context):
    # Simple API that returns user data
    path = event.get('rawPath', '/')
    method = event.get('requestContext', {}).get('http', {}).get('method', 'GET')
    
    if path == '/health' and method == 'GET':
        return {
            'statusCode': 200,
            'headers': {'Content-Type': 'application/json'},
            'body': json.dumps({'status': 'healthy', 'service': 'MyAPI'})
        }
    elif path == '/users' and method == 'GET':
        return {
            'statusCode': 200,
            'headers': {'Content-Type': 'application/json'},
            'body': json.dumps({
                'users': [
                    {'id': 1, 'name': 'Alice'},
                    {'id': 2, 'name': 'Bob'}
                ]
            })
        }
    else:
        return {
            'statusCode': 404,
            'headers': {'Content-Type': 'application/json'},
            'body': json.dumps({'error': 'Not Found'})
        }
EOF

# Create requirements for dependencies
cat > requirements.txt << 'EOF'
boto3>=1.26.0
EOF

cd ..

# Step 2: Deploy Lambda API
genesys interact
# Provider: aws
# Resource: Lambda Function
# Name: my-api
# Runtime: python3.12
# Source: MyAPI
# Function URL: yes
# Memory: 512 MB

# Step 3: Deploy with dry-run first
genesys execute lambda-my-api-*.toml --dry-run
genesys execute lambda-my-api-*.toml --apply

# Step 4: Test your API
curl https://your-function-url.lambda-url.us-east-1.on.aws/health
curl https://your-function-url.lambda-url.us-east-1.on.aws/users

# Step 5: Create storage for the API
genesys interact
# Provider: aws  
# Resource: S3 Storage Bucket
# Name: my-api-data-bucket
# Versioning: yes, Encryption: yes

genesys execute s3-my-api-data-bucket-*.yaml --apply
```

### Multi-Environment Management

**Goal**: Manage development, staging, and production environments

```bash
# Configure multiple regions/accounts
genesys config setup  # Development: us-east-1
genesys config setup  # Staging: us-west-1  
genesys config setup  # Production: us-west-2

# Create resources per environment
genesys interact  # Dev resources
genesys interact  # Staging resources  
genesys interact  # Production resources

# Deploy with appropriate validation
genesys execute dev-*.yaml
genesys execute staging-*.yaml --dry-run
genesys execute staging-*.yaml
genesys execute prod-*.yaml --dry-run
genesys execute prod-*.yaml
```

## Monitoring and Management (No AWS CLI Required!)

Genesys now includes powerful monitoring and management capabilities without requiring AWS CLI installation.

### Real-Time Resource Monitoring

Monitor your resources with CloudWatch metrics integration:

```bash
# Monitor all resources with live updates
genesys monitor resources --watch

# Output:
# === Resource Monitor === 10:35:42
# ✓ i-123: CPU 45.2%
# ✓ i-456: CPU 23.1%
# ✓ my-bucket: 45.6 GB, 1,234 objects
# ✓ my-function: 1,234 invocations/day
```

### S3 File Management

Upload, download, and sync files without AWS CLI:

```bash
# List objects
$ genesys manage s3 my-bucket ls
PREFIX    data/
PREFIX    logs/
FILE      config.json          1.2 KB    2024-11-20 10:30:00

# Download files
$ genesys manage s3 my-bucket get data/file.txt ./local-file.txt
Downloading: s3://my-bucket/data/file.txt → ./local-file.txt
✓ Download complete: 5.2 MB (2.3 MB/s)

# Upload files
$ genesys manage s3 my-bucket put ./report.pdf reports/2024-11.pdf
Uploading: ./report.pdf → s3://my-bucket/reports/2024-11.pdf
✓ Upload complete: 5.2 MB (1.8 MB/s)

# Sync entire directories
$ genesys manage s3 my-bucket sync ./backups/ /backups/2024-11-24/
Syncing ./backups/ ↔ s3://my-bucket/backups/2024-11-24/ (direction: upload)
Found 15 local files to sync
✓ Sync complete
```

### Lambda Log Streaming

View Lambda logs in real-time:

```bash
# View recent logs
$ genesys manage lambda my-function logs
[10:30:15] START RequestId: abc123-def456
[10:30:15] INFO: Processing request for user: user@example.com
[10:30:16] END RequestId: abc123-def456
[10:30:16] REPORT Duration: 145.23 ms  Memory: 128/512 MB

# Tail logs in real-time
$ genesys monitor lambda my-function --tail
Tailing logs for Lambda function: my-function
[10:30:15] START RequestId: abc123-def456
[10:30:15] INFO: Processing request...
```

### Deep Resource Inspection

Inspect resources without switching tools:

```bash
$ genesys inspect ec2 i-1234567890abcdef0
Instance ID:   i-1234567890abcdef0
Name:          web-server-prod
Type:          t3.medium
State:         running
Private IP:    10.0.1.15
Recent CPU Usage: 45.2%

$ genesys inspect s3 my-bucket --analyze
Bucket Analysis:
  Total Objects: 1,234
  Total Size:    45.6 GB
  Versioning:    Enabled
  Encryption:    AES256
```

## Advanced Usage

### Cost Management
- **Estimate Before Deploy**: All resources show cost estimates during creation
- **Free Tier Awareness**: Free Tier eligible options are clearly marked
- **Regional Pricing**: Cost estimates reflect your selected region
- **Resource Tagging**: Automatic tagging helps with cost allocation

### Security Best Practices  
- **Encryption by Default**: Storage and compute resources encrypted automatically
- **Private by Default**: Public access disabled unless explicitly enabled
- **Policy Validation**: Built-in policies prevent common security mistakes
- **Name Uniqueness**: Prevents resource conflicts and naming collisions

### Team Workflows
- **Version Control**: Save generated configuration files in Git
- **Consistent Naming**: Use descriptive, environment-specific names  
- **Tag Standards**: Consistent tagging across all resources
- **Review Process**: Always use dry-run for production deployments

## Troubleshooting

### Common Issues and Solutions

**"Provider not configured"**
```bash
genesys config setup
```

**"Invalid credentials"**  
```bash
genesys config show aws    # Check current configuration
genesys config setup       # Reconfigure credentials
```

**"Resource name already exists"**
- Choose a different, unique name
- Check existing resources: `genesys list resources`

**"Permission denied"**
- Verify IAM permissions in AWS console
- Check account limits and quotas
- Ensure credentials have required permissions

**"Dry-run shows different results than expected"**
- AMI IDs and availability zones are resolved dynamically
- Cost estimates may vary by region and current pricing
- Instance types may have regional availability differences

### Getting Help

```bash
genesys --help                     # General help
genesys interact --help            # Interactive mode help
genesys config --help              # Configuration help
genesys execute --help             # Deployment help
genesys list --help                # Resource listing help
```

### Debug Information
```bash
genesys config list                # Show all provider configurations
genesys config show aws            # Show AWS-specific configuration  
genesys list resources --output json  # Get detailed resource information
```

## Next Steps

### Learn More
- **[Getting Started Guide](docs/getting-started.md)** - Detailed walkthrough
- **[Interactive Workflows](docs/interactive-workflow.md)** - Advanced interactive usage
- **[Configuration Guide](docs/configuration.md)** - Provider setup and management
- **[Commands Reference](docs/commands.md)** - Complete command documentation

### Extend Your Usage
1. **Explore More Resources** - Try databases and serverless functions when available
2. **Multi-Cloud Setup** - Configure multiple cloud providers  
3. **Automation** - Integrate with CI/CD pipelines
4. **Team Standards** - Establish naming and tagging conventions
5. **Cost Optimization** - Use resource tagging for cost allocation

## Architecture Overview

### Core Principles
1. **Simplicity First** - Complex infrastructure should be simple to deploy
2. **Secure by Default** - Best practices built into every resource
3. **Cost Conscious** - Always show cost implications before deployment
4. **Provider Agnostic** - Same interface works across cloud providers
5. **Human Readable** - Plans and configurations anyone can understand

### Technical Design
- **Direct API Integration** - Fast, lightweight cloud provider communication
- **Interactive CLI** - Rich terminal experience with guided workflows  
- **Configuration Generation** - TOML/YAML files for version control and repeatability
- **Validation First** - Extensive validation before any cloud API calls
- **State Awareness** - Tracks resources locally to prevent conflicts

---

**Ready to get started?** Run `genesys config setup` to configure your first cloud provider, then `genesys interact` to create your first resource!