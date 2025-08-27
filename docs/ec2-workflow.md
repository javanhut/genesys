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

Always use dry-run to preview changes:
```bash
genesys execute ec2-my-instance-*.toml --dry-run
```

### Step 6: Deploy Instance

```bash
genesys execute ec2-my-instance-*.toml
```

### Step 7: Manage Instance

```bash
# List your instances
genesys list resources --service compute

# Update instance tags (if needed)
genesys execute ec2-my-instance-*.toml

# Terminate when done
genesys execute deletion ec2-my-instance-*.toml
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

- **Private Networking**: Instances created in private subnets by default
- **Encryption**: EBS volumes encrypted with AES-256
- **Security Groups**: Restrictive rules, SSH only from your IP
- **IAM Integration**: Uses your existing AWS credentials securely
- **Resource Tagging**: All resources tagged for management and compliance

### Additional Security Recommendations

1. **SSH Key Management**:
   ```bash
   # Create EC2 key pair (do this separately)
   aws ec2 create-key-pair --key-name my-dev-key --output text --query 'KeyMaterial' > my-dev-key.pem
   chmod 400 my-dev-key.pem
   
   # Reference in security group configuration
   # (Future: Genesys will support SSH key configuration)
   ```

2. **Network Security**:
   - Instances are private by default (secure)
   - Use bastion hosts or VPN for SSH access
   - Consider AWS Systems Manager Session Manager for secure access

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

# AWS CLI verification
aws ec2 describe-instances --output table
aws sts get-caller-identity  # Verify credentials
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
genesys execute s3-myapp-data-bucket-*.yaml
genesys execute ec2-myapp-server-*.toml
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
2. **Always Dry-Run**: Preview all changes, especially in production
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