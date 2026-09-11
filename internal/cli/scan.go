package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/saurabhsharma2u/iambot/internal/config"
	"github.com/saurabhsharma2u/iambot/internal/logparse"
	"github.com/saurabhsharma2u/iambot/internal/registry"
	"github.com/saurabhsharma2u/iambot/internal/report"
	"github.com/spf13/cobra"
)

var (
	scanFormat        string
	scanOutput        string
	scanOnlyMatched   bool
	scanOnlyUnmatched bool
	scanQuiet         bool
)

func init() {
	scanCmd.Flags().StringVarP(&configPath, "config", "c", "botcheck.yaml", "Path to config file")
	scanCmd.Flags().StringVar(&scanFormat, "format", "auto", "Log format (auto|nginx|apache|json|cloudflare)")
	scanCmd.Flags().StringVar(&scanOutput, "output", "text", "Output format (text|json|csv)")
	scanCmd.Flags().BoolVar(&scanOnlyMatched, "only-matched", false, "Only output matched IPs")
	scanCmd.Flags().BoolVar(&scanOnlyUnmatched, "only-unmatched", false, "Only output unmatched IPs")
	scanCmd.Flags().BoolVar(&scanQuiet, "quiet", false, "Suppress non-essential output")

	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan <logfile...>",
	Short: "Scan log files for known bot IPs",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		var cacheDir string
		cfg, err := config.Load(configPath)
		if err == nil {
			cacheDir = cfg.CacheDir
		} else if !errors.Is(err, os.ErrNotExist) && !scanQuiet {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		}

		reg := registry.NewDiskRegistry(cacheDir)
		if err := reg.Load(ctx); err != nil {
			return fmt.Errorf("load registry: %w", err)
		}

		parser, err := logparse.GetParser(scanFormat)
		if err != nil {
			return err
		}

		reporter, err := report.GetReporter(scanOutput)
		if err != nil {
			return err
		}

		for _, filename := range args {
			if !scanQuiet {
				fmt.Fprintf(os.Stderr, "Scanning %s...\n", filename)
			}

			f, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", filename, err)
				continue
			}

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				entry, err := parser.Parse(line)
				if err != nil {
					continue // skip invalid lines
				}

				hit, matched := reg.Contains(entry.IP)

				if scanOnlyMatched && !matched {
					continue
				}
				if scanOnlyUnmatched && matched {
					continue
				}

				res := report.Result{
					Entry: entry,
					Hit:   hit,
					Match: matched,
				}

				if err := reporter.Report(os.Stdout, res); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filename, err)
			}
			f.Close()
		}

		return nil
	},
}
