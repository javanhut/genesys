# DynamoDB Table Workflow Guide

Complete guide to creating, managing, and browsing AWS DynamoDB tables using Genesys interactive workflows and TUI.

## Overview

DynamoDB is AWS's fully managed NoSQL database service. Genesys simplifies DynamoDB table management with:

- **Interactive Configuration** - Guided setup for table creation
- **TUI Management** - Browse tables, view items, and manage tables visually
- **Flexible Billing** - Support for both on-demand and provisioned capacity modes
- **Key Schema Configuration** - Easy partition and sort key setup
- **Stream Configuration** - Enable DynamoDB Streams for event-driven architectures
- **TTL Support** - Configure Time To Live for automatic item expiration

## Supported Features

### Billing Modes
- **On-Demand (PAY_PER_REQUEST)** - Pay per request, no capacity planning required
- **Provisioned** - Set fixed read/write capacity units for predictable workloads

### Key Types
- **Partition Key (Hash Key)** - Required primary key component
- **Sort Key (Range Key)** - Optional secondary key for complex queries

### Attribute Types
- **String (S)** - Text data
- **Number (N)** - Numeric data
- **Binary (B)** - Binary data

### Additional Features
- **DynamoDB Streams** - Capture item-level changes for triggers and replication
- **Time To Live (TTL)** - Automatic deletion of expired items
- **Global Secondary Indexes (GSI)** - Alternative query patterns
- **Local Secondary Indexes (LSI)** - Additional sort key options

## Interactive Configuration

### Step 1: Start Interactive Mode

```bash
genesys interact
```

### Step 2: Select Provider and Resource

1. **Provider**: Choose `aws`
2. **Resource Type**: Choose `Database`
3. **Database Type**: Choose `DynamoDB (NoSQL)`

### Step 3: Configure Table

Follow the interactive prompts:

#### Table Details
- **Table Name**: 3-255 characters, alphanumeric, underscores, hyphens, and dots
- **Billing Mode**: On-Demand or Provisioned

#### Capacity Configuration (Provisioned Mode Only)
- **Read Capacity Units (RCU)**: Number of reads per second (1 RCU = 1 strongly consistent read/sec for items up to 4KB)
- **Write Capacity Units (WCU)**: Number of writes per second (1 WCU = 1 write/sec for items up to 1KB)

#### Key Schema
- **Partition Key**: Required - attribute name and type
- **Sort Key**: Optional - attribute name and type

#### Stream Configuration
- **Enable Streams**: Capture item changes
- **Stream View Type**: NEW_AND_OLD_IMAGES, NEW_IMAGE, OLD_IMAGE, or KEYS_ONLY

#### TTL Configuration
- **Enable TTL**: Automatic item expiration
- **TTL Attribute**: Attribute name containing expiration timestamp (Unix epoch)

#### Tags
- Add optional key-value pairs for resource organization

### Step 4: Review and Save

Configuration is saved to `~/.genesys/resources/dynamodb/<table-name>.toml`

```bash
Configuration saved to: ~/.genesys/resources/dynamodb/my-table.toml

Next steps:
  - Review the configuration: cat ~/.genesys/resources/dynamodb/my-table.toml
  - Preview deployment: genesys execute ~/.genesys/resources/dynamodb/my-table.toml
  - Deploy the table: genesys execute ~/.genesys/resources/dynamodb/my-table.toml --apply
  - Delete when done: genesys execute ~/.genesys/resources/dynamodb/my-table.toml --delete
```

## TUI Dashboard

Access DynamoDB management through the TUI dashboard:

```bash
genesys tui
```

Select **DynamoDB Tables** (option 5) from the main menu.

### Table List View

The table list displays all DynamoDB tables in your region with:

| Column | Description |
|--------|-------------|
| Table Name | Name of the DynamoDB table |
| Status | ACTIVE, CREATING, UPDATING, DELETING |
| Billing Mode | On-Demand or Provisioned |
| Items | Number of items in the table |
| Size | Total storage size |
| Region | AWS region |

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Up/Down | Navigate tables |
| Enter | View table details |
| b | Browse table items |
| d | Delete table (with confirmation) |
| r | Refresh table list |
| ESC | Back to dashboard |
| q | Quit application |

### Table Detail View

Press Enter on a table to view detailed information:

- Table name, status, and ARN
- Billing mode and capacity settings
- Key schema (partition and sort keys)
- Provisioned throughput (if applicable)
- Stream configuration
- Global Secondary Indexes

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Up/Down | Scroll content |
| b | Browse table items |
| ESC | Back to table list |
| q | Quit application |

### Item Browser

Press `b` to browse items in a table:

