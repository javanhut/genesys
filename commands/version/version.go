package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command
func NewVersionCommand(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Genesys version %s (commit: %s)\n", version, commit)
		},
	}
}
