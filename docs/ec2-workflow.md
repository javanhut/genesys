# EC2 Instance Workflow Guide

Complete guide to creating, managing, and deploying AWS EC2 instances using Genesys interactive workflows.

## Overview

EC2 (Elastic Compute Cloud) instances provide scalable virtual machines in AWS. Genesys simplifies EC2 deployment with:

- **Interactive Configuration** - Guided setup with cost estimation
- **Automatic AMI Resolution** - Latest operating system images automatically selected
- **Free Tier Awareness** - Clear indication of Free Tier eligible options
- **Secure Defaults** - Encryption and private networking by default
- **Unique Name Validation** - Prevents duplicate instance names
- **Real-Time Cost Estimates** - See costs before deployment

## Supported Features

### Instance Types
- **t3.micro** - 1 vCPU, 1GB RAM (Free Tier eligible)
- **t3.small** - 1 vCPU, 2GB RAM (Free Tier eligible) 
- **t3.medium** - 2 vCPU, 4GB RAM
- **t3.large** - 2 vCPU, 8GB RAM
- **t3.xlarge** - 4 vCPU, 16GB RAM
- **c7i-flex.large** - Flexible compute (Free Tier eligible)
- **m7i-flex.large** - Flexible memory (Free Tier eligible)

### Operating Systems
- **ubuntu-lts** - Latest Ubuntu LTS (22.04)
- **amazon-linux** - Latest Amazon Linux 2023
- **centos** - Latest CentOS Stream
- **debian** - Latest Debian stable
- **windows** - Windows Server (latest)

### Storage Options
- **Size** - 8GB to 1TB+ EBS volumes
- **Type** - gp3 (General Purpose SSD), gp2, io1, io2
- **Encryption** - AES-256 encryption at rest
- **Delete on Termination** - Configurable cleanup behavior

## Basic Workflow

### Step 1: Start Interactive Mode

```bash
genesys interact
```

### Step 2: Select Provider and Resource

1. **Provider**: Choose `aws`
2. **Resource Type**: Choose `Compute Instance`

### Step 3: Configure Instance

Follow the interactive prompts:

#### Instance Details
- **Name**: Unique name (validated against existing instances)
- **Instance Type**: Choose based on performance needs and budget
- **Operating System**: Select from supported OS images

#### Storage Configuration  
- **Size**: Storage capacity in GB (8GB minimum)
- **Type**: SSD type (gp3 recommended for best performance/cost)
- **Encryption**: Enable for security (recommended)

#### Network & Security
- **VPC**: Uses default VPC (secure private network)
- **Public IP**: Disabled by default (secure)
- **Security Groups**: Default allows SSH from your IP only

#### Resource Tags
- **Environment**: development, staging, production
- **Purpose**: web-server, database, development
- **Owner**: Your name or team identifier
- **Additional Tags**: Custom key-value pairs

### Step 4: Review Cost Estimate

Genesys provides real-time cost estimates:
```bash
Cost Estimate for EC2 Instance (t3.micro):
- Instance: $0.0104/hour = $7.49/month (FREE for 750 hours/month)
- EBS Storage: $0.08/GB/month × 8GB = $0.64/month (FREE for 30GB/month)
- Data Transfer: First 1GB/month free

Total Monthly Cost: FREE (under AWS Free Tier limits)
Free Tier Eligible: ✓ Yes

Configuration saved to: ec2-my-instance-1703876543.toml
```

### Step 5: Preview Deployment

Preview changes (default behavior):
```bash
genesys execute ec2-my-instance-*.toml
```

### Step 6: Deploy Instance

Deploy with `--apply`:
```bash
genesys execute ec2-my-instance-*.toml --apply
```

### Step 7: Manage Instance

```bash
# List your instances
genesys list resources --service compute

# Preview instance update (if needed)
genesys execute ec2-my-instance-*.toml

# Terminate when done
genesys execute ec2-my-instance-*.toml --delete
```

## Advanced Configuration Examples

### Development Server

**Use Case**: Personal development environment

```toml
provider = "aws"
region = "us-east-1"

[[resources.compute]]
name = "dev-server-john"
type = "t3.micro"           # Free Tier
image = "ubuntu-lts"
count = 1

[resources.compute.tags]
Environment = "development"
Purpose = "personal-dev"
Owner = "john"
Project = "myapp"
ManagedBy = "Genesys"

[policies]
require_encryption = false
no_public_instances = true
require_tags = ["Environment", "Owner", "ManagedBy"]
```

**Estimated Cost**: FREE (Free Tier eligible)

### Web Application Server

**Use Case**: Small web application with moderate traffic

