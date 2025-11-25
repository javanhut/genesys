# Genesys Interactive TUI

The Genesys Terminal User Interface (TUI) provides an interactive way to browse and manage AWS cloud resources without requiring the AWS CLI.

## Features

- Interactive dashboard for resource overview
- Browse EC2 instances with real-time status
- Navigate S3 buckets and files
- View Lambda functions
- Keyboard-driven navigation
- No mouse required
- Direct AWS API integration

## Usage

### Launch Main Dashboard

```bash
genesys tui
```

This launches the main dashboard where you can navigate to different resource types.

### Launch from Management Commands

You can launch the TUI directly for specific resources:

```bash
# S3 bucket browser
genesys manage s3 my-bucket --tui

# Launch management TUI
genesys manage --tui
```

### Launch from Monitoring Commands

```bash
# Launch monitoring dashboard
genesys monitor --tui
```

## Navigation

### Global Keyboard Shortcuts

- `↑/↓` or `j/k` - Navigate lists
- `Enter` - Select item / Open
- `ESC` - Go back / Cancel
- `q` - Quit application
- `r` - Refresh current view
- `?` - Show help

### Dashboard

The main dashboard provides quick access to:

1. **EC2 Instances** (press 2) - View all compute instances
   - Press Enter on an instance to see details
   - Press 'm' to view metrics
2. **S3 Buckets** (press 3) - Browse storage buckets
   - Press Enter to browse bucket contents
   - Press 'd' to download files
3. **Lambda Functions** (press 4) - Manage serverless functions
   - Press Enter to view details
   - Press 'i' to invoke
   - Press 'l' to view logs
4. **Quit** (press q) - Exit the TUI

### Resource Lists

When viewing resource lists (EC2, S3, Lambda):

- Use arrow keys to navigate
- Press `Enter` to view details
- Press `r` to refresh the list
- Press `ESC` to return to dashboard

**EC2 Specific:**
- Press `m` to jump directly to metrics

**Lambda Specific:**
- Press `i` to invoke function
- Press `l` to view logs

### S3 Bucket Browser

When browsing S3 buckets:

- `Enter` - Open folder / Select file
- `Backspace` - Navigate to parent folder
- `d` - Download selected file (shows progress)
- `u` - Enter upload mode (split-pane file browser)
- `r` - Refresh current directory
- `ESC` - Return to bucket list

Files are downloaded to the current directory where you launched the TUI.

### S3 Upload Mode (Split-Pane Browser)

Press `u` in the S3 browser to enter upload mode. This displays a dual-pane view:
- **Left pane (green border)**: Local filesystem browser
- **Right pane (blue border)**: S3 bucket browser

**Upload Mode Controls:**

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between local and S3 panes |
| `Enter` | Navigate into folder (if directory) or upload file (if file in local pane) |
| `Backspace` | Go to parent directory |
| `h` | Toggle hidden files (local pane) |
| `~` | Jump to home directory (local pane) |
| `r` | Refresh current pane |
| `ESC` | Exit upload mode, return to S3 browser |

**Uploading Files:**
1. Navigate the local pane to find your file
2. Press `Enter` on a file to upload it to the current S3 location
3. A progress modal shows upload status
4. The S3 pane refreshes automatically after successful upload

The uploaded file will be placed in the current S3 prefix (folder) shown in the right pane.

## Screens

### Dashboard Screen

```
┌────────────────────────────────────────────────────────────┐
│ Genesys TUI - AWS (us-east-2)              Press ? for help│
├────────────────────────────────────────────────────────────┤
│  Dashboard                                                 │
│                                                            │
│  > EC2 Instances                                           │
│    Manage compute instances                                │
│                                                            │
│  > S3 Buckets                                             │
│    Browse storage buckets                                  │
│                                                            │
│  > Lambda Functions                                        │
│    Manage serverless functions                             │
│                                                            │
│  > Quit                                                    │
│    Exit the TUI                                            │
│                                                            │
├────────────────────────────────────────────────────────────┤
│ ↑↓: Navigate | Enter: Select | r: Refresh | q: Quit       │
└────────────────────────────────────────────────────────────┘
```

### EC2 Instances List

```
┌────────────────────────────────────────────────────────────┐
│ EC2 Instances                                              │
├────────────────────────────────────────────────────────────┤
│ Instance ID        Name        State    Type      IP      │
│────────────────────────────────────────────────────────────│
│ i-1234567890abc   web-server  running  t2.micro  1.2.3.4 │
│ i-0987654321def   db-server   stopped  t3.small  5.6.7.8 │
│                                                            │
├────────────────────────────────────────────────────────────┤
│ ↑↓: Navigate | m: Metrics | r: Refresh | ESC: Back        │
└────────────────────────────────────────────────────────────┘
```

