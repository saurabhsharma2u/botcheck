package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/saurabhsharma2u/iambot/internal/config"
	"github.com/saurabhsharma2u/iambot/internal/registry"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd.Flags().StringVarP(&configPath, "config", "c", "botcheck.yaml", "Path to config file")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current status of the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var cacheDir, registryURL string
		cfg, err := config.Load(configPath)
		if err == nil {
			cacheDir = cfg.CacheDir
			registryURL = cfg.ResolvedRegistryURL()
		} else {
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Printf("Warning: failed to load config: %v\n", err)
			}
			registryURL = (&config.Config{}).ResolvedRegistryURL()
		}

		reg := registry.NewDiskRegistry(cacheDir)
		if err := reg.Load(ctx); err != nil {
			return fmt.Errorf("load registry: %w", err)
		}

		stats := reg.Stats()
		fmt.Printf("Registry Status:\n")
		fmt.Printf("  Registry:       %s\n", registryURL)
		if stats.DataVersion != "" {
			fmt.Printf("  Data Version:   %s\n", stats.DataVersion)
		}
		fmt.Printf("  Last Updated:   %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Total Prefixes: %d\n", stats.TotalPrefixes)

		fmt.Printf("\nSources:\n")
		for name, count := range stats.Sources {
			fmt.Printf("  - %s: %d\n", name, count)
		}

		fmt.Printf("\nCategories:\n")
		for cat, count := range stats.Categories {
			fmt.Printf("  - %s: %d\n", cat, count)
		}

		return nil
	},
}
