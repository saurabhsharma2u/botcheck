package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/saurabhsharma2u/botcheck/internal/config"
	"github.com/saurabhsharma2u/botcheck/internal/dnsverify"
	"github.com/saurabhsharma2u/botcheck/internal/registry"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	dnsCheck   bool
)

// dnsTimeout bounds both PTR + forward lookups so --dns never hangs.
const dnsTimeout = 10 * time.Second

func init() {
	checkCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	checkCmd.Flags().BoolVar(&dnsCheck, "dns", false, "Verify with forward-confirmed reverse DNS; unmatched IPs fall back to DNS automatically (requires network)")
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
				fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
			}
		}

		reg := registry.NewDiskRegistry(cacheDir)
		if err := reg.Load(ctx); err != nil {
			if errors.Is(err, registry.ErrEmpty) {
				return err
			}
			return fmt.Errorf("load registry: %w", err)
		}

		hit, ok := reg.Contains(ip)

		// DNS verification only runs on a registry match: the match tells
		// us which source's suffixes to expect. It confirms the verdict,
		// it never overrides a match on its own.
		var verdict dnsverify.Verdict
		dnsRan := false
		if ok && dnsCheck {
			ctx, cancel := context.WithTimeout(ctx, dnsTimeout)
			defer cancel()
			var err error
			verdict, err = dnsverify.NewVerifier().VerifyIP(ctx, ip, dnsverify.Suffixes(hit.Meta.Extra))
			if err != nil {
				return fmt.Errorf("dns verification: %w", err)
			}
			dnsRan = true
		}

		// DNS fallback for IPs with no range match: check the PTR hostname
		// against every known source's suffixes. This catches genuine bots
		// from unlisted ranges (e.g. Yandex rotates undisclosed IPs).
		// Single-pass inside VerifyAny: no recursion, no retries, no loops.
		var fbOwners []string
		var fbPTRs []string
		if !ok && dnsCheck {
			dir := cacheDir
			if dir == "" {
				dir = registry.DefaultCacheDir()
			}
			bySource, err := registry.LoadVerifyMap(dir)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if !jsonOutput {
						fmt.Fprintf(os.Stderr, "Note: dns fallback unavailable (run refresh to create verify data)\n")
					}
				} else {
					return fmt.Errorf("load verify map: %w", err)
				}
			} else {
				ctx, cancel := context.WithTimeout(ctx, dnsTimeout)
				defer cancel()
				var err error
				fbOwners, verdict, fbPTRs, err = dnsverify.VerifyAny(ctx, &net.Resolver{}, ip, bySource)
				if err != nil {
					return fmt.Errorf("dns verification: %w", err)
				}
				dnsRan = true
			}
		}

		if jsonOutput {
			out := map[string]interface{}{
				"ip":      ip.String(),
				"matched": ok,
			}
			if ok {
				out["prefix"] = hit.Prefix.String()
				out["meta"] = hit.Meta
			}
			if dnsRan {
				out["dns_verified"] = verdict.Verified
				if verdict.Hostname != "" {
					out["dns_hostname"] = verdict.Hostname
				}
				if len(fbOwners) > 0 {
					out["dns_source"] = strings.Join(fbOwners, ",")
				}
				if len(fbPTRs) > 0 {
					out["dns_ptrs"] = fbPTRs
				}
				if !verdict.Verified {
					out["dns_reason"] = verdict.Reason
				}
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return nil
		}

		if ok {
			fmt.Printf("MATCH: %s\n", ip)
			fmt.Printf("  Prefix:   %s\n", hit.Prefix)
			fmt.Printf("  Source:   %s\n", hit.Meta.Source)
			fmt.Printf("  Category: %s\n", hit.Meta.Category)
			fmt.Printf("  Name:     %s\n", hit.Meta.Name)
			if dnsRan {
				if verdict.Verified {
					fmt.Printf("  DNS:      verified (%s)\n", verdict.Hostname)
				} else {
					fmt.Printf("  DNS:      NOT VERIFIED (%s)\n", verdict.Reason)
				}
			}
		} else {
			fmt.Printf("NO MATCH: %s\n", ip)
			if dnsRan {
				if verdict.Verified {
					fmt.Printf("  DNS:      verified via %s (%s)\n", strings.Join(fbOwners, ","), verdict.Hostname)
				} else {
					fmt.Printf("  DNS:      NOT VERIFIED (%s)\n", verdict.Reason)
					if len(fbPTRs) > 0 {
						fmt.Printf("  PTR:      %s\n", strings.Join(fbPTRs, ", "))
					}
				}
			}
		}

		return nil
	},
}
