# Genesys Data Refresh & Reload Implementation

## Summary of Changes

This implementation fixes the data refresh and reload issues in Genesys by adding comprehensive refresh mechanisms across the entire system.

## New Features Added

### 1. Credential Refresh Logic (`pkg/provider/aws/client.go`)
- **Automatic token expiry detection**: Checks `ExpiresAt` field in stored credentials
- **Credential refresh function**: `refreshAWSCredentials()` reloads from AWS credentials file
- **AWS credentials file parser**: `parseAWSCredentialsFile()` handles malformed credential files
- **Credential validation**: `ValidateAWSCredentials()` tests credentials with AWS STS

### 2. Configuration Reload Mechanisms (`pkg/config/config.go`)
- **ConfigManager**: New struct for managing configuration with change detection
- **Hot reload capability**: `LoadConfig()` with caching and file modification tracking
- **Force reload**: `ReloadConfig()` bypasses cache for fresh configuration
- **Provider credential refresh**: `RefreshProviderCredentials()` reloads all provider configs

### 3. State Management Refresh (`pkg/state/local.go`, `pkg/provider/aws/state.go`)
- **Local state refresh**: `RefreshLocalState()` reloads from disk
- **Remote state sync**: `SyncWithRemote()` synchronizes with cloud state backends
- **State validation**: `ValidateResources()` checks if tracked resources still exist
- **Remote state refresh**: `Refresh()` forces fresh state retrieval from S3

### 4. Cache Management (`pkg/provider/aws/ami.go`)
- **Enhanced AMI cache**: Added `RefreshCache()` to remove expired entries
- **Improved cache stats**: Extended `GetCacheStats()` with source tracking
- **Existing clear functionality**: `ClearCache()` for complete cache reset

### 5. New CLI Commands

#### Config Refresh Commands
```bash
genesys config refresh    # Refresh provider credentials
genesys config validate   # Validate current credentials
```

#### Cache Management Commands
```bash
genesys cache clear [ami|all]  # Clear specific or all caches
genesys cache refresh          # Remove expired cache entries
genesys cache stats           # Show cache statistics
```

#### State Management Commands
```bash
genesys state list       # List managed resources
genesys state refresh    # Refresh state from disk and remote
genesys state validate   # Validate state consistency
genesys state sync      # Sync local and remote state
```

## Key Improvements

### 1. **Credential Management**
- Automatic detection and refresh of expired temporary credentials
- Better handling of malformed AWS credentials files
- Validation before operations to prevent auth failures

### 2. **Configuration Hot-Reload**
- File modification tracking to avoid unnecessary reloads
- Cached configuration for better performance
- Force refresh capability for development/debugging

### 3. **State Synchronization**
- Local state reload without restart
- Remote state refresh for multi-environment consistency
- Resource validation to detect drift

### 4. **Cache Management**
- Selective cache clearing (AMI-specific vs. all caches)
- Expired entry cleanup without full cache reset
- Detailed statistics for monitoring and debugging

## Usage Examples

### Fixing Credential Issues
```bash
# When credentials are expired or malformed:
genesys config refresh

# To test if credentials are working:
genesys config validate
```

### Refreshing Cached Data
```bash
# Clear AMI cache for fresh lookups:
genesys cache clear ami

# Remove expired entries only:
genesys cache refresh

# Check cache status:
genesys cache stats
```

### State Management
```bash
# Reload state after external changes:
genesys state refresh

# Check what resources are tracked:
genesys state list

# Validate that tracked resources still exist:
genesys state validate
```

## Technical Details

- **Backward Compatible**: All existing functionality preserved
- **Error Handling**: Graceful degradation when refresh fails
- **Performance**: Caching prevents unnecessary API calls
- **Security**: No credentials logged or exposed in error messages
- **Multi-Provider Ready**: Framework supports AWS, GCP, Azure expansion

## Files Modified

1. `pkg/provider/aws/client.go` - Credential refresh and validation
2. `pkg/provider/aws/state.go` - Remote state refresh
3. `pkg/provider/aws/ami.go` - Enhanced cache management
4. `pkg/state/local.go` - Local state refresh
5. `pkg/config/config.go` - Configuration reload mechanisms
6. `cmd/genesys/commands/config.go` - Config refresh commands
7. `cmd/genesys/commands/cache.go` - New cache management commands
8. `cmd/genesys/commands/state.go` - New state management commands
9. `cmd/genesys/main.go` - Added new commands to CLI

The implementation addresses all the original data refresh and reload issues while providing a comprehensive set of tools for managing Genesys data lifecycle.