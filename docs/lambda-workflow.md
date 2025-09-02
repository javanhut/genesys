# Lambda Function Builder Workflow

This document describes the Lambda function builder integration in Genesys, which provides automated building, layer management, and deployment for AWS Lambda functions.

## Overview

The Lambda builder provides:
- Automatic runtime detection from source code
- Containerized builds using Podman for consistency
- Automatic dependency layer creation and management
- Interactive configuration wizard
- Support for multiple runtimes (Python, Node.js, Go, Java)
- Integrated deployment with the existing Genesys workflow

## Workflow

### 1. Interactive Lambda Creation

Start the Lambda creation workflow:

```bash
genesys interact
# Select: aws → Function → Lambda Builder
```

The interactive wizard will:
1. Ask for source code location (local directory, git repo, or zip)
2. Auto-detect runtime from dependency files or code
3. Find and suggest the handler function
4. Configure memory, timeout, and environment variables
5. Collect IAM requirements (policies needed by your function)
6. Set up tags and triggers
7. Generate a TOML configuration file

### 2. Runtime Detection

The builder automatically detects the runtime by checking for:

**Python** (3.8 - 3.11):
- `requirements.txt`
- `Pipfile`
- `poetry.lock`
- `.py` files

**Node.js** (18.x, 20.x):
- `package.json`
- `yarn.lock`
- `pnpm-lock.yaml`
- `.js`, `.ts`, `.mjs` files

**Go** (1.x):
- `go.mod`
- `go.sum`
- `.go` files

**Java** (11, 17):
- `pom.xml`
- `build.gradle`
- `.java` files

### 3. Layer Management

Dependencies are automatically built into Lambda layers:

1. **Detection**: Scans for dependency files
2. **Containerized Build**: Uses Podman with official AWS Lambda images
3. **Caching**: Layers are cached locally to speed up subsequent builds
4. **Versioning**: Automatic version management for layers

Example layer structure:
```
/opt/
├── python/           # Python packages
├── nodejs/           # Node modules
└── java/lib/         # Java JARs
```

### 4. Build Process

The build process uses Podman containers for consistency:

```bash
# Python example
podman run --rm \
  -v ./src:/src:ro \
  -v ./layer:/opt/python \
  public.ecr.aws/lambda/python:3.11 \
  pip install -r /src/requirements.txt -t /opt/python

# Node.js example
podman run --rm \
  -v ./src:/src:ro \
  -v ./layer:/opt/nodejs \
  public.ecr.aws/lambda/nodejs:20 \
  npm install --production --prefix /opt/nodejs
```

### 5. Configuration Format

The builder generates a TOML configuration file:

```toml
[metadata]
name = "my-api-handler"
runtime = "python3.11"
handler = "app.lambda_handler"
description = "API handler function"

[build]
source_path = "/home/user/my-lambda-project"
build_method = "podman"
layer_auto = true
requirements_file = "requirements.txt"

[function]
memory_mb = 512
timeout_seconds = 30

[function.environment]
API_KEY = "your-api-key"
DATABASE_URL = "your-db-url"

[deployment]
function_url = true
cors_enabled = true
auth_type = "AWS_IAM"

[iam]
role_name = "genesys-lambda-my-api-handler"
required_policies = ["Basic CloudWatch Logs access", "S3 read access"]
auto_manage = true
auto_cleanup = true

[[triggers]]
type = "api_gateway"
path = "/{proxy+}"
method = "ANY"

[layer]
name = "my-api-handler-deps"
description = "Dependencies for my-api-handler"
compatible_runtimes = ["python3.11"]
```

### 6. Deployment

Deploy the Lambda function:

```bash
# Preview deployment
genesys execute lambda_myfunction_python311.toml --dry-run

# Deploy
genesys execute lambda_myfunction_python311.toml
```

The deployment process:
0. Ensures IAM role exists (creates if needed)
1. Builds the function package
2. Creates/updates the dependency layer
3. Uploads both to AWS
4. Creates/updates the Lambda function
5. Configures triggers and permissions
6. Returns the function URL (if enabled)

## IAM Role Management

Genesys automatically manages IAM roles for Lambda functions, eliminating the need to manually create and configure roles.

### Automatic Role Creation

During the interactive configuration, you'll be asked about the permissions your function needs:

```bash
? What AWS services will your function access? (Select all that apply)
  ✓ CloudWatch Logs (for basic logging)
  ✓ S3 buckets (for file storage)
    DynamoDB tables (for database access)  
    SQS queues (for message processing)
    SNS topics (for notifications)
```

Genesys will:
1. Create an appropriately named IAM role (e.g., `genesys-lambda-my-function`)
2. Attach only the necessary policies based on your selections
3. Configure the Lambda execution trust policy
4. Update the TOML configuration with the created role ARN

### Zero-Touch Deployment

When you deploy, Genesys automatically:
- Creates the IAM role if it doesn't exist
- Updates policies if requirements change
- Handles AWS IAM propagation delays
- Never requires manual IAM configuration

### Cleanup During Deletion

