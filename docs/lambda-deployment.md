# Lambda Function Deployment

Genesys now supports full Lambda function deployment, including building, packaging, and deploying functions to AWS Lambda.

## Features

- **Container-based builds** using Podman for consistent, reproducible builds
- **Automatic dependency management** with layer creation for Python, Node.js, Go, and Java
- **Multi-architecture support** for both x86_64 and ARM64 (Graviton2)
- **Function URL configuration** for easy HTTP endpoints
- **Automated IAM role management** with policy templates and cleanup
- **State tracking** to manage deployed functions

## Prerequisites

1. **Podman** must be installed and running
2. **AWS credentials** configured via `genesys config setup`
3. **Source code** with appropriate handler function

## Configuration Format

Lambda functions are configured using TOML files with the following structure:

```toml
[metadata]
  name = "MyFunction"
  runtime = "python3.12"
  handler = "app.lambda_handler"
  description = "My Lambda function"

[build]
  source_path = "/path/to/source"
  build_method = "podman"
  layer_auto = true
  requirements_file = "requirements.txt"  # Optional

[function]
  memory_mb = 512
  timeout_seconds = 30

[deployment]
  function_url = true
  cors_enabled = true
  auth_type = "AWS_IAM"
  architecture = "arm64"  # or "x86_64"

[iam]  # Automatically managed by Genesys
  role_name = "genesys-lambda-MyFunction"
  required_policies = ["Basic CloudWatch Logs access"]
  auto_manage = true
  auto_cleanup = true

[layer]  # Optional, created automatically if dependencies exist
  name = "MyFunction-deps"
  description = "Dependencies layer"
  compatible_runtimes = ["python3.12"]
```

## IAM Role Management

Genesys automatically manages IAM roles for Lambda functions, providing seamless deployment without manual role setup.

### Automatic Role Creation

When deploying a Lambda function, Genesys automatically:

1. **Creates IAM role** if it doesn't exist
2. **Attaches required policies** based on your function's needs
3. **Updates configuration** with the created role ARN
4. **Manages cleanup** during function deletion

### Configuration Options

The `[iam]` section supports the following options:

```toml
[iam]
  role_name = "genesys-lambda-MyFunction"    # Auto-generated if not specified
  required_policies = [                       # Policies to attach
    "Basic CloudWatch Logs access",
    "S3 read access", 
    "DynamoDB read/write access"
  ]
  auto_manage = true                         # Let Genesys manage the role
  auto_cleanup = true                        # Delete role when function is deleted
  role_arn = ""                             # Auto-populated after creation
  managed_by = "genesys"                    # Auto-populated for tracking
```

### Available Policy Templates

Genesys provides predefined policy templates:

- `Basic CloudWatch Logs access` - Essential logging permissions
- `S3 read access` - Read from S3 buckets  
- `S3 read/write access` - Full S3 access
- `DynamoDB read access` - Read from DynamoDB tables
- `DynamoDB read/write access` - Full DynamoDB access
- `SQS read/write access` - Send/receive SQS messages
- `SNS publish access` - Publish to SNS topics

### Role Lifecycle

1. **During Deployment**: Role is created or updated automatically
2. **Function Updates**: Existing role is validated and updated if needed
3. **Function Deletion**: Role is cleaned up if `auto_cleanup = true`

### Using External Roles

To use an existing IAM role, set `auto_manage = false`:

```toml
[iam]
  role_name = "my-existing-role"
  auto_manage = false
  auto_cleanup = false
```

## Deployment Process

### 1. Create Configuration

Use interactive mode to create a Lambda configuration:

```bash
genesys interact
# Select: serverless > lambda > Configure new function
```

Or create a TOML file manually.

### 2. Preview Deployment

Always preview changes before deploying:

```bash
genesys execute lambda_config.toml --dry-run
```

### 3. Deploy Function

Deploy the function to AWS:

```bash
genesys execute lambda_config.toml --apply
```

This will:
0. Ensure IAM role exists (create if needed)
1. Build the function code using Podman
2. Create a layer if dependencies are detected
3. Deploy the function to AWS Lambda
4. Configure function URL if enabled

### 4. Update Function

To update an existing function, simply run the deploy command again:

```bash
genesys execute lambda_config.toml --apply
```

### 5. Delete Function

To delete a deployed function:

```bash
genesys execute deletion lambda_config.toml
```

This will:
1. Delete the Lambda function
2. Delete associated layer (if created)
3. Clean up IAM role (if auto-managed)
4. Remove any configured triggers

## Build Process Details

### Python Functions

- Detects `requirements.txt`, `Pipfile`, or `poetry.lock`
- Builds dependencies in Lambda-compatible environment
- Creates layer with Python packages in correct structure

### Node.js Functions

- Detects `package.json` and lock files
- Supports npm, yarn, and pnpm
- Creates layer with node_modules

### Go Functions

- Compiles to static binary named `bootstrap`
- Supports cross-compilation for target architecture
- No layer needed (single binary)

### Java Functions

- Supports Maven and Gradle builds
- Creates layer with dependency JARs
- Handles classpath configuration

## Architecture Selection

Choose between x86_64 and ARM64 based on your needs:

- **x86_64**: Standard architecture, widely compatible
- **ARM64**: Better price/performance (up to 34% cost savings)

## Function URLs

When enabled, provides a dedicated HTTPS endpoint:

```
https://{function-name}.lambda-url.{region}.on.aws/
```

Configure CORS and authentication as needed.

## Troubleshooting

### Build Failures

1. Ensure Podman is installed: `podman --version`
2. Check source path exists and contains handler file
3. Verify dependency files are present and valid

### Deployment Failures

1. Verify AWS credentials: `genesys config verify`
2. Check IAM permissions for Lambda operations
3. Ensure function name is unique in the region

### IAM Issues

1. **Role creation fails**: Ensure your AWS credentials have IAM permissions
2. **Policy attachment fails**: Check policy names exist and are accessible
3. **External role issues**: Verify the role exists and allows Lambda to assume it

### Common Issues

- **"entrypoint requires handler name"**: Fixed in latest version
- **"Unable to determine service/operation"**: Check AWS credentials
- **"Function already exists"**: Function will be updated automatically

## Best Practices

1. Always use `--dry-run` before deploying
2. Keep functions small and focused
3. Use layers for shared dependencies
4. Monitor costs with appropriate memory/timeout settings
5. Use ARM64 for cost optimization when possible