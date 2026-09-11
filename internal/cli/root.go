package cli

import (
	"context"

	"github.com/saurabhsharma2u/botcheck/internal/version"
	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:          "botcheck",
	Short:        "botcheck is a tool to manage and check IP addresses against known bot lists",
	Version:      version.Version,
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "botcheck.yaml", "Path to config file")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
