# Genesys Documentation

Genesys is a simplicity-first Infrastructure as a Service tool that focuses on outcomes rather than resources. It provides a discovery-first approach to cloud resource management with human-readable plans.

## Documentation Structure

- [Getting Started](getting-started.md) - Quick start guide and installation
- [Commands](commands.md) - Complete command reference
- [Interactive Workflow](interactive-workflow.md) - Step-by-step interactive usage
- [S3 Management](s3-workflow.md) - S3 bucket lifecycle management
- [Configuration](configuration.md) - Provider configuration and credentials
- [Architecture](../ARCHITECTURE.md) - Technical architecture overview
- [Examples](examples.md) - Example configurations and workflows

## Quick Start

1. Configure your cloud provider credentials:
   ```bash
   genesys config setup
   ```

2. Start the interactive workflow:
   ```bash
   genesys interact
   ```

3. List existing resources:
   ```bash
   genesys list resources
   ```

4. Execute a configuration:
   ```bash
   genesys execute config.yaml --dry-run
   genesys execute config.yaml
   ```

## Key Features

- Interactive provider and resource selection
- YAML-based configuration management
- Dry-run capability for safe previews
- Multi-cloud provider support (AWS, GCP, Azure, Tencent)
- Direct API integration for fast performance
- Configuration-driven resource lifecycle management

## Supported Cloud Providers

- AWS (implemented with direct API calls)
- GCP (configuration support)
- Azure (configuration support) 
- Tencent Cloud (configuration support)

## Project Structure

```
genesys/
├── cmd/genesys/           # Main CLI application
├── pkg/config/            # Configuration management
├── pkg/provider/          # Cloud provider abstractions
├── pkg/intent/            # Intent parsing
├── pkg/planner/           # Resource planning
├── docs/                  # Documentation
└── examples/              # Example configurations
```

## Installation

### From Source

```bash
git clone <repository-url>
cd genesys
go build -o genesys ./cmd/genesys
```

### Usage

The tool provides several commands for managing cloud resources:

- `genesys interact` - Interactive resource creation workflow
- `genesys config` - Manage cloud provider credentials
- `genesys execute` - Deploy or delete resources from configuration files
- `genesys list` - Discover existing resources in your cloud account
- `genesys version` - Show version information