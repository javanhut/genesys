# S3 Bucket Management with Genesys

Manage S3 buckets and objects without AWS CLI. Upload, download, list, and sync files using simple commands.

## Overview

Genesys provides S3 file operations similar to `scp` or `rsync`:
- **List objects** - Browse bucket contents
- **Upload files** - Upload single files or directories
- **Download files** - Download files with progress tracking
- **Delete objects** - Remove files from buckets
- **Sync directories** - Bi-directional sync like rsync
- **Progress tracking** - Real-time upload/download progress

All operations use direct S3 API calls - no AWS CLI required.

## Quick Start

### List Bucket Contents

```bash
# List all objects
genesys manage s3 my-bucket ls

# List with prefix (folder)
genesys manage s3 my-bucket ls data/
genesys manage s3 my-bucket ls logs/2024-11/
```

Example output:
```
Listing objects in s3://my-bucket/

  PREFIX    data/
  PREFIX    logs/
  FILE      config.json                          1.2 KB    2024-11-20 10:30
  FILE      README.md                            856 B     2024-11-15 14:22

Total: 2 prefixes, 2 files
```

### Download Files

```bash
# Download single file
genesys manage s3 my-bucket get config.json ./config.json

# Download to different name
genesys manage s3 my-bucket get data/file.txt ./local-copy.txt
```

Example output:
```
Downloading s3://my-bucket/config.json → ./config.json
Progress: 100.0% (1.2 KB/1.2 KB)
✓ Download complete: 1.2 KB (0.45 MB/s)
```

### Upload Files

```bash
# Upload single file
genesys manage s3 my-bucket put ./report.pdf reports/2024-11.pdf

# Upload with auto-detection of content type
genesys manage s3 my-bucket put ./image.png images/logo.png
```

Example output:
```
Uploading ./report.pdf → s3://my-bucket/reports/2024-11.pdf
Progress: 100.0% (5.2 MB/5.2 MB)
✓ Upload complete: 5.2 MB (1.8 MB/s)
```

### Delete Objects

```bash
# Delete with confirmation
genesys manage s3 my-bucket rm old-file.txt
```

Example output:
```
Delete s3://my-bucket/old-file.txt? [y/N]: y
Deleting s3://my-bucket/old-file.txt
✓ Deleted successfully
```

## Directory Sync

Sync entire directories between local and S3, similar to `rsync`.

### Upload Directory

```bash
# Sync local directory to S3
genesys manage s3 my-bucket sync ./backups/ /backups/2024-11-24/
```

This will:
1. Scan local directory
2. Show number of files to upload
3. Ask for confirmation
4. Upload all files preserving directory structure

Example output:
```
Syncing ./backups/ ↔ s3://my-bucket/backups/2024-11-24/ (direction: upload)
Analyzing changes...
Found 15 local files to sync
Proceed? [Y/n]: y
Uploading: ./backups/db.sql → s3://my-bucket/backups/2024-11-24/db.sql
  Complete
Uploading: ./backups/files.tar.gz → s3://my-bucket/backups/2024-11-24/files.tar.gz
  Complete
...
✓ Sync complete
```

### Download Directory

```bash
# Sync S3 prefix to local directory
genesys manage s3 my-bucket sync ./local-backup/ /backups/2024-11-24/ download
```

### Sync Directions

- `upload` or `up` - Local to S3
- `download` or `down` - S3 to local

## Advanced Usage

### Large File Uploads

Genesys automatically handles large files:
- Files < 5MB: Single upload
- Files >= 5MB: Multipart upload (when implemented)
- Progress tracking for all sizes

### Content Type Detection

Content types are automatically detected:
- `.jpg`, `.png` → `image/jpeg`, `image/png`
- `.pdf` → `application/pdf`
- `.txt` → `text/plain`
- `.html` → `text/html`
- `.json` → `application/json`

### Working with Prefixes (Folders)

S3 doesn't have real folders, but uses prefixes:

```bash
# List "folder" contents
genesys manage s3 my-bucket ls data/2024/

# Upload to "folder"
genesys manage s3 my-bucket put ./file.txt data/2024/file.txt

# Sync to "folder"
genesys manage s3 my-bucket sync ./local/ data/2024/
```

## Common Workflows

### Backup Workflow

```bash
#!/bin/bash
# Daily backup script

BUCKET="my-backups"
DATE=$(date +%Y-%m-%d)

# Create local backup
tar -czf /tmp/backup-${DATE}.tar.gz /var/www/html

# Upload to S3
genesys manage s3 $BUCKET put /tmp/backup-${DATE}.tar.gz backups/${DATE}.tar.gz

# Clean up
rm /tmp/backup-${DATE}.tar.gz

echo "Backup complete: s3://$BUCKET/backups/${DATE}.tar.gz"
```

