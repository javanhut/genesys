# Configuration Guide

Complete guide for configuring cloud provider credentials and settings in Genesys.

## Overview

Genesys requires cloud provider credentials to be configured before creating or managing resources. The configuration system supports multiple providers and authentication methods.

## Provider Configuration

### Setup Command

```bash
genesys config setup
```

This interactive wizard guides you through:
1. **Provider Selection** - Choose cloud provider
2. **Credential Detection** - Check for existing local credentials  
3. **Credential Configuration** - Set up authentication
4. **Region Selection** - Choose default region
5. **Validation** - Verify credentials work
6. **Default Setting** - Optionally set as default provider

### Supported Providers

- **AWS** - Amazon Web Services (fully implemented)
- **GCP** - Google Cloud Platform (configuration support)
- **Azure** - Microsoft Azure (configuration support)
- **Tencent** - Tencent Cloud (configuration support)

## AWS Configuration

### Credential Methods

AWS supports two authentication methods:

#### 1. Local Credentials (Recommended)

Uses existing AWS credentials from:
- **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- **AWS Profile**: Set via `AWS_PROFILE` environment variable
- **AWS Config Files**: `~/.aws/credentials` and `~/.aws/config`
- **Default Region**: From `AWS_DEFAULT_REGION` environment variable

If local credentials are detected, you'll see:
```
Found existing AWS credentials:
  - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
  - AWS Profile: default
  - Region: us-east-1
```

#### 2. Manual Configuration

Enter credentials directly:
- **Access Key ID**: AWS access key ID
- **Secret Access Key**: AWS secret access key  
- **Session Token**: Optional temporary session token
- **Profile**: Optional AWS profile name

### Region Selection

Choose from common AWS regions:
- `us-east-1` - US East (N. Virginia) - Default region
- `us-east-2` - US East (Ohio)
- `us-west-1` - US West (N. California)
- `us-west-2` - US West (Oregon)
- `eu-west-1` - Europe (Ireland)
- `eu-central-1` - Europe (Frankfurt)
- `ap-southeast-1` - Asia Pacific (Singapore)
- `ap-northeast-1` - Asia Pacific (Tokyo)

### Validation

AWS credentials are validated by:
1. Creating an AWS client with provided credentials
2. Testing basic API access
3. Verifying regional access if specified

## GCP Configuration

### Credential Methods

#### 1. Local Credentials
- **Service Account Key**: Path in `GOOGLE_APPLICATION_CREDENTIALS`
- **Project ID**: From `GOOGLE_CLOUD_PROJECT` environment variable
- **gcloud CLI**: Default credentials from `~/.config/gcloud`

#### 2. Manual Configuration
- **Project ID**: GCP project identifier
- **Service Account Key**: Path to JSON key file
- **Region**: GCP region (e.g., `us-central1`)

### Region Selection

Common GCP regions:
- `us-central1` - US Central (Iowa)
- `us-east1` - US East (South Carolina)  
- `us-west1` - US West (Oregon)
- `europe-west1` - Europe West (Belgium)
- `asia-east1` - Asia East (Taiwan)

## Azure Configuration

### Credential Methods

#### 1. Local Credentials
- **Service Principal**: From environment variables
  - `AZURE_CLIENT_ID`
  - `AZURE_CLIENT_SECRET`  
  - `AZURE_TENANT_ID`
  - `AZURE_SUBSCRIPTION_ID`
- **Azure CLI**: Default credentials from `~/.azure`

#### 2. Manual Configuration

**Service Principal Authentication**:
- **Client ID**: Application (client) ID
- **Client Secret**: Client secret value
- **Tenant ID**: Directory (tenant) ID
- **Subscription ID**: Azure subscription ID

**Managed Identity Authentication**:
- **Subscription ID**: Azure subscription ID

### Region Selection

Common Azure regions:
- `eastus` - East US
- `westus2` - West US 2
- `centralus` - Central US
- `northeurope` - North Europe
- `westeurope` - West Europe

## Tencent Cloud Configuration

### Credential Methods

#### 1. Local Credentials
- **Environment Variables**: `TENCENTCLOUD_SECRET_ID`, `TENCENTCLOUD_SECRET_KEY`
- **Region**: From `TENCENTCLOUD_REGION`
- **Tencent CLI**: Default credentials from `~/.tccli`

#### 2. Manual Configuration
- **Secret ID**: Tencent Cloud secret ID
- **Secret Key**: Tencent Cloud secret key  
- **Security Token**: Optional temporary token
- **Region**: Tencent Cloud region

### Region Selection

Common Tencent regions:
- `ap-beijing` - Beijing
- `ap-shanghai` - Shanghai
- `ap-guangzhou` - Guangzhou
- `ap-singapore` - Singapore

