# Getting Started with Genesys

Complete beginner's guide to cloud infrastructure management with Genesys - from installation to deploying your first resources.

## What is Genesys?

Genesys transforms cloud infrastructure deployment from complex to simple. Instead of learning hundreds of cloud service parameters, you answer guided questions and get production-ready resources with secure defaults.

### Why Choose Genesys?

**Traditional Infrastructure Tools:**
- Require deep cloud provider knowledge
- Complex configuration files with hundreds of options  
- Easy to misconfigure security settings
- No cost visibility until after deployment
- Provider-specific syntax and commands

**Genesys Approach:**
- **Interactive Guided Setup** - Answer simple questions, get secure infrastructure
- **Built-in Best Practices** - Security, encryption, and cost optimization by default
- **Real-Time Cost Estimates** - See costs before you deploy anything
- **Human-Readable Plans** - Understand exactly what will be created
- **Multi-Cloud Ready** - Same commands work across AWS, GCP, Azure
- **Version Control Friendly** - Generated configuration files work with Git

## Installation

### Prerequisites

**Required:**
- **Go 1.21+** - For building from source ([Download Go](https://golang.org/dl/))
- **Cloud Provider Account** - AWS (Free Tier available), GCP, or Azure account
- **Terminal/Command Line** - Command line interface (Terminal on Mac/Linux, PowerShell/CMD on Windows)

**Recommended:**
- **Git** - For version control of generated configuration files
- **AWS CLI** - For easier credential setup (if using AWS)

### Step 1: Install Genesys

#### Option A: Local Installation (Recommended)

```bash
# Clone, build, and install to user directory
git clone https://github.com/javanhut/genesys.git
cd genesys
make install-local

# Verify installation
genesys version
```

#### Option B: System-Wide Installation

```bash
# Clone, build, and install system-wide
git clone https://github.com/javanhut/genesys.git
cd genesys
sudo make install

# Verify installation
genesys version
```

#### Option C: Development Build

```bash
# Clone and build for development
git clone https://github.com/javanhut/genesys.git
cd genesys
make build

# Run from current directory
./genesys version
```

For installation help, run `make help`.

# Now you can use 'genesys' from anywhere
genesys version
```

### Step 2: Verify Installation

```bash
# Check version (should show current version)
genesys version

# List available commands
genesys --help

# Check that interactive mode is available
genesys interact --help
```

Expected output:
```
Genesys v1.0.0
Interactive cloud infrastructure management tool

Available Commands:
  config      Manage provider configurations
  execute     Deploy or delete resources
  interact    Start interactive resource creation
  list        List resources (alias: discover)
  version     Show version information
```

## Cloud Provider Setup

Before creating any resources, you need to configure access to your cloud provider. Genesys currently supports AWS with full interactive workflows.

### Step 1: Prepare Your AWS Credentials  

You have several options for AWS authentication:

#### Option A: AWS CLI (Recommended for Beginners)
```bash
# Install AWS CLI if not already installed
# See: https://aws.amazon.com/cli/

# Configure your credentials
aws configure
# Enter: Access Key ID, Secret Access Key, Default region, Default output format
```

#### Option B: Environment Variables
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

#### Option C: IAM Role (For EC2 instances)
If running on an EC2 instance, Genesys can use IAM roles automatically.

### Step 2: Configure Genesys

```bash
# Start the interactive configuration wizard
genesys config setup
```

**Follow the Interactive Prompts:**

1. **Select Cloud Provider**: Choose `aws`
2. **Select Region**: Choose your preferred region (e.g., `us-east-1`)
3. **Credential Detection**: Genesys automatically detects your existing AWS credentials
4. **Validation**: Tests that your credentials work with a simple API call
5. **Set as Default**: Choose whether to make this your default provider

### Step 3: Verify Configuration

```bash
# List all configured providers
genesys config list

# Show detailed AWS configuration
genesys config show aws
```

**Expected Output:**
```
Configured Cloud Providers:

  ✓ AWS *
     Region: us-east-1
     Auth Method: Environment Variables
     Status: Connected
     Last Verified: 2024-01-15 10:30:00

* = Default provider

AWS Configuration Details:
  Region: us-east-1
  Access Method: AWS CLI Profile
  Permissions: Validated
  Services Available: S3, EC2
```

### Troubleshooting Configuration

**Issue: "AWS credentials not found"**
```bash
# Check if AWS CLI is configured
aws sts get-caller-identity

# If not working, reconfigure
aws configure
```

**Issue: "Permission denied"**  
- Verify your AWS user has the necessary IAM permissions
- For S3: `s3:CreateBucket`, `s3:PutObject`, `s3:GetObject`
- For EC2: `ec2:RunInstances`, `ec2:DescribeInstances`, `ec2:TerminateInstances`

**Issue: "Invalid region"**
```bash
# List available regions
aws ec2 describe-regions --output table

# Reconfigure with valid region
genesys config setup
```

## Complete Tutorials

Now that you have Genesys configured, let's walk through creating your first resources. We'll start with storage, then move to compute instances.

---

## Tutorial 1: Create Your First S3 Bucket

S3 buckets provide secure, scalable object storage. Perfect for storing application data, backups, or static website content.

### Step 1: Start Interactive Mode

```bash
genesys interact
```

### Step 2: Configure Your Bucket

Follow these prompts step-by-step:

1. **Cloud Provider**: Select `aws` (use arrow keys, press Enter)
2. **Resource Type**: Select `S3 Storage Bucket` 
3. **Bucket Configuration**:
   - **Bucket Name**: `tutorial-bucket-yourname-123` 
     - Must be globally unique across ALL S3 buckets
     - Use lowercase, numbers, and hyphens only
     - Include your name or random numbers for uniqueness
   - **Versioning**: Choose `yes` (protects against accidental deletion)
   - **Encryption**: Choose `yes` (encrypts data at rest automatically) 
   - **Public Access**: Choose `no` (secure by default)
   - **Lifecycle Management**: Choose `no` for this tutorial
   - **Tags**: Add at least:
     - `Environment`: `tutorial`
     - `Purpose`: `learning`
     - `Owner`: `your-name`

### Step 3: Review Cost Estimate

Genesys will show you cost estimates:
```
Cost Estimate for S3 Bucket:
- Storage (first 50TB): $0.023/GB/month
- Requests: $0.0004 per 1,000 requests  
- Estimated monthly cost: $1-5 USD (depends on usage)

Configuration will be saved to: s3-tutorial-bucket-yourname-123-1703876543.toml
```

### Step 4: Generate Configuration

The interactive wizard creates a file like `s3-tutorial-bucket-yourname-123-1703876543.toml`:

```bash
# View the generated configuration
cat s3-tutorial-bucket-*.toml
```

**Example Generated Configuration:**
```toml
provider = "aws"
region = "us-east-1"

[[resources.storage]]
name = "tutorial-bucket-yourname-123"
type = "bucket"
versioning = true
encryption = true
public_access = false

[resources.storage.tags]
Environment = "tutorial"
Purpose = "learning"
Owner = "your-name"
ManagedBy = "Genesys"

[policies]
require_encryption = true
no_public_buckets = true
require_tags = ["Environment", "ManagedBy"]
```

### Step 5: Preview Deployment (Always Dry-Run First!)

```bash
genesys execute s3-tutorial-bucket-*.toml --dry-run
```

**Expected Dry-Run Output:**
```
Plan: Deploy S3 Bucket 'tutorial-bucket-yourname-123'
====================================================

What will happen:
> 1. Create S3 bucket 'tutorial-bucket-yourname-123'
     -> Secure object storage in us-east-1
> 2. Enable versioning on bucket
     -> Protects against accidental deletion/modification  
> 3. Enable AES-256 encryption
     -> Encrypts all data at rest automatically
> 4. Block all public access
     -> Prevents accidental public exposure

AWS API calls required:
- s3:CreateBucket
- s3:PutBucketVersioning
- s3:PutBucketEncryption
- s3:PutBucketPublicAccessBlock

Estimated monthly cost: $2-5 USD (depends on usage)
Time to complete: 15-30 seconds

No actual changes will be made during this dry-run.
```

### Step 6: Deploy the Bucket

If the dry-run looks correct, deploy it:

```bash
genesys execute s3-tutorial-bucket-*.toml
```

**Success Output:**
```
Deploying S3 bucket from: s3-tutorial-bucket-yourname-123-1703876543.toml
========================================================================

✓ Creating S3 bucket 'tutorial-bucket-yourname-123'... Done!
✓ Enabling versioning... Done!
✓ Configuring encryption... Done!  
✓ Blocking public access... Done!
✓ Adding tags... Done!

S3 bucket 'tutorial-bucket-yourname-123' created successfully!
Region: us-east-1
Encryption: AES-256 (AWS managed)
Versioning: Enabled
Public Access: Blocked

Next steps:
  • Upload files: aws s3 cp myfile.txt s3://tutorial-bucket-yourname-123/
  • List contents: aws s3 ls s3://tutorial-bucket-yourname-123/
  • Delete bucket: genesys execute deletion s3-tutorial-bucket-yourname-123-1703876543.toml
```

### Step 7: Verify Your Bucket

```bash
# List all your resources
genesys list resources

# List only storage resources  
genesys list resources --service storage

# Check bucket in AWS CLI (if installed)
aws s3 ls s3://tutorial-bucket-yourname-123/
```

### Step 8: Test Your Bucket

```bash
# Upload a test file (if you have AWS CLI)
echo "Hello from Genesys!" > test.txt
aws s3 cp test.txt s3://tutorial-bucket-yourname-123/

# Verify upload
aws s3 ls s3://tutorial-bucket-yourname-123/

# Clean up test file
rm test.txt
```

### Step 9: Clean Up (When Done Learning)

```bash
# IMPORTANT: Empty the bucket first if you uploaded any files
aws s3 rm s3://tutorial-bucket-yourname-123/ --recursive

# Preview deletion (always safe to run)
genesys execute deletion s3-tutorial-bucket-*.toml --dry-run

# Actually delete the bucket
genesys execute deletion s3-tutorial-bucket-*.toml
```

**Congratulations!** You've created, used, and deleted your first S3 bucket with Genesys.

---

## Tutorial 2: Create Your First EC2 Instance  

EC2 instances provide virtual machines in the cloud. Perfect for web servers, development environments, or any application that needs compute power.

### Step 1: Start Interactive Mode

```bash  
genesys interact
```

### Step 2: Configure Your Instance

Follow these prompts carefully:

1. **Cloud Provider**: Select `aws`
2. **Resource Type**: Select `Compute Instance`
3. **Instance Configuration**:
   - **Instance Name**: `tutorial-server-yourname`
     - Must be unique across your AWS account
     - Use descriptive names: `dev-server`, `web-app-staging`, etc.
   - **Instance Type**: Select `t3.micro` (Free Tier eligible - clearly marked)
   - **Operating System**: Select `ubuntu-lts` (Latest Ubuntu LTS automatically)
   - **Storage Configuration**:
     - **Size**: `8 GB` (Free Tier includes up to 30GB)
     - **Type**: `gp3` (Modern general purpose SSD)
     - **Encryption**: `yes` (secure by default)
   - **Network Settings**: Accept defaults (uses default VPC)
   - **Tags**: Add:
     - `Environment`: `tutorial`
     - `Purpose`: `learning`
     - `Owner`: `your-name`

### Step 3: Review Cost Estimate

Genesys shows real-time pricing:
```
Cost Estimate for EC2 Instance (t3.micro):
- Instance: $0.0104/hour = $7.49/month (FREE for first 750 hours/month)
- Storage: $0.08/GB/month × 8GB = $0.64/month (FREE for first 30GB)  
- Data Transfer: First 1GB/month free

Total Estimated Cost: FREE (under Free Tier limits)
Free Tier Status: ✓ This configuration is Free Tier eligible

Configuration will be saved to: ec2-tutorial-server-yourname-1703876543.toml
```

### Step 4: Review Generated Configuration

```bash
# View the generated configuration
cat ec2-tutorial-server-*.toml
```

**Example Generated Configuration:**
```toml
provider = "aws"
region = "us-east-1"

[[resources.compute]]
name = "tutorial-server-yourname"
type = "t3.micro"           # Free Tier eligible
image = "ubuntu-lts"        # Resolves to latest Ubuntu 22.04 LTS AMI
count = 1

[resources.compute.tags]
Environment = "tutorial"
Purpose = "learning"
Owner = "your-name"
ManagedBy = "Genesys"

[policies]
require_encryption = false
no_public_instances = true
require_tags = ["Environment", "ManagedBy", "Purpose"]
```

### Step 5: Preview Instance Creation (Critical Step!)

```bash
genesys execute ec2-tutorial-server-*.toml --dry-run
```

**Expected Dry-Run Output:**
```
Plan: Deploy EC2 Instance 'tutorial-server-yourname'
==================================================

What will happen:
> 1. Resolve Ubuntu LTS AMI for us-east-1
     -> Found: ami-0abcdef123456789 (Ubuntu 22.04.3 LTS)
> 2. Create t3.micro instance 'tutorial-server-yourname'  
     -> 1 vCPU, 1 GB RAM, Free Tier eligible
> 3. Create 8GB gp3 EBS volume with encryption
     -> General Purpose SSD, encrypted at rest
> 4. Configure default security group
     -> SSH access from your IP only
> 5. Add resource tags
     -> Environment, Purpose, Owner, ManagedBy

AWS API calls required:
- ec2:RunInstances
- ec2:DescribeImages (to resolve AMI)
- ec2:CreateTags

Estimated costs:
- Instance: FREE (Free Tier eligible)
- Storage: FREE (under 30GB Free Tier limit)
- Data Transfer: First 1GB free

Time to complete: 2-3 minutes (instance boot time)
No actual changes will be made during this dry-run.
```

### Step 6: Deploy the Instance

If everything looks correct:

```bash
genesys execute ec2-tutorial-server-*.toml
```

**Success Output:**
```
Deploying EC2 instance from: ec2-tutorial-server-yourname-1703876543.toml
========================================================================

✓ Resolving Ubuntu LTS AMI... Found ami-0abcdef123456789
✓ Creating t3.micro instance 'tutorial-server-yourname'... Done!
✓ Waiting for instance to reach 'running' state... Done!
✓ Configuring EBS volume encryption... Done!
✓ Adding resource tags... Done!

EC2 instance 'tutorial-server-yourname' created successfully!
Instance ID: i-0123456789abcdef0
Instance Type: t3.micro (Free Tier)  
Region: us-east-1
Private IP: 172.31.45.67
AMI: ami-0abcdef123456789 (Ubuntu 22.04 LTS)
Status: running

Next steps:
  • Connect: aws ec2 describe-instances --instance-ids i-0123456789abcdef0
  • SSH: You'll need to configure SSH keys separately
  • Monitor: Check AWS EC2 console for status updates
  • Delete: genesys execute deletion ec2-tutorial-server-yourname-1703876543.toml
```

### Step 7: Verify Your Instance

```bash
# List all your resources
genesys list resources

# List only compute instances
genesys list resources --service compute

# Check instance in AWS CLI (if installed)
aws ec2 describe-instances --filters "Name=tag:Name,Values=tutorial-server-yourname" --output table
```

### Step 8: Understanding Your Instance

Your instance is now running! Here's what you have:

- **Virtual Machine**: 1 vCPU, 1GB RAM (t3.micro)
- **Operating System**: Ubuntu 22.04 LTS (latest)
- **Storage**: 8GB encrypted SSD
- **Network**: Private IP in default VPC
- **Security**: No public access (secure by default)
- **Cost**: Free under AWS Free Tier

**Important Notes:**
- Instance is running and incurring time (free for first 750 hours/month)
- No SSH key configured - you'd need to add this separately for access
- Instance has private IP only - not directly accessible from internet
- All data encrypted and tagged for easy management

### Step 9: Clean Up Your Instance

**IMPORTANT**: Always clean up instances when done to avoid unexpected charges.

```bash
# Preview deletion (safe - shows what will be deleted)
genesys execute deletion ec2-tutorial-server-*.toml --dry-run

# Actually terminate the instance
genesys execute deletion ec2-tutorial-server-*.toml
```

**Deletion Output:**
```
Deleting EC2 instance from: ec2-tutorial-server-yourname-1703876543.toml
======================================================================

✓ Terminating instance 'tutorial-server-yourname' (i-0123456789abcdef0)... Done!
✓ Waiting for instance termination... Done!
✓ EBS volume will be deleted automatically... Done!

EC2 instance 'tutorial-server-yourname' terminated successfully!
All associated resources (EBS volumes) have been cleaned up.
No ongoing charges for this instance.
```

**Congratulations!** You've successfully created and managed your first EC2 instance with Genesys.

### Step 10: Verify Cleanup

```bash
# Confirm instance is terminated
genesys list resources --service compute

# Should show no compute resources, or terminated status
aws ec2 describe-instances --filters "Name=tag:Name,Values=tutorial-server-yourname" --output table
```

---

## What You've Learned

After completing both tutorials, you now know how to:

* **Install and configure Genesys** with your AWS credentials  
* **Create secure S3 buckets** with encryption and versioning  
* **Deploy EC2 instances** with automatic AMI resolution and cost awareness  
* **Use dry-run mode** to preview all changes safely  
* **Manage resources** with generated configuration files  
* **Clean up resources** to avoid unexpected charges  
* **Understand cost implications** before deploying anything

## Next Steps and Advanced Usage

### 1. Explore More Configuration Options

```bash
# Try different instance types
genesys interact
# Select: Compute Instance
# Try: t3.small, t3.medium (see cost differences)

# Try different storage options  
genesys interact
# Select: S3 Storage Bucket
# Configure: Lifecycle policies, different encryption options
```

### 2. Multi-Environment Workflows

```bash
# Configure multiple environments
genesys config setup  # Development environment
genesys config setup  # Production environment

# Create environment-specific resources
genesys interact  # dev-app-storage  
genesys interact  # prod-app-storage
```

### 3. Team Workflows

**Share configurations with your team:**
```bash
# Save configuration files to Git
git add s3-*.toml ec2-*.toml
git commit -m "Add infrastructure configurations"

# Team members can deploy the same infrastructure
genesys execute your-config.toml --dry-run
genesys execute your-config.toml
```

**Establish naming conventions:**
```bash
# Good naming patterns:
# {project}-{environment}-{purpose}-{resource}
myapp-dev-storage-bucket
myapp-prod-web-server  
myapp-staging-db-primary
```

### 4. Cost Management Best Practices

- **Always dry-run first** - See costs before deployment
- **Use Free Tier resources** - Look for "Free Tier eligible" labels
- **Tag everything** - Use consistent tags for cost allocation
- **Clean up regularly** - Delete unused development resources
- **Monitor usage** - Check AWS billing dashboard regularly

### 5. Security Best Practices

Genesys implements security best practices by default:
- **Encryption enabled** - All storage and compute encrypted
- **Private by default** - No public access unless explicitly enabled
- **Resource tagging** - All resources tagged for management
- **Unique names** - Built-in validation prevents conflicts

### 6. Learn More Advanced Features

- **[Interactive Workflow Guide](interactive-workflow.md)** - Advanced interactive usage patterns
- **[Configuration Management](configuration.md)** - Provider setup and credential management
- **[Commands Reference](commands.md)** - Complete command documentation with examples
- **[S3 Workflow Guide](s3-workflow.md)** - Advanced S3 bucket configurations

### 7. Troubleshooting Resources

If you encounter issues:

```bash
# Check your configuration
genesys config list
genesys config show aws

# Verify resource status
genesys list resources --output json

# Test with dry-run
genesys execute your-config.toml --dry-run

# Get command help
genesys --help
genesys <command> --help
```

**Common Solutions:**
- **Credential issues**: Run `genesys config setup` to reconfigure
- **Resource conflicts**: Use unique names with your username or project
- **Permission errors**: Check your AWS IAM permissions
- **Cost concerns**: Always use dry-run to see estimates first

## Real-World Use Cases

### Development Environment
```bash
# Create complete dev environment
genesys interact  # S3 bucket for application data
genesys interact  # EC2 instance for development server
genesys interact  # Additional S3 bucket for backups (coming soon: RDS database)
```

### Static Website Hosting  
```bash
# Create S3 bucket configured for static website hosting
genesys interact  # S3 bucket with public access for static site
# Coming soon: CloudFront distribution, Route53 DNS
```

### Multi-Environment Application
```bash
# Development
genesys config setup  # us-east-1, dev credentials
genesys interact      # Create dev resources

# Staging  
genesys config setup  # us-west-1, staging credentials
genesys interact      # Create staging resources

# Production
genesys config setup  # us-west-2, prod credentials  
genesys interact      # Create production resources
```

## Community and Support

### Getting Help
- **Built-in Help**: `genesys --help` and `genesys <command> --help`
- **Dry-Run Testing**: Always safe to run `--dry-run` to understand what will happen
- **Configuration Validation**: `genesys config list` and `genesys config show <provider>`

### Best Practices Summary
1. **Always dry-run first** - Especially in production
2. **Use descriptive names** - Include environment, purpose, and owner
3. **Tag consistently** - Environment, Owner, Purpose, ManagedBy
4. **Version control configs** - Save TOML files in Git
5. **Clean up regularly** - Delete unused development resources
6. **Monitor costs** - Use AWS billing dashboard and Free Tier alerts
7. **Security first** - Use Genesys secure defaults, add custom security as needed

## What's Coming Next

Genesys is actively being developed. Coming soon:

- **Database Resources** - RDS instances with automated backups
- **Serverless Functions** - Lambda functions with event triggers  
- **Network Resources** - VPCs, subnets, security groups
- **Multi-Cloud Support** - GCP, Azure with same interface
- **Advanced Templates** - Pre-built configurations for common use cases
- **Team Features** - Resource sharing and collaboration tools

---

**You're now ready to use Genesys for real infrastructure projects!** 

Start with small, development resources to get comfortable, then gradually expand to more complex, production deployments. The interactive workflows and dry-run capabilities make it safe to experiment and learn.

**Next recommended reading**: [Interactive Workflow Guide](interactive-workflow.md) for advanced usage patterns and [Commands Reference](commands.md) for complete command documentation.

