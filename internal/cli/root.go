package cli

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "botcheck",
	Short: "botcheck is a tool to manage and check IP addresses against known bot lists",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute(ctx context.Context) error {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
