package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/saurabhsharma2u/iambot/internal/config"
	"github.com/saurabhsharma2u/iambot/internal/registry"
	"github.com/saurabhsharma2u/iambot/internal/source"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

func init() {
	updateCmd.Flags().StringVarP(&configPath, "config", "c", "botcheck.yaml", "Path to config file")
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the IP registry from configured sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := config.Load(configPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) && configPath == "botcheck.yaml" {
				return fmt.Errorf("config file not found, please provide a valid botcheck.yaml")
			}
			return err
		}

		reg := registry.NewDiskRegistry(cfg.CacheDir)

		fmt.Printf("Updating botcheck registry...\n")

		var allEntries []registry.Entry
		stats := registry.Stats{
			Sources:    make(map[string]int),
			Categories: make(map[string]int),
		}

		for _, sc := range cfg.Sources {
			if !sc.Enabled {
				continue
			}

			fmt.Printf("Fetching %s (%s)...\n", sc.Name, sc.URL)

			var src source.Source
			if sc.Type == "http" {
				src = source.NewHTTP(sc.Name, sc.Category, sc.URL)
			} else {
				fmt.Printf("Warning: unsupported source type %s for %s\n", sc.Type, sc.Name)
				continue
			}

			prefixes, meta, err := src.Fetch(ctx)
			if err != nil {
				fmt.Printf("Error fetching %s: %v\n", sc.Name, err)
				continue
			}

			for _, p := range prefixes {
				allEntries = append(allEntries, registry.Entry{
					Prefix: p.String(),
					Meta:   meta,
				})
			}
			stats.TotalPrefixes += len(prefixes)
			stats.Sources[sc.Name] += len(prefixes)
			stats.Categories[sc.Category] += len(prefixes)

			fmt.Printf("  Loaded %d prefixes for %s\n", len(prefixes), sc.Name)
		}

		if len(allEntries) == 0 {
			return fmt.Errorf("no prefixes fetched from any source")
		}

		if err := reg.SaveRaw(ctx, allEntries, stats); err != nil {
			return fmt.Errorf("save registry: %w", err)
		}

		fmt.Printf("Update complete. Total prefixes: %d\n", stats.TotalPrefixes)
		return nil
	},
}