When you delete a Lambda function, Genesys will:
- Remove the Lambda function and layers
- Clean up the auto-created IAM role and policies
- Only delete roles that Genesys created and manages

### Policy Templates

Available policy templates:
- **Basic CloudWatch Logs access** - Essential for all functions
- **S3 read access** - Read objects from S3 buckets
- **S3 read/write access** - Full S3 bucket access
- **DynamoDB read access** - Query and get items  
- **DynamoDB read/write access** - Full table operations
- **SQS read/write access** - Send and receive messages
- **SNS publish access** - Publish to topics

### Using External Roles

If you have existing IAM roles, set `auto_manage = false`:

```toml
[iam]
role_name = "my-existing-lambda-role"
auto_manage = false
auto_cleanup = false
```

## Advanced Features

### Custom Runtimes

Add support for custom runtimes by extending the runtime detection:

```go
// In pkg/lambda/runtime.go
"ruby3.2": {
    Name:            "ruby3.2",
    Version:         "3.2",
    BuildImage:      "public.ecr.aws/lambda/ruby:3.2",
    LayerPath:       "/opt/ruby/gems",
    FileExtensions:  []string{".rb"},
    DependencyFiles: []string{"Gemfile", "Gemfile.lock"},
}
```

### Layer Sharing

Layers can be shared across multiple functions:

```toml
[function]
layers = ["arn:aws:lambda:region:account:layer:shared-deps:1"]
```

### Multi-Architecture Support

Build for different architectures:

```toml
[build]
architectures = ["x86_64", "arm64"]
```

### Build Hooks

Add pre/post build commands:

```toml
[build]
pre_build = ["npm test", "npm run lint"]
post_build = ["./verify.sh"]
```

## Troubleshooting

### Podman Not Found

Install Podman:
```bash
# Ubuntu/Debian
sudo apt-get install podman

# macOS
brew install podman

# Fedora
sudo dnf install podman
```

### Build Failures

Check build logs:
```bash
genesys lambda build --debug lambda_config.toml
```

### Layer Size Limits

AWS Lambda layers have a 50MB limit (zipped). If your dependencies exceed this:
1. Use multiple layers
2. Include only production dependencies
3. Consider using container images instead

### Permission Issues

Ensure Podman can access directories:
```bash
# Fix permissions
chmod -R 755 ./src
```

## Best Practices

1. **Use Layers**: Always use layers for dependencies to reduce function size
2. **Cache Layers**: Reuse layers across functions with similar dependencies
3. **Minimize Size**: Only include production dependencies
4. **Version Lock**: Use lock files (requirements.txt, package-lock.json)
5. **Test Locally**: Use SAM or LocalStack for local testing
6. **Monitor Costs**: Use CloudWatch to monitor invocations and costs

## Examples

### Python Flask API

```bash
# Create project
mkdir flask-api && cd flask-api
echo "Flask==2.3.0" > requirements.txt
cat > app.py << EOF
from flask import Flask
app = Flask(__name__)

@app.route('/')
def lambda_handler(event, context):
    return {'statusCode': 200, 'body': 'Hello from Lambda!'}
EOF

# Create Lambda
genesys interact  # Select AWS → Function
```

### Node.js Express API

```bash
# Create project
mkdir express-api && cd express-api
npm init -y
npm install express

cat > index.js << EOF
const express = require('express');
const app = express();

exports.handler = async (event, context) => {
    return {
        statusCode: 200,
        body: JSON.stringify({ message: 'Hello from Lambda!' })
    };
};
EOF

# Create Lambda
genesys interact  # Select AWS → Function
```

### Go REST API

```bash
# Create project
mkdir go-api && cd go-api
go mod init example.com/api

cat > main.go << EOF
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context, event interface{}) (string, error) {
    return "Hello from Lambda!", nil
}

func main() {
    lambda.Start(HandleRequest)
}
EOF

# Create Lambda
genesys interact  # Select AWS → Function
```

## Integration with CI/CD

### GitHub Actions

```yaml
name: Deploy Lambda
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install Genesys
        run: |
          curl -fsSL https://genesys.sh/install | sh
      - name: Configure AWS
        run: |
          genesys config setup --provider aws \
            --region ${{ secrets.AWS_REGION }} \
            --access-key ${{ secrets.AWS_ACCESS_KEY }} \
            --secret-key ${{ secrets.AWS_SECRET_KEY }}
      - name: Deploy Lambda
        run: |
          genesys execute lambda_config.toml
```

### GitLab CI

```yaml
deploy:
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - apk add --no-cache curl
    - curl -fsSL https://genesys.sh/install | sh
  script:
    - genesys config setup --provider aws
    - genesys execute lambda_config.toml
```

## Future Enhancements

- **Container Image Support**: Deploy Lambda functions as container images
- **Local Testing**: Integrated local Lambda testing
- **Performance Profiling**: Built-in performance analysis
- **Cost Estimation**: Pre-deployment cost estimates
- **Multi-Region Deployment**: Deploy to multiple regions simultaneously
- **Blue/Green Deployments**: Safe deployment strategies
- **Canary Releases**: Gradual rollout support