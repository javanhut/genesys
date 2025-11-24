# Resource Monitoring with Genesys

Monitor your AWS resources without requiring AWS CLI installation. Genesys provides direct CloudWatch integration for real-time metrics and logs.

## Overview

The monitoring features include:
- **Real-time metrics** from CloudWatch
- **Resource health status** checks
- **Lambda log streaming** and viewing
- **Watch mode** for continuous monitoring
- **Multiple output formats** (human-readable and JSON)

## Quick Start

### Monitor All Resources

```bash
genesys monitor resources
```

Example output:
```
Monitoring Resources
====================

Compute (2 resources):
  ✓ web-server-prod (i-1234567890abcdef0)
    cpu_utilization: 45.20
  ✓ api-server-prod (i-0987654321fedcba0)
    cpu_utilization: 78.50

Storage (1 resources):
  ✓ my-app-data (my-app-data-bucket)

Serverless (2 resources):
  ✓ my-api-function
  ✓ background-worker
```

### Watch Mode

Monitor resources with automatic refresh:

```bash
genesys monitor resources --watch
```

Updates every 30 seconds. Press Ctrl+C to stop.

## EC2 Instance Monitoring

Monitor CPU, network, and disk metrics for EC2 instances.

### Basic Monitoring

```bash
genesys monitor ec2 i-1234567890abcdef0
```

### Custom Time Periods

```bash
genesys monitor ec2 i-1234567890abcdef0 --period 24h
```

Supported periods:
- `5m`, `15m`, `30m` - Minutes
- `1h`, `3h`, `6h`, `12h` - Hours
- `24h`, `1d` - Day
- `7d`, `1w` - Week
- `30d`, `1M` - Month

### Metrics Available

- **CPU Utilization** - Percentage of CPU used
- **Network In** - Bytes received
- **Network Out** - Bytes sent
- **Disk Read Ops** - Disk read operations
- **Disk Write Ops** - Disk write operations
- **Disk Read Bytes** - Bytes read from disk
- **Disk Write Bytes** - Bytes written to disk
- **Status Checks** - Instance and system status

Example output:
```
Monitoring EC2 Instance: i-1234567890abcdef0
Period: 24h
=================================

CPU Utilization:
  10:00: 45.23 %
  11:00: 52.10 %
  12:00: 48.75 %
  ...
  Average: 47.36 %
  Min: 23.45 % | Max: 89.12 %

Network In:
  10:00: 1024000.00 bytes
  11:00: 1536000.00 bytes
  ...
  Average: 1280000.00 bytes
```

## S3 Bucket Monitoring

Monitor S3 bucket size, object count, and request metrics.

### Basic Monitoring

```bash
genesys monitor s3 my-bucket
```

### Metrics Available

- **Bucket Size** - Total size in bytes
- **Object Count** - Number of objects
- **All Requests** - Total API requests
- **GET Requests** - Download requests
- **PUT Requests** - Upload requests

Example output:
```
Monitoring S3 Bucket: my-bucket
Period: 1h
=============================

Bucket Size: 45.6 GB
Object Count: 1234

Total Requests:
  10:00: 125.00 requests
  10:15: 143.00 requests
  ...
  Average: 134.50 requests
```

## Lambda Function Monitoring

Monitor Lambda invocations, duration, errors, and view logs.

### Basic Monitoring

```bash
genesys monitor lambda my-function
```

### View Recent Logs

```bash
genesys monitor lambda my-function --logs
```

### Stream Logs in Real-Time

```bash
genesys monitor lambda my-function --tail
```

Press Ctrl+C to stop streaming.

### Metrics Available

- **Invocations** - Number of function invocations
- **Duration** - Execution time in milliseconds
- **Errors** - Number of errors
- **Throttles** - Number of throttled invocations
- **Concurrent Executions** - Concurrent execution count