### S3 Browser

```
┌────────────────────────────────────────────────────────────┐
│ S3: my-bucket / images /                                   │
├────────────────────────────────────────────────────────────┤
│ Name               Size      Modified                     │
│────────────────────────────────────────────────────────────│
│ 📁 thumbnails/     -         2025-11-20 10:30            │
│ 📄 logo.png        45.2 KB   2025-11-20 09:15            │
│ 📄 hero.jpg        2.1 MB    2025-11-19 14:22            │
│                                                            │
├────────────────────────────────────────────────────────────┤
│ ↑↓: Navigate | Enter: Open | Backspace: Up | ESC: Back    │
└────────────────────────────────────────────────────────────┘
```

## Requirements

- Valid AWS credentials configured in `~/.genesys/aws.json`
- Network access to AWS APIs
- Terminal with Unicode support (for icons)

## Configuration

The TUI uses the same provider configuration as other Genesys commands. Ensure you have configured your AWS credentials:

```bash
genesys config setup
```

## Features

### Phase 1-4 Complete

All core features are now implemented:

- **EC2 Detailed Views**: Full instance information with metrics
- **S3 File Operations**: Download and upload files with progress tracking
- **S3 Upload Mode**: Dual-pane local/S3 file browser for easy uploads
- **Lambda Invocation**: Invoke functions with custom payloads
- **Metrics Visualization**: Real-time CloudWatch metrics
- **Log Viewing**: View recent Lambda logs

## Troubleshooting

### TUI won't launch

Make sure your AWS credentials are configured:
```bash
cat ~/.genesys/aws.json
```

### Resources not showing

- Check your AWS region is correct
- Verify you have permissions to list resources
- Try refreshing with `r` key

### Display issues

- Ensure your terminal supports Unicode
- Try resizing your terminal window
- Check terminal has at least 80x24 character dimensions

## Implemented Features

### Phases 1-5 (Complete)
- Dashboard with resource navigation
- EC2 instances list with status
- S3 bucket browser with folder navigation
- Lambda functions list
- Detailed resource views (EC2, S3, Lambda)
- Real-time metrics from CloudWatch
- File downloads with progress
- File uploads to S3 with dual-pane browser
- Lambda function invocation
- Log viewing

### Future Enhancements (Phase 6)
- Live log streaming with auto-scroll
- EC2 instance start/stop
- Custom color themes
- Resource filtering and search
- Bulk operations

## Detailed Features

### EC2 Instance Management

1. **View All Instances**
   - Lists all EC2 instances in your account
   - Shows: ID, Name, State, Type, IP Address
   - Color-coded status (green=running, red=stopped)

2. **Instance Details**
   - Full instance information
   - Public and private IP addresses
   - Creation timestamp
   - All instance tags
   - Recent CPU metrics

3. **Metrics View**
   - CPU utilization (current, average, min, max)
   - Network In/Out statistics
   - 6-hour time window
   - Direct CloudWatch integration

### S3 Bucket Management

1. **Browse Buckets**
   - Lists all S3 buckets
   - Shows region and creation date
   - Enter to browse contents

2. **File Navigation**
   - Folder icons (📁) and file icons (📄)
   - Hierarchical navigation
   - Breadcrumb in title shows current path
   - File sizes and modification dates

3. **Download Files**
   - Press 'd' on any file
   - Progress modal shows percentage
   - Files saved to current directory
   - Success/failure notifications

### Lambda Function Management

1. **View All Functions**
   - Lists all Lambda functions
   - Shows runtime, memory, timeout
   - Quick invoke from list (press 'i')

2. **Function Details**
   - Complete configuration display
   - Environment variables
   - Handler information
   - Creation timestamp

3. **Invoke Function**
   - Custom JSON payload input
   - Progress indicator
   - Result display in modal
   - Error handling

4. **View Logs**
   - Last 100 log events
   - Formatted timestamps
   - Scrollable view
   - Automatic CloudWatch Logs integration

## Examples

### Basic Usage

```bash
# Launch dashboard
genesys tui

# Navigate to EC2 instances (press 2)
# View instance list
# Press ESC to go back
# Navigate to S3 buckets (press 3)
# Press Enter on a bucket to browse files
# Press q to quit
```