## Configuration Management

### List Providers

View all configured providers:

```bash
genesys config list
```

Example output:
```
Configured Cloud Providers:

  ✓ AWS *
     Region: us-east-1
     Auth: Local Credentials

  ✓ GCP
     Region: us-central1
     Auth: Manual Configuration

* = Default provider
```

### Show Provider Details

View detailed configuration for a specific provider:

```bash
genesys config show aws
genesys config show gcp
```

Example output:
```
Configuration for AWS:
========================================
Provider: AWS
Region: us-east-1  
Default: true
Use Local Credentials: true

Credential Validation:
  ✓ Credentials are valid
```

### Set Default Provider

Change the default provider:

```bash
genesys config default aws
genesys config default gcp
```

The default provider is used when:
- Creating new configuration files
- No provider is explicitly specified
- Using provider-agnostic commands

## Configuration Files

### Storage Location

Configuration files are stored in:
```
~/.genesys/
├── config.json          # Global configuration
├── aws.json             # AWS provider configuration  
├── gcp.json             # GCP provider configuration
├── azure.json           # Azure provider configuration
└── tencent.json         # Tencent provider configuration
```

### Security

- **File Permissions**: Provider config files use restrictive permissions (0600)
- **Credential Storage**: Credentials are encrypted when stored
- **Environment Variables**: Can override stored credentials
- **Temporary Tokens**: Session tokens are supported for temporary access

### Global Configuration

The global config file (`config.json`) contains:
```json
{
  "default_provider": "aws",
  "version": "1.0"
}
```

### Provider Configuration Format

Example AWS configuration (`aws.json`):
```json
{
  "provider": "aws",
  "region": "us-east-1",
  "credentials": {
    "access_key_id": "AKIA...",
    "secret_access_key": "...",
    "session_token": "..."
  },
  "use_local": false,
  "default_config": true
}
```

## Environment Variables

### Override Configuration

Environment variables can override stored configuration:

**AWS**:
- `AWS_ACCESS_KEY_ID` - Access key ID
- `AWS_SECRET_ACCESS_KEY` - Secret access key
- `AWS_SESSION_TOKEN` - Session token
- `AWS_DEFAULT_REGION` - Default region
- `AWS_PROFILE` - AWS profile to use

**GCP**:
- `GOOGLE_APPLICATION_CREDENTIALS` - Service account key file path
- `GOOGLE_CLOUD_PROJECT` - Project ID

**Azure**:
- `AZURE_CLIENT_ID` - Client ID
- `AZURE_CLIENT_SECRET` - Client secret
- `AZURE_TENANT_ID` - Tenant ID
- `AZURE_SUBSCRIPTION_ID` - Subscription ID

**Tencent**:
- `TENCENTCLOUD_SECRET_ID` - Secret ID
- `TENCENTCLOUD_SECRET_KEY` - Secret key
- `TENCENTCLOUD_REGION` - Region

## Troubleshooting

### Common Issues

**Provider Not Configured**:
```
Error: AWS provider not configured
```
Solution: Run `genesys config setup` to configure the provider.

**Invalid Credentials**:
```
Error: AWS credentials validation failed
```
Solutions:
- Check credentials are correct
- Verify IAM permissions
- Check network connectivity
- Try different authentication method

**No Local Credentials**:
```
No local AWS credentials found
```
Solutions:
- Install and configure AWS CLI
- Set environment variables
- Use manual credential entry

**Permission Denied**:
```
Access denied to AWS services
```
Solutions:
- Check IAM policy permissions
- Verify account has required service access
- Contact AWS administrator

### Validation Process

Each provider goes through validation:

1. **Credential Format Check** - Verify credential format is correct
2. **Authentication Test** - Attempt to authenticate with provider
3. **Permission Test** - Test basic service permissions
4. **Regional Access** - Verify access to specified region

### Getting Help

- Check configuration status: `genesys config list`
- View provider details: `genesys config show <provider>`
- Reconfigure if needed: `genesys config setup`
- Use help flags: `genesys config --help`

## Best Practices

### Security

- **Use IAM roles** when possible instead of long-term credentials
- **Rotate credentials** regularly
- **Use least privilege** permissions
- **Don't share credentials** between environments
- **Use temporary tokens** for CI/CD systems

### Organization

- **Configure multiple providers** for multi-cloud deployments
- **Set meaningful default provider** for your primary cloud
- **Document credential sources** for team members
- **Use consistent regions** within projects
- **Test credentials** after configuration

### Maintenance

- **Validate credentials** periodically
- **Update expired credentials** promptly
- **Monitor credential usage** through provider logs
- **Clean up unused providers** to reduce complexity
- **Keep configuration files secure** with proper file permissions