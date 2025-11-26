# Genesys Commands Reference

Complete reference for all Genesys CLI commands.

## Overview

Genesys provides several main commands for cloud resource management:

- `interact` - Interactive resource creation wizard
- `config` - Manage cloud provider credentials  
- `execute` - Deploy or delete resources from configuration files
- `list` / `discover` - List existing cloud resources
- `version` - Show version information

## genesys interact

Start an interactive wizard to create cloud resources.

```bash
genesys interact
```

### Workflow

1. **Provider Selection**: Choose from AWS, GCP, Azure, or Tencent Cloud
2. **Resource Type Selection**: Choose resource type (S3 Storage Bucket, Compute Instance, Database, Function, Network)
3. **Resource Configuration**: Interactive prompts specific to the selected resource
4. **Configuration Save**: Saves configuration as TOML file for later execution

### Example

```bash
genesys interact
# Select provider: aws
# Select resource: S3 Storage Bucket  
# Configure bucket settings...
# Saves: s3-mybucket-1234567890.toml
```

## genesys config

Manage cloud provider credentials and configuration.

### genesys config setup

Interactive setup wizard for cloud provider credentials.

```bash
genesys config setup
```

Guides you through:
- Selecting a cloud provider (AWS, GCP, Azure, Tencent)
- Choosing between local credentials or manual input
- Configuring regions and provider-specific settings
- Validating credentials
- Setting as default provider (optional)

### genesys config list

List all configured cloud providers.

```bash
genesys config list
```

Shows:
- All configured providers
- Default provider (marked with *)
- Region for each provider  
- Authentication method (local vs configured)

### genesys config show

Show configuration details for a specific provider.

```bash
genesys config show <provider>
```

Examples:
```bash
genesys config show aws
genesys config show gcp
```

### genesys config default

Set a provider as the default.

```bash
genesys config default <provider>
```

Examples:
```bash
genesys config default aws
genesys config default gcp
```

## genesys execute

Deploy or delete resources from configuration files. By default, shows a preview of changes without making any modifications.

### Basic Usage

```bash
genesys execute <config-file.toml>           # Preview changes (default, no changes made)
genesys execute <config-file.toml> --apply   # Apply changes and create resources
genesys execute <config-file.toml> --delete  # Delete resources defined in config
```

### Flags

- `--apply` - Apply the changes (required to actually create resources)
- `--delete` - Delete resources defined in the configuration file
- `--dry-run` - Show what would be done without making changes (this is now the default)
- `--force-deletion` - Force delete bucket contents including all versions (use with --delete)
- `-c, --config string` - Configuration file (TOML)
- `--provider string` - Cloud provider (default "aws")
- `--region string` - Cloud region
- `-o, --output string` - Output format (human|json) (default "human")

### Examples

```bash
# Preview S3 bucket creation (no changes made)
genesys execute s3-mybucket.toml

# Create S3 bucket (requires --apply)
genesys execute s3-mybucket.toml --apply

# Preview EC2 instance creation
genesys execute ec2-instance.toml

# Create EC2 instance
genesys execute ec2-instance.toml --apply

# Delete S3 bucket
genesys execute s3-mybucket.toml --delete

# Force delete S3 bucket with all versions
genesys execute s3-mybucket.toml --delete --force-deletion

# Delete EC2 instance
genesys execute ec2-instance.toml --delete

# Legacy deletion syntax (still supported)
genesys execute deletion s3-mybucket.toml
```

## genesys list / genesys discover

Discover existing resources in your cloud account.

```bash
genesys list                           # List all resources (alias for discover)
genesys discover                       # Discover all resources
genesys discover --service storage     # Discover only storage resources
genesys discover --provider aws        # Use specific provider
genesys discover --region us-west-2    # Use specific region
genesys discover --output json         # JSON output format
```

### Flags

- `--provider string` - Cloud provider (aws|gcp|azure) (default "aws")
- `--region string` - Cloud region
- `-o, --output string` - Output format (human|json) (default "human")
- `--service string` - Specific service to discover (storage|compute|network|database|serverless)

### Examples

```bash
# List all AWS resources
genesys list

# List only S3 buckets
genesys list --service storage

# List resources in specific region
genesys list --region us-east-1

# Get JSON output
genesys list --output json
```

## genesys version

Show version information.

```bash
genesys version
```

## Global Flags

All commands support these global flags:

- `-h, --help` - Help for the command
- `-v, --version` - Version for genesys (root command only)

## Configuration Files

Genesys uses TOML configuration files for resource management. These files are generated by the interactive workflow and used by the execute command.

### Example S3 Configuration

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

[policies]
require_encryption = true
no_public_buckets = true
require_tags = ["Environment", "ManagedBy", "Purpose"]
```

## Error Handling

Genesys provides clear error messages for common scenarios:

- **Provider not configured**: Prompts to run `genesys config setup`
- **Invalid configuration files**: Shows specific TOML parsing errors
- **Missing files**: Clear file not found messages
- **API errors**: Formatted cloud provider error responses

## Tips

1. **Preview is now default**: Running `genesys execute config.toml` shows a preview without making changes
2. **Use --apply to create**: Add the `--apply` flag when ready to create resources
3. **Use --delete to remove**: Add the `--delete` flag to delete resources defined in a config file
4. **Configure providers once**: Set up credentials with `genesys config setup`
5. **Use meaningful names**: Choose descriptive names for resources
6. **Keep configurations**: Save TOML files for resource lifecycle management
7. **Check existing resources**: Use `genesys list` to avoid naming conflicts
