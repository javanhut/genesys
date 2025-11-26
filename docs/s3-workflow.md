# S3 Bucket Management Workflow

Complete guide for managing S3 buckets with Genesys interactive workflow.

## Overview

Genesys provides a complete S3 bucket lifecycle management through its interactive workflow:

1. **Interactive Configuration** - Create bucket configuration through guided prompts
2. **Configuration Review** - Review generated TOML configuration
3. **Preview** - Preview bucket creation without making changes (default behavior)
4. **Deployment** - Create the actual S3 bucket with `--apply`
5. **Management** - List and manage existing buckets
6. **Deletion** - Clean removal of buckets with `--delete`

## Step-by-Step Workflow

### Step 1: Interactive Configuration Creation

Start the interactive workflow:

```bash
genesys interact
```

Follow the prompts:

1. **Select provider**: Choose `aws`
2. **Select resource**: Choose `S3 Storage Bucket`
3. **Configure bucket settings**:
   - **Bucket name**: Enter globally unique DNS-compliant name
   - **Region**: Select from common AWS regions
   - **Versioning**: Enable to keep multiple versions of objects (recommended: yes)
   - **Encryption**: Enable AES256 encryption at rest (recommended: yes) 
   - **Public access**: Allow public access (recommended: no for security)
   - **Tags**: Add metadata tags
     - Default tags: Environment, ManagedBy, Purpose
     - Optional custom tags
   - **Lifecycle policies**: Configure automatic archiving/deletion
     - Archive to Glacier after N days (optional)
     - Delete objects after N days (optional)

The configuration will be saved as: `s3-<bucket-name>-<timestamp>.toml`

### Step 2: Review Configuration

Review the generated configuration file:

```bash
cat s3-mybucket-1234567890.toml
```

Example configuration:
```toml
provider = "aws"
region = "us-east-1"

[[resources.storage]]
name = "my-test-bucket"
type = "bucket"
versioning = true
encryption = true
public_access = false

[resources.storage.tags]
Environment = "development"
ManagedBy = "Genesys"
Purpose = "demo"

[resources.storage.lifecycle]
archive_after_days = 90
delete_after_days = 365

[policies]
require_encryption = true
no_public_buckets = true
require_tags = ["Environment", "ManagedBy", "Purpose"]
```

### Step 3: Preview (Default Behavior)

Preview what will be created without making actual changes:

```bash
genesys execute s3-mybucket-1234567890.toml
```

Output shows:
- Bucket name and configuration details
- All settings that would be applied
- No actual AWS resources are created

### Step 4: Deploy the Bucket

Create the actual S3 bucket with `--apply`:

```bash
genesys execute s3-mybucket-1234567890.toml --apply
```

This will:
- Validate AWS credentials are configured
- Create the S3 bucket with specified name and region
- Apply versioning, encryption, and public access settings
- Set bucket tags
- Configure lifecycle policies if specified
- Show success confirmation

### Step 5: List Resources

View existing S3 buckets:

```bash
# List all resources
genesys list resources

# List only storage resources
genesys list resources --service storage

# List with JSON output
genesys list resources --service storage --output json
```

### Step 6: Deletion

When the bucket is no longer needed:

```bash
# Preview deletion
genesys execute s3-mybucket-1234567890.toml --delete

# Actually delete the bucket (force deletion for buckets with content)
genesys execute s3-mybucket-1234567890.toml --delete --force-deletion
```

The deletion process:
- Shows what will be deleted
- Empties the bucket if it contains objects
- Deletes the bucket
- Confirms successful deletion

## Configuration Options

### Bucket Naming Rules

S3 bucket names must be:
- Globally unique across all AWS accounts
- 3-63 characters long
- Lowercase letters, numbers, hyphens only
- DNS-compliant (no spaces, uppercase, or special characters)

### Region Selection

Common AWS regions available:
- `us-east-1` - US East (N. Virginia) - Default region
- `us-east-2` - US East (Ohio)
- `us-west-1` - US West (N. California)  
- `us-west-2` - US West (Oregon)
- `eu-west-1` - Europe (Ireland)
- `eu-central-1` - Europe (Frankfurt)
- `ap-southeast-1` - Asia Pacific (Singapore)
- `ap-northeast-1` - Asia Pacific (Tokyo)

### Security Settings

**Versioning**:
- Keeps multiple versions of objects
- Protects against accidental deletion or modification
- Recommended: Enable

**Encryption**:
- Server-side encryption with AES256
- Encrypts objects at rest
- Recommended: Enable

**Public Access**:
- Controls whether bucket allows public read/write
- Security risk if enabled inappropriately
- Recommended: Disable unless specifically needed

### Tags

**Default Tags** (automatically added):
- `Environment: development`
- `ManagedBy: Genesys`
- `Purpose: demo`

**Custom Tags**:
- Add additional key-value pairs for organization
- Useful for billing, access control, automation

### Lifecycle Policies

**Archive to Glacier**:
- Automatically move objects to cheaper storage after specified days
- Common setting: 90 days
- Reduces storage costs for infrequently accessed data

**Automatic Deletion**:
- Permanently delete objects after specified days
- Common setting: 365 days (1 year)
- Helps with compliance and cost management

## Prerequisites

### AWS Configuration

Before using S3 workflow, configure AWS credentials:

```bash
genesys config setup
```

Choose AWS and either:
- Use existing local AWS credentials (AWS CLI, environment variables)
- Enter credentials manually (Access Key ID, Secret Access Key)

### Permissions Required

AWS credentials must have permissions for:
- `s3:CreateBucket`
- `s3:DeleteBucket`
- `s3:GetBucketLocation`
- `s3:GetBucketVersioning`
- `s3:PutBucketVersioning`
- `s3:GetBucketEncryption`
- `s3:PutBucketEncryption`
- `s3:GetBucketTagging`
- `s3:PutBucketTagging`
- `s3:ListAllMyBuckets`

## Best Practices

1. **Preview is default**: Running execute shows a preview before deployment
2. **Use descriptive names**: Include project, environment, or purpose in bucket names
3. **Enable versioning**: Protects against accidental data loss
4. **Enable encryption**: Protects data at rest
5. **Disable public access**: Unless specifically required for static websites
6. **Add meaningful tags**: Helps with organization and cost tracking
7. **Configure lifecycle policies**: Manage costs and compliance
8. **Keep configuration files**: Needed for bucket deletion and documentation

## Troubleshooting

### Common Errors

**Bucket name already exists**:
- S3 bucket names are globally unique
- Try a different name with additional identifiers

**Access denied**:
- Check AWS credentials are configured: `genesys config show aws`
- Verify IAM permissions include required S3 actions

**Invalid bucket name**:
- Must be DNS-compliant (lowercase, no spaces, 3-63 characters)
- Cannot contain uppercase letters or special characters

**Region errors**:
- Some AWS features not available in all regions
- Try a different region like us-east-1

### Getting Help

- Check command help: `genesys execute --help`
- List configured providers: `genesys config list`
- Validate provider configuration: `genesys config show aws`
- Preview changes: `genesys execute config.toml` (default behavior)