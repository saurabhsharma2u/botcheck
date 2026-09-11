package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"

	"github.com/saurabhsharma2u/iambot/internal/config"
	"github.com/saurabhsharma2u/iambot/internal/registry"
	"github.com/spf13/cobra"
)

var jsonOutput bool

func init() {
	checkCmd.Flags().StringVarP(&configPath, "config", "c", "botcheck.yaml", "Path to config file")
	checkCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check <IP>",
	Short: "Check a single IP against the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		ipStr := args[0]

		ip, err := netip.ParseAddr(ipStr)
		if err != nil {
			return fmt.Errorf("invalid IP address: %w", err)
		}

		var cacheDir string
		cfg, err := config.Load(configPath)
		if err == nil {
			cacheDir = cfg.CacheDir
		} else if !errors.Is(err, os.ErrNotExist) {
			if !jsonOutput {
				fmt.Printf("Warning: failed to load config: %v\n", err)
			}
		}

		reg := registry.NewDiskRegistry(cacheDir)
		if err := reg.Load(ctx); err != nil {
			return fmt.Errorf("load registry: %w", err)
		}

		hit, ok := reg.Contains(ip)

		if jsonOutput {
			out := map[string]interface{}{
				"ip":      ip.String(),
				"matched": ok,
			}
			if ok {
				out["meta"] = hit.Meta
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return nil
		}

		if ok {
			fmt.Printf("MATCH: %s\n", ip)
			fmt.Printf("  Source:   %s\n", hit.Meta.Source)
			fmt.Printf("  Category: %s\n", hit.Meta.Category)
			fmt.Printf("  Name:     %s\n", hit.Meta.Name)
		} else {
			fmt.Printf("NO MATCH: %s\n", ip)
		}

		return nil
	},
}