Example output:
```
Monitoring Lambda Function: my-function
Period: 1h
=====================================

Invocations:
  10:00: 125.00 count
  10:15: 143.00 count
  ...
  Average: 134.50 count

Duration:
  10:00: 145.50 ms
  10:15: 152.30 ms
  ...
  Average: 148.90 ms

Errors:
  Average: 0.00 count

Recent Logs (last 50 events):
==============================
[10:30:15] START RequestId: abc123-def456
[10:30:15] INFO: Processing request for user: user@example.com
[10:30:15] INFO: Database query completed in 45ms
[10:30:16] END RequestId: abc123-def456
[10:30:16] REPORT Duration: 145.23 ms  Memory: 128/512 MB
```

## JSON Output

Get machine-readable output for scripting:

```bash
genesys monitor ec2 i-1234567890abcdef0 --output json
```

Example output:
```json
{
  "instance_id": "i-1234567890abcdef0",
  "cpu_utilization": [
    {
      "timestamp": "2024-11-24T10:00:00Z",
      "value": 45.23,
      "unit": "Percent",
      "statistic": "Average"
    }
  ],
  "network_in": [...],
  "network_out": [...]
}
```

## Filtering by Service Type

Monitor specific service types:

```bash
genesys monitor resources --service compute    # Only EC2
genesys monitor resources --service storage    # Only S3
genesys monitor resources --service serverless # Only Lambda
```

## Best Practices

1. **Use watch mode for ops**: Monitor production resources continuously
2. **Set appropriate periods**: Use shorter periods (5m, 15m) for real-time monitoring
3. **JSON for automation**: Use `--output json` in scripts and CI/CD
4. **Tail logs during debugging**: Use `--tail` to debug Lambda functions
5. **Regular health checks**: Run `genesys monitor resources` regularly

## Cost Implications

CloudWatch API calls have minimal cost:
- **GetMetricStatistics**: $0.01 per 1,000 requests
- **FilterLogEvents**: $0.01 per 1,000 requests
- **First 1 million requests per month are free**

Typical usage:
- Monitoring 10 resources once = ~30 API calls
- Watch mode for 1 hour (120 refreshes) = ~3,600 API calls = $0.036

## Troubleshooting

### "Failed to get metrics"

**Problem**: Cannot retrieve CloudWatch metrics

**Solutions**:
1. Check AWS credentials: `genesys config show aws`
2. Verify IAM permissions for CloudWatch
3. Ensure resource exists: `genesys list resources`

### "No data available"

**Problem**: Metrics show no data

**Possible causes**:
1. Resource is newly created (CloudWatch has 5-15 minute delay)
2. Resource is not generating metrics (stopped instance, unused bucket)
3. CloudWatch monitoring not enabled (basic vs detailed monitoring)

**Solutions**:
- Wait 15 minutes for new resources
- Check resource is active and generating traffic
- Enable detailed monitoring for EC2 instances

### "Permission denied"

**Problem**: Access denied when fetching metrics

**Solution**:
Add CloudWatch permissions to your IAM user/role:
```json
{
  "Effect": "Allow",
  "Action": [
    "cloudwatch:GetMetricStatistics",
    "cloudwatch:ListMetrics",
    "cloudwatch:DescribeAlarms",
    "logs:FilterLogEvents",
    "logs:GetLogEvents",
    "logs:DescribeLogStreams"
  ],
  "Resource": "*"
}
```

## Examples

### Production Monitoring Workflow

```bash
# Morning health check
genesys monitor resources

# Check specific resource if issues found
genesys monitor ec2 i-problematic-instance --period 24h

# View Lambda logs for debugging
genesys monitor lambda failing-function --logs

# Continuous monitoring during deployment
genesys monitor resources --watch
```

### Automation Script

```bash
#!/bin/bash
# Check resource health and alert if issues

HEALTH=$(genesys monitor resources --output json)

if echo "$HEALTH" | jq -e '.[] | select(.status == "unhealthy")' > /dev/null; then
  echo "ALERT: Unhealthy resources detected!"
  echo "$HEALTH" | jq '.[] | select(.status == "unhealthy")'
  # Send alert notification
fi
```

## Related Commands

- `genesys inspect` - Detailed resource inspection
- `genesys manage` - Interactive resource management
- `genesys list` - List all resources

For more information, see:
- [Management Guide](management.md)
- [AWS Documentation](https://docs.aws.amazon.com/cloudwatch/)