### Browse S3 Bucket

```bash
# Launch directly to S3 browser
genesys manage s3 my-production-bucket --tui

# Navigate folders with Enter
# Go up with Backspace
# Refresh with r
# Exit with q
```

## Getting Help

- Press `?` in any view to see keyboard shortcuts
- Run `genesys tui --help` for command-line help
- See main documentation: `docs/README.md`

## Related Commands

- `genesys manage` - CLI management operations
- `genesys monitor` - CLI monitoring operations
- `genesys inspect` - Detailed resource inspection
- `genesys list` - List all resources

## Quick Reference Card

### All Screens
| Key | Action |
|-----|--------|
| `q` | Quit application |
| `ESC` | Go back to previous screen |
| `r` | Refresh current view |
| `↑/↓` | Navigate up/down in lists |

### Dashboard
| Key | Action |
|-----|--------|
| `2` | Go to EC2 Instances |
| `3` | Go to S3 Buckets |
| `4` | Go to Lambda Functions |
| `Enter` | Select menu item |

### EC2 List
| Key | Action |
|-----|--------|
| `Enter` | View instance details |
| `m` | View metrics directly |
| `r` | Refresh instance list |

### EC2 Detail
| Key | Action |
|-----|--------|
| `m` | View detailed metrics |
| `ESC` | Back to instance list |

### S3 Browser
| Key | Action |
|-----|--------|
| `Enter` | Open folder or select file |
| `Backspace` | Go up to parent folder |
| `d` | Download selected file |
| `u` | Enter upload mode (dual-pane browser) |
| `r` | Refresh current listing |

### S3 Upload Mode
| Key | Action |
|-----|--------|
| `Tab` | Switch between local and S3 panes |
| `Enter` | Navigate folder / Upload selected file |
| `Backspace` | Go to parent directory |
| `h` | Toggle hidden files (local pane) |
| `~` | Go to home directory (local pane) |
| `r` | Refresh current pane |
| `ESC` | Exit upload mode |

### Lambda List
| Key | Action |
|-----|--------|
| `Enter` | View function details |
| `i` | Invoke function immediately |
| `l` | View function logs |
| `r` | Refresh function list |

### Lambda Detail
| Key | Action |
|-----|--------|
| `i` | Invoke function |
| `l` | View logs |
| `ESC` | Back to function list |

## Tips & Tricks

1. **Fast Navigation**: Use number keys (2, 3, 4) on dashboard for quick access
2. **Refresh Data**: Press 'r' on any list to reload from AWS
3. **Downloads**: Downloaded files go to the directory where you launched the TUI
4. **Uploads**: Press 'u' in S3 browser to open dual-pane upload mode
5. **Hidden Files**: Press 'h' in upload mode to toggle hidden file visibility
6. **Lambda Payload**: Default payload is `{}` - customize as needed
7. **Logs**: Most recent logs appear at the top
8. **Metrics**: Metrics auto-load when viewing details
9. **ESC Key**: Always takes you back one level
10. **Quit Anytime**: Press 'q' from any screen to exit

## Performance Notes

- Resource lists load asynchronously (won't block UI)
- Metrics fetch in background
- Large file downloads show progress
- Error messages display inline
- No API calls made until you navigate to a screen

## Common Workflows

### Check EC2 Status
```
1. Launch: genesys tui
2. Press '2' for EC2
3. See all instances with status
4. Press Enter on one to see metrics
```

### Download S3 File
```
1. Launch: genesys tui
2. Press '3' for S3
3. Press Enter on bucket
4. Navigate to file with Enter/Backspace
5. Press 'd' to download
6. Check current directory for file
```

### Upload File to S3
```
1. Launch: genesys tui
2. Press '3' for S3
3. Press Enter on bucket
4. Navigate to desired S3 folder (optional)
5. Press 'u' to enter upload mode
6. Use Tab to switch between local/S3 panes
7. Navigate local filesystem to find your file
8. Press Enter on a file to upload it
9. Progress modal shows upload status
10. Press ESC to exit upload mode
```

### Test Lambda Function
```
1. Launch: genesys tui
2. Press '4' for Lambda
3. Press Enter on function
4. Press 'i' to invoke
5. Modify payload if needed
6. View result in modal
```

### Monitor EC2 Performance
```
1. Launch: genesys tui
2. Press '2' for EC2
3. Press Enter on instance
4. Press 'm' for metrics
5. See CPU, Network, Disk stats
```