### Restore Workflow

```bash
#!/bin/bash
# Restore from S3 backup

BUCKET="my-backups"
RESTORE_DATE="2024-11-24"

# Download backup
genesys manage s3 $BUCKET get backups/${RESTORE_DATE}.tar.gz /tmp/restore.tar.gz

# Extract
tar -xzf /tmp/restore.tar.gz -C /var/www/

echo "Restore complete from ${RESTORE_DATE}"
```

### Website Deployment

```bash
#!/bin/bash
# Deploy static website to S3

BUILD_DIR="./dist"
BUCKET="my-website"

# Build site
npm run build

# Sync to S3
genesys manage s3 $BUCKET sync $BUILD_DIR / upload

echo "Website deployed to s3://$BUCKET/"
```

### Download Latest Files

```bash
#!/bin/bash
# Download latest data files

BUCKET="data-exports"
LOCAL_DIR="./data"

# List and download latest
genesys manage s3 $BUCKET ls exports/latest/
genesys manage s3 $BUCKET sync $LOCAL_DIR exports/latest/ download

echo "Downloaded latest data to $LOCAL_DIR"
```

## Performance Tips

1. **Use sync for multiple files**: More efficient than individual uploads/downloads
2. **Organize with prefixes**: Use meaningful prefix structure (dates, categories)
3. **Monitor progress**: Large operations show real-time progress
4. **Compression**: Compress before upload for faster transfers
5. **Parallel operations**: Use shell backgrounding for multiple buckets

Example parallel upload:
```bash
genesys manage s3 bucket1 sync ./data1/ /data/ &
genesys manage s3 bucket2 sync ./data2/ /data/ &
wait
echo "All uploads complete"
```

## Cost Optimization

### S3 API Costs

- **PUT requests**: $0.005 per 1,000 requests
- **GET requests**: $0.0004 per 1,000 requests
- **Data transfer out**: $0.09 per GB (first 10 TB/month)

### Best Practices

1. **Batch operations**: Use sync instead of many individual uploads
2. **Compress files**: Reduce transfer size and cost
3. **Use prefixes wisely**: Organize files to minimize listing operations
4. **Monitor with genesys**: `genesys monitor s3 my-bucket` to track usage

## Troubleshooting

### "Failed to upload file"

**Solutions**:
1. Check file exists: `ls -la /path/to/file`
2. Check permissions: Ensure read access
3. Check bucket exists: `genesys list resources --service storage`
4. Verify IAM permissions for S3

### "Permission denied"

**IAM Permissions Required**:
```json
{
  "Effect": "Allow",
  "Action": [
    "s3:PutObject",
    "s3:GetObject",
    "s3:DeleteObject",
    "s3:ListBucket"
  ],
  "Resource": [
    "arn:aws:s3:::my-bucket",
    "arn:aws:s3:::my-bucket/*"
  ]
}
```

### "Bucket not found"

**Solutions**:
1. Check bucket name (case-sensitive)
2. Verify region: `genesys config show aws`
3. List available buckets: `genesys list resources --service storage`

### Slow transfers

**Possible causes**:
1. Large file size
2. Network bandwidth limitations
3. Geographic distance from S3 region

**Solutions**:
- Use compression: `tar -czf archive.tar.gz files/`
- Choose closer region
- Use sync for batch operations

## Comparison with AWS CLI

| Operation | Genesys | AWS CLI |
|-----------|---------|---------|
| List | `genesys manage s3 BUCKET ls` | `aws s3 ls s3://BUCKET` |
| Upload | `genesys manage s3 BUCKET put FILE KEY` | `aws s3 cp FILE s3://BUCKET/KEY` |
| Download | `genesys manage s3 BUCKET get KEY FILE` | `aws s3 cp s3://BUCKET/KEY FILE` |
| Sync | `genesys manage s3 BUCKET sync DIR PREFIX` | `aws s3 sync DIR s3://BUCKET/PREFIX` |
| Delete | `genesys manage s3 BUCKET rm KEY` | `aws s3 rm s3://BUCKET/KEY` |

**Advantages of Genesys**:
- No separate CLI installation
- Integrated with resource management
- Progress tracking built-in
- Unified command structure
- Automatic content-type detection

## Related Commands

- `genesys inspect s3 BUCKET` - Inspect bucket configuration
- `genesys monitor s3 BUCKET` - Monitor bucket metrics
- `genesys list resources --service storage` - List all buckets

For more information, see:
- [Monitoring Guide](monitoring.md)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
