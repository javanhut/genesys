# S3 Cross-Region Bucket Copy

This feature enables copying S3 objects from one bucket to another bucket in a different AWS region directly from the Genesys TUI.

## Overview

Cross-region S3 copy is useful for:

- **Disaster Recovery**: Replicate data to a different geographic region
- **Compliance**: Meet data residency requirements by storing copies in specific regions
- **Performance**: Position data closer to users in different regions
- **Migration**: Move data between regions during infrastructure changes

## TUI Usage

### Accessing the Copy Feature

1. Launch the Genesys TUI dashboard:
   ```bash
   genesys tui
   ```

2. Navigate to **S3 Buckets** from the dashboard

3. Select a bucket and press **Enter** to browse its contents

4. Press **c** to open the cross-region copy interface

### Copy Interface

The cross-region copy view displays a split-pane interface:

```
+---------------------------+---------------------------+
| Source: my-bucket (us-e1) | Destination               |
+---------------------------+---------------------------+
| [x] file1.txt     10 KB   | Select Region:            |
| [x] file2.txt     25 KB   | > us-west-2 (Oregon)      |
| [x] folder/data   1.5 MB  |   eu-west-1 (Ireland)     |
| [ ] old-backup    500 MB  |   ap-northeast-1 (Tokyo)  |
+---------------------------+---------------------------+
| Progress                                              |
| Press 'c' to start copying...                         |
+-------------------------------------------------------+
| Focus: Source | Selected: 3 objects (1.5 MB) | ...    |
+-------------------------------------------------------+
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between source list, region selector, and bucket input |
| `Space` | Toggle selection of current object |
| `a` | Select all objects |
| `n` | Deselect all objects |
| `c` | Start the copy operation |
| `ESC` | Go back / Cancel |
| `q` | Quit the TUI |

### Copy Process

1. **Select Objects**: Use Space to toggle individual objects, or `a`/`n` for all/none

2. **Choose Destination Region**: Navigate the region list and select target region

3. **Set Bucket Name**: The destination bucket name auto-populates but can be edited

4. **Start Copy**: Press `c` to begin the copy operation

5. **Monitor Progress**: Real-time progress shows:
   - Objects copied / total
   - Data transferred
   - Current object being copied
   - Transfer speed
   - Elapsed time

## Programmatic API

The cross-region copy functionality is also available programmatically through the provider interface.

### Copy Single Object

```go
err := provider.Storage().CopyObjectCrossRegion(
    ctx,
    "source-bucket",
    "path/to/object.txt",
    "us-west-2",           // destination region
    "dest-bucket",
    "new/path/object.txt", // destination key
)
```

### Copy Entire Bucket

```go
progress := make(chan *provider.CrossRegionCopyProgress, 100)

go func() {
    for p := range progress {
        fmt.Printf("Progress: %.1f%% (%d/%d objects)\n", 
            p.PercentComplete, p.CopiedObjects, p.TotalObjects)
    }
}()

err := provider.Storage().CopyBucketCrossRegion(
    ctx,
    "source-bucket",
    "eu-west-1",      // destination region
    "dest-bucket",
    "",               // prefix (empty for all objects)
    progress,
)
```

### Progress Tracking

The `CrossRegionCopyProgress` struct provides detailed progress information:

```go
type CrossRegionCopyProgress struct {
    SourceBucket    string
    SourceRegion    string
    DestBucket      string
    DestRegion      string
    TotalObjects    int64
    CopiedObjects   int64
    FailedObjects   int64
    TotalBytes      int64
    CopiedBytes     int64
    CurrentObject   string
    PercentComplete float64
    BytesPerSecond  float64
    StartTime       time.Time
    Status          string   // "preparing", "copying", "complete", "failed"
    Error           error
    FailedKeys      []string
}
```

## Technical Details

### How It Works

1. **Source Listing**: Objects are listed from the source bucket
2. **Destination Setup**: The destination bucket is created if it doesn't exist
3. **Object Copy**: Each object is copied using S3's server-side copy
4. **Large Files**: Objects larger than 5GB use multipart copy

### Server-Side Copy

Cross-region copy uses S3's server-side copy mechanism via the `x-amz-copy-source` header. This means:

- Data is transferred directly between AWS data centers
- No data passes through your local machine
- Copy speed depends on AWS internal network, not your internet connection

### Large Object Handling

For objects larger than 5GB, the copy uses multipart upload:

1. Initiate multipart upload in destination region
2. Copy object in 100MB parts using `UploadPartCopy`
3. Complete multipart upload

### Concurrent Copies

Bulk bucket copies use a worker pool of 10 concurrent copy operations for optimal performance.

## Supported Regions

The following AWS regions are supported:

| Region Code | Location |
|-------------|----------|
| us-east-1 | N. Virginia |
| us-east-2 | Ohio |
| us-west-1 | N. California |
| us-west-2 | Oregon |
| eu-west-1 | Ireland |
| eu-west-2 | London |
| eu-west-3 | Paris |
| eu-central-1 | Frankfurt |
| eu-north-1 | Stockholm |
| ap-northeast-1 | Tokyo |
| ap-northeast-2 | Seoul |
| ap-southeast-1 | Singapore |
| ap-southeast-2 | Sydney |
| ap-south-1 | Mumbai |
| sa-east-1 | Sao Paulo |
| ca-central-1 | Canada |
| ... | (and more) |

## Cost Considerations

Cross-region S3 copy incurs the following AWS costs:

1. **Data Transfer**: Standard cross-region data transfer rates apply
2. **PUT Requests**: One PUT request per object in the destination region
3. **GET Requests**: One GET request per object from the source region
4. **Storage**: Standard storage costs in the destination region

For large data transfers, consider:
- Using S3 Transfer Acceleration for faster transfers
- Scheduling copies during off-peak hours
- Using S3 Replication for ongoing synchronization needs

## Error Handling

The copy operation handles errors gracefully:

- **Partial Failures**: If some objects fail to copy, the operation continues with remaining objects
- **Failed Keys List**: All failed object keys are tracked and reported
- **Retry**: Failed objects can be re-copied by running the operation again

## Limitations

- Maximum single object size: 5TB (S3 limit)
- Objects larger than 5GB require multipart copy
- Source and destination must use the same AWS account
- Versioned objects: Only the latest version is copied

## See Also

- [S3 Management](s3-management.md) - General S3 operations
- [S3 Workflow](s3-workflow.md) - S3 workflow examples
- [TUI Guide](tui.md) - TUI usage guide
