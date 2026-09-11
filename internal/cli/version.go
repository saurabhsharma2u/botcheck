package cli

import (
	"fmt"
	"github.com/saurabhsharma2u/iambot/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of botcheck",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("botcheck version %s (commit: %s, built at: %s)\n", version.Version, version.GitCommit, version.BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
