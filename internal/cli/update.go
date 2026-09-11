package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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
		return runUpdate(cmd.Context(), configPath, os.Stdout, os.Stderr)
	},
}

func runUpdate(ctx context.Context, cfgPath string, stdout, stderr io.Writer) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && cfgPath == "botcheck.yaml" {
			cfg = &config.Config{}
		} else {
			return err
		}
	}

	reg := registry.NewDiskRegistry(cfg.CacheDir)
	if err := reg.Load(ctx); err != nil {
		return fmt.Errorf("load registry: %w", err)
	}
	haveCache := reg.Stats().TotalPrefixes > 0

	var man *config.RegistryManifest
	if cfg.RegistryURL == "off" {
		_, _ = fmt.Fprintf(stderr, "Registry disabled, using local sources only.\n")
	} else {
		man, err = config.FetchRegistryManifest(ctx, cfg.ResolvedRegistryURL())
		if err != nil {
			if len(cfg.Sources) == 0 {
				if haveCache {
					_, _ = fmt.Fprintf(stderr, "Warning: registry unreachable (%v), keeping last-good cache.\n", err)
					return nil
				}
				return fmt.Errorf("fetch registry: %w", err)
			}
			_, _ = fmt.Fprintf(stderr, "Warning: registry unreachable (%v), using local sources only.\n", err)
		}
	}

	sources := cfg.Sources
	dataVersion := ""
	if man != nil {
		_, _ = fmt.Fprintf(stdout, "Registry %s (data %s).\n", cfg.ResolvedRegistryURL(), man.Version)
		sources = config.MergeSources(man.Sources, cfg.Sources)
		dataVersion = man.Version
	}

	_, _ = fmt.Fprintf(stdout, "Updating botcheck registry...\n")

	var allEntries []registry.Entry
	stats := registry.Stats{
		Sources:     make(map[string]int),
		Categories:  make(map[string]int),
		DataVersion: dataVersion,
	}

	for _, sc := range sources {
		if !sc.Enabled {
			continue
		}

		loc := sc.URL
		if sc.Type == "file" {
			loc = sc.Path
		}
		_, _ = fmt.Fprintf(stdout, "Fetching %s (%s)...\n", sc.Name, loc)

		var src source.Source
		switch sc.Type {
		case "http":
			src = source.NewHTTP(sc.Name, sc.Category, sc.URL)
		case "file":
			src = source.NewFile(sc.Name, sc.Category, sc.Path)
		default:
			_, _ = fmt.Fprintf(stdout, "Warning: unsupported source type %s for %s\n", sc.Type, sc.Name)
			continue
		}

		prefixes, meta, err := src.Fetch(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "Error fetching %s: %v\n", sc.Name, err)
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

		_, _ = fmt.Fprintf(stdout, "  Loaded %d prefixes for %s\n", len(prefixes), sc.Name)
	}

	if len(allEntries) == 0 {
		return fmt.Errorf("no prefixes fetched from any source")
	}

	if err := reg.SaveRaw(ctx, allEntries, stats); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Update complete. Total prefixes: %d\n", stats.TotalPrefixes)
	return nil
}