- Displays items in a paginated table format
- Shows key attributes and up to 8 additional columns
- Supports pagination for large tables

#### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Up/Down | Navigate items |
| Enter | View item detail |
| n | Next page |
| r | Refresh items |
| ESC | Back to table list |
| q | Quit application |

### Item Detail View

Press Enter on an item to view its full attributes in a formatted display.

## Configuration Examples

### Simple Key-Value Table

**Use Case**: Session storage, caching

```toml
provider = "aws"
region = "us-east-1"

[[resources.dynamodb]]
name = "sessions"
billing_mode = "PAY_PER_REQUEST"
hash_key_name = "session_id"
hash_key_type = "S"
enable_streams = false
enable_ttl = true
ttl_attribute_name = "expires_at"
```

### Table with Sort Key

**Use Case**: User orders, time-series data

```toml
provider = "aws"
region = "us-east-1"

[[resources.dynamodb]]
name = "user-orders"
billing_mode = "PAY_PER_REQUEST"
hash_key_name = "user_id"
hash_key_type = "S"
range_key_name = "order_date"
range_key_type = "S"
enable_streams = true
stream_view_type = "NEW_AND_OLD_IMAGES"
enable_ttl = false

[resources.dynamodb.tags]
Environment = "production"
Application = "ecommerce"
```

### Provisioned Capacity Table

**Use Case**: Predictable workloads with cost optimization

```toml
provider = "aws"
region = "us-east-1"

[[resources.dynamodb]]
name = "products"
billing_mode = "PROVISIONED"
hash_key_name = "product_id"
hash_key_type = "S"
read_capacity_units = 10
write_capacity_units = 5
enable_streams = false
enable_ttl = false

[resources.dynamodb.tags]
Environment = "production"
Team = "catalog"
```

## Best Practices

### Key Design

1. **Partition Key Selection**: Choose a high-cardinality attribute for even distribution
2. **Sort Key Usage**: Use for range queries and hierarchical data
3. **Avoid Hot Partitions**: Distribute access patterns across partition keys

### Capacity Planning

1. **Start with On-Demand**: Use for unpredictable or new workloads
2. **Switch to Provisioned**: Once patterns are understood, for cost optimization
3. **Monitor Capacity**: Watch consumed capacity vs. provisioned

### Cost Optimization

1. **On-Demand Mode**: Best for sporadic or unpredictable traffic
2. **Provisioned Mode**: Best for steady, predictable workloads
3. **Reserved Capacity**: Consider for long-running production tables
4. **TTL**: Use to automatically delete expired data

### Security

1. **IAM Policies**: Use fine-grained access control
2. **Encryption**: DynamoDB encrypts all data at rest by default
3. **VPC Endpoints**: Use for private access from VPCs

## Troubleshooting

### Common Issues

**"Table not found"**
- Verify the table name and region
- Check IAM permissions: dynamodb:DescribeTable

**"Provisioned throughput exceeded"**
- Increase capacity units or switch to on-demand mode
- Check for hot partitions

**"Validation error"**
- Verify table name format (3-255 chars, alphanumeric)
- Check attribute types match key schema

**"Access denied"**
- Verify IAM permissions for DynamoDB operations
- Check resource-based policies

### Required IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:ListTables",
        "dynamodb:DescribeTable",
        "dynamodb:CreateTable",
        "dynamodb:DeleteTable",
        "dynamodb:UpdateTable",
        "dynamodb:Scan",
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem",
        "dynamodb:UpdateTimeToLive",
        "dynamodb:DescribeTimeToLive"
      ],
      "Resource": "*"
    }
  ]
}
```

## Integration with Other Services

### With Lambda Functions

Create event-driven architectures with DynamoDB Streams:

1. Enable streams on your table with NEW_AND_OLD_IMAGES
2. Create a Lambda function triggered by the stream
3. Process item changes in real-time

### With S3

Export DynamoDB data to S3 for analytics:

1. Use DynamoDB export to S3 feature
2. Query with Athena or process with other analytics tools

## API Reference

### DynamoDB Service Methods

| Method | Description |
|--------|-------------|
| ListTables | List all tables in the region |
| DescribeTable | Get detailed table information |
| CreateTable | Create a new table |
| DeleteTable | Delete a table |
| UpdateTable | Update table settings |
| ScanTable | Scan items with pagination |
| GetItem | Get a single item by key |
| PutItem | Create or update an item |
| DeleteItem | Delete an item by key |
| UpdateTTL | Configure TTL settings |
| DescribeTTL | Get TTL configuration |

---

**Next Steps**: Try the [Getting Started Guide](getting-started.md) for your first DynamoDB table, or explore [Interactive Workflow Guide](interactive-workflow.md) for advanced usage patterns.