```toml
provider = "aws"
region = "us-east-1"

[[resources.compute]]
name = "webapp-prod-server"
type = "t3.medium"          # 2 vCPU, 4GB RAM
image = "ubuntu-lts"
count = 1

[resources.compute.tags]
Environment = "production"
Purpose = "web-server"
Application = "company-website"
Owner = "devops-team"
ManagedBy = "Genesys"

[policies]
require_encryption = true
no_public_instances = true
require_tags = ["Environment", "Application", "Owner", "ManagedBy"]
```

**Estimated Cost**: ~$30/month (t3.medium instance)

### High-Performance Compute

**Use Case**: CPU-intensive workloads, data processing

```toml
provider = "aws"
region = "us-west-2"

[[resources.compute]]
name = "compute-cluster-01"
type = "c7i-flex.large"     # Flexible compute
image = "amazon-linux"
count = 1

[resources.compute.tags]  
Environment = "production"
Purpose = "data-processing"
Team = "analytics"
Owner = "data-team"
ManagedBy = "Genesys"

[policies]
require_encryption = true
no_public_instances = true
require_tags = ["Environment", "Team", "Owner", "ManagedBy"]
```

**Estimated Cost**: ~$25-40/month (depending on usage patterns)

## Instance Management Patterns

### Multi-Environment Deployment

Deploy the same application across environments:

```bash
# Development
genesys interact  # Create: myapp-dev-server (t3.micro)
genesys execute ec2-myapp-dev-server-*.toml

# Staging  
genesys interact  # Create: myapp-staging-server (t3.small)
genesys execute ec2-myapp-staging-server-*.toml

# Production
genesys interact  # Create: myapp-prod-server (t3.medium)
genesys execute ec2-myapp-prod-server-*.toml --dry-run
genesys execute ec2-myapp-prod-server-*.toml
```

### Auto-Scaling Preparation

Create template configurations for auto-scaling groups:

```bash
# Create base instance configuration
genesys interact  # Configure: webapp-template

# Copy configuration for scaling
cp ec2-webapp-template-*.toml webapp-asg-template.toml
# Edit webapp-asg-template.toml for ASG use
```

### Development Team Workflows

Share instance configurations across team:

```bash
# Team lead creates template
genesys interact  # Create: team-dev-template

# Save to version control
git add ec2-team-dev-template-*.toml
git commit -m "Add team development server template"

# Team members deploy personal instances  
genesys execute ec2-team-dev-template-*.toml --dry-run
# Modify name to be unique: team-dev-alice, team-dev-bob
genesys execute ec2-team-dev-template-*.toml
```

## Security Best Practices

### Built-in Security Features

Genesys implements security best practices automatically:

- **Public IP Assignment**: Instances can be configured with public IPs for internet access
- **Encryption**: EBS volumes encrypted with AES-256
- **Security Groups**: Automatic SSH security group creation with your IP
- **Key Pair Management**: Create or select key pairs during instance creation
- **IAM Integration**: Uses your existing AWS credentials securely
- **Resource Tagging**: All resources tagged for management and compliance

### SSH Access Configuration

Genesys provides comprehensive SSH access setup during instance creation:

1. **Key Pair Configuration**:
   - Create new key pairs automatically (saved to ~/.ssh/)
   - Select existing key pairs from your AWS account
   - Key files are saved with proper 0600 permissions

2. **Security Group Setup**:
   - Automatic security group creation with SSH (port 22) enabled
   - Option to restrict SSH to your current IP (most secure)
   - Option to allow SSH from anywhere (0.0.0.0/0)
   - Custom CIDR block support

3. **Pre-flight Checks**:
   - Validates public IP assignment before SSH attempts
   - Checks security group rules for SSH access
   - Verifies key pair configuration
   - Waits for instance status checks to pass

### Additional Security Recommendations

1. **SSH Key Management**:
   - Genesys automatically creates and saves key pairs to ~/.ssh/
   - Key files are created with restrictive permissions (0600)
   - Keep your private keys secure and backed up

2. **Network Security**:
   - Use "my-ip" option to restrict SSH to your current IP
   - For production, consider VPN or bastion host access
   - Regularly review and update security group rules

3. **Data Protection**:
   - EBS encryption enabled by default
   - Use separate data volumes for sensitive data
   - Implement backup strategies for critical instances

4. **Access Control**:
   - Use IAM roles for instance permissions (not access keys)
   - Implement least-privilege access policies
   - Regular security audits and updates

## Cost Optimization

### Free Tier Maximization

AWS Free Tier includes:
- **750 hours/month** of t3.micro instances (24/7 for one instance)
- **30GB** of EBS storage per month
- **1GB** of data transfer out per month

**Strategy**: Use t3.micro for development, staging, and small production workloads.

