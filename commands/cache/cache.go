package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewCacheCommand creates the cache command
func NewCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage Genesys caches",
		Long: `Manage Genesys caching systems including AMI cache, Lambda layer cache, and pricing cache.

This command allows you to:
  • Clear all caches or specific cache types
  • View cache statistics and status
  • Refresh caches by removing expired entries`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newCacheClearCommand())
	cmd.AddCommand(newCacheListCommand())
	cmd.AddCommand(newCacheRefreshCommand())

	return cmd
}

// newCacheClearCommand creates the cache clear subcommand
func newCacheClearCommand() *cobra.Command {
	var cacheType string

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear Genesys caches",
		Long: `Clear Genesys caches to force fresh data retrieval.

Cache types:
  • ami     - AMI resolver cache (in-memory, cleared on restart)
  • lambda  - Lambda layer build cache (filesystem-based)
  • pricing - Pricing data cache (static data, informational only)
  • all     - Clear all filesystem-based caches

Note: AMI cache is in-memory and automatically cleared when the process restarts.
Only filesystem-based caches (Lambda layers) are permanently cleared.

Examples:
  genesys cache clear --type lambda    # Clear Lambda layer cache
  genesys cache clear --type all       # Clear all filesystem caches
  genesys cache clear                  # Interactive selection`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no type specified, show options
			if cacheType == "" {
				fmt.Println("Available cache types to clear:")
				fmt.Println("  • lambda  - Lambda layer build cache (~/.cache/genesys/lambda-layers)")
				fmt.Println("  • all     - All filesystem-based caches")
				fmt.Println("")
				fmt.Println("Note: AMI cache is in-memory and clears automatically on restart")
				fmt.Println("")
				fmt.Println("Usage: genesys cache clear --type <type>")
				return nil
			}

			switch strings.ToLower(cacheType) {
			case "lambda":
				return clearLambdaCache()
			case "all":
				fmt.Println("Clearing all filesystem-based caches...")
				if err := clearLambdaCache(); err != nil {
					return err
				}
				fmt.Println("[OK] All caches cleared successfully")
				return nil
			case "ami":
				fmt.Println("[INFO] AMI cache is in-memory and automatically cleared on process restart")
				fmt.Println("[INFO] No persistent cache to clear")
				return nil
			case "pricing":
				fmt.Println("[INFO] Pricing data is static and not cached")
				fmt.Println("[INFO] No cache to clear")
				return nil
			default:
				return fmt.Errorf("unknown cache type: %s (valid types: lambda, all)", cacheType)
			}
		},
	}

	cmd.Flags().StringVar(&cacheType, "type", "", "Cache type to clear (lambda, all)")

	return cmd
}

// newCacheListCommand creates the cache list subcommand
func newCacheListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List cache status and statistics",
		Long: `List status and statistics for all Genesys caches.

This shows:
  • Lambda layer cache location and size
  • Number of cached layers
  • Total disk space used
  • Cache type information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Genesys Cache Status:")
			fmt.Println(strings.Repeat("=", 60))
			fmt.Println("")

			// Lambda Layer Cache
			fmt.Println("Lambda Layer Cache:")
			lambdaCachePath := getLambdaCachePath()
			if info, err := os.Stat(lambdaCachePath); err == nil && info.IsDir() {
				entries, _ := os.ReadDir(lambdaCachePath)
				layerCount := 0
				var totalSize int64

				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zip") {
						layerCount++
						if fileInfo, err := entry.Info(); err == nil {
							totalSize += fileInfo.Size()
						}
					}
				}

				fmt.Printf("  Location: %s\n", lambdaCachePath)
				fmt.Printf("  Cached Layers: %d\n", layerCount)
				fmt.Printf("  Total Size: %.2f MB\n", float64(totalSize)/(1024*1024))
			} else {
				fmt.Printf("  Location: %s\n", lambdaCachePath)
				fmt.Println("  Status: No cache directory (will be created on first use)")
			}
			fmt.Println("")

			// AMI Cache
			fmt.Println("AMI Cache:")
			fmt.Println("  Type: In-memory (per-process)")
			fmt.Println("  TTL: 24 hours (default)")
			fmt.Println("  Status: Cleared automatically on process restart")
			fmt.Println("")

			// Pricing Cache
			fmt.Println("Pricing Cache:")
			fmt.Println("  Type: Static data (embedded in binary)")
			fmt.Println("  Status: No caching required")
			fmt.Println("")

			return nil
		},
	}

	return cmd
}

// newCacheRefreshCommand creates the cache refresh subcommand
func newCacheRefreshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh caches by removing expired entries",
		Long: `Refresh caches by removing expired or old entries.

This command:
  • Removes Lambda layers older than 7 days
  • Cleans up orphaned cache files
  • Reports cache cleanup statistics

Note: AMI cache expiration is handled automatically at runtime.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Refreshing Genesys caches...")
			fmt.Println("")

			// Clean old Lambda layers (older than 7 days)
			lambdaCachePath := getLambdaCachePath()
			if info, err := os.Stat(lambdaCachePath); err == nil && info.IsDir() {
				removed := 0
				var freedSpace int64

				entries, _ := os.ReadDir(lambdaCachePath)
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					filePath := filepath.Join(lambdaCachePath, entry.Name())
					fileInfo, err := entry.Info()
					if err != nil {
						continue
					}

					// Remove files older than 7 days
					if fileInfo.ModTime().AddDate(0, 0, 7).Before(time.Now()) {
						freedSpace += fileInfo.Size()
						if err := os.Remove(filePath); err == nil {
							removed++
						}
					}
				}

				fmt.Printf("Lambda Layer Cache:\n")
				fmt.Printf("  Removed: %d old layers\n", removed)
				fmt.Printf("  Freed: %.2f MB\n", float64(freedSpace)/(1024*1024))
			} else {
				fmt.Println("Lambda Layer Cache: No cache to refresh")
			}

			fmt.Println("")
			fmt.Println("[OK] Cache refresh completed")
			return nil
		},
	}

	return cmd
}

// getLambdaCachePath returns the Lambda layer cache path
func getLambdaCachePath() string {
	// Use the same path as in pkg/lambda/layer.go
	return filepath.Join(os.TempDir(), "genesys-lambda-layers")
}

// clearLambdaCache removes all Lambda layer cache files
func clearLambdaCache() error {
	lambdaCachePath := getLambdaCachePath()

	if _, err := os.Stat(lambdaCachePath); os.IsNotExist(err) {
		fmt.Println("[INFO] Lambda cache directory does not exist, nothing to clear")
		return nil
	}

	// Remove the entire cache directory
	if err := os.RemoveAll(lambdaCachePath); err != nil {
		return fmt.Errorf("failed to clear Lambda cache: %w", err)
	}

	fmt.Printf("[OK] Lambda layer cache cleared: %s\n", lambdaCachePath)
	return nil
}