### Right-Sizing Instances

Choose instance types based on actual requirements:

- **t3.micro**: Light workloads, development, small websites
- **t3.small**: Small databases, low-traffic web apps
- **t3.medium**: Medium web applications, small enterprise apps  
- **t3.large**: High-traffic websites, medium databases
- **c7i-flex.large**: CPU-intensive workloads, data processing
- **m7i-flex.large**: Memory-intensive applications, caching

### Cost Monitoring

```bash
# Check current instances and estimated costs
genesys list resources --service compute

# Use dry-run to see cost estimates before deployment
genesys execute ec2-new-instance-*.toml --dry-run

# Regular cleanup of unused development instances
genesys execute deletion ec2-dev-*.toml
```

## Troubleshooting

### SSH Connection Issues

**"Connection timed out"**
- Verify instance has a public IP address assigned
- Check security group allows SSH (port 22) from your IP
- Ensure instance is in "running" state and status checks passed
- Wait 2-3 minutes after launch for instance to be fully ready

**"Permission denied (publickey)"**
- Verify you're using the correct key pair file (.pem)
- Check key file permissions: `chmod 600 ~/.ssh/your-key.pem`
- Ensure the key pair matches the one assigned at instance launch
- Verify you're using the correct username for the AMI:
  - Ubuntu: `ubuntu`
  - Amazon Linux: `ec2-user`
  - Debian: `admin`
  - CentOS: `centos` or `ec2-user`

**"No route to host" or "Connection refused"**
- Security group may not have SSH rule - use 's' key in TUI to add rule
- Instance may be in a private subnet without public IP
- Check that your IP hasn't changed (re-add SSH rule with current IP)

**SSH Pre-flight Check Failures**
- TUI shows warnings before connection if issues detected
- Use "SSH Rules" button to add missing security group rules
- Use "New Key" button to create key pair if missing

### Common Issues

**"Instance name already exists"**
- Genesys validates names against existing instances
- Choose a unique name or check existing instances: `genesys list resources`

**"AMI resolution failed"**  
- Check internet connectivity
- Verify AWS credentials have ec2:DescribeImages permission
- Try different region if AMI not available

**"Insufficient instance capacity"**
- Try different availability zone
- Consider different instance type
- Check AWS service health dashboard

**"Permission denied"**
- Verify IAM permissions: ec2:RunInstances, ec2:CreateTags, ec2:DescribeInstances
- Check account limits and quotas
- Ensure credentials are valid

### Debug Commands

```bash
# Check configuration
genesys config show aws

# Verify permissions with dry-run
genesys execute ec2-instance-*.toml --dry-run

# List existing instances
genesys list resources --service compute --output json

# Check SSH connectivity manually
ssh -v -i ~/.ssh/your-key.pem ec2-user@<instance-public-ip>

# Verify security group rules
aws ec2 describe-security-groups --group-ids <sg-id>
```

## Integration with Other Services

### With S3 Storage

Create compute and storage together:

```bash
# Create S3 bucket for application data
genesys interact  # S3 bucket: myapp-data-bucket

# Create EC2 instance for application
genesys interact  # EC2 instance: myapp-server

# Deploy together
genesys execute s3-myapp-data-bucket-*.toml --apply
genesys execute ec2-myapp-server-*.toml --apply
```

### With Databases (Future)

Coming soon - integrated database and compute deployment:

```bash
# Future capability
genesys interact  # RDS database: myapp-database
genesys interact  # EC2 instance: myapp-server
# Automatic security group configuration between resources
```

## Best Practices Summary

1. **Naming Convention**: Use descriptive, environment-specific names
2. **Preview First**: Execute without `--apply` to preview all changes
3. **Tag Consistently**: Environment, Owner, Purpose, ManagedBy
4. **Security First**: Use private instances, enable encryption
5. **Cost Awareness**: Monitor Free Tier usage, right-size instances
6. **Regular Cleanup**: Terminate unused development instances
7. **Version Control**: Save configuration files in Git
8. **Team Standards**: Establish consistent patterns across team

## Future Enhancements

Planned features for EC2 workflows:

- **SSH Key Integration** - Automatic key pair creation and configuration
- **Auto Scaling Groups** - Template-based scaling configuration
- **Load Balancer Integration** - Automatic ALB/NLB setup
- **Security Group Management** - Custom rule configuration
- **Instance Monitoring** - CloudWatch integration and alerting
- **Backup Management** - Automatic AMI snapshot scheduling

---

**Next Steps**: Try the [Getting Started Guide](getting-started.md) for your first EC2 instance, or explore [Interactive Workflow Guide](interactive-workflow.md) for advanced usage patterns.