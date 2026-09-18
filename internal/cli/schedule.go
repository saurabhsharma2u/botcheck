package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saurabhsharma2u/botcheck/internal/scheduler"
	"github.com/spf13/cobra"
)

var (
	scheduleConfig string
	scheduleAt     string
)

func init() {
	scheduleInstallCmd.Flags().StringVar(&scheduleConfig, "unit-config", "", "Config file the scheduled refresh should use (default: botcheck default discovery)")
	scheduleInstallCmd.Flags().StringVar(&scheduleAt, "at", "03:17", "Daily run time as HH:MM (24h)")

	scheduleCmd.AddCommand(scheduleInstallCmd)
	scheduleCmd.AddCommand(scheduleUninstallCmd)
	scheduleCmd.AddCommand(scheduleStatusCmd)
	rootCmd.AddCommand(scheduleCmd)
}

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage OS-native scheduled refresh (systemd on Linux, launchd on macOS)",
}

var scheduleInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and enable daily refresh",
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		msg, err := scheduler.Install(scheduler.Options{Binary: exe, Config: scheduleConfig, At: scheduleAt})
		if msg != "" {
			fmt.Println(msg)
		}
		return err
	},
}

var scheduleUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Disable and remove scheduled refresh",
	RunE: func(cmd *cobra.Command, args []string) error {
		msg, err := scheduler.Uninstall()
		if msg != "" {
			fmt.Println(msg)
		}
		return err
	},
}

var scheduleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show scheduled refresh state",
	RunE: func(cmd *cobra.Command, args []string) error {
		msg, err := scheduler.Status()
		if msg != "" {
			fmt.Println(msg)
		}
		return err
	},
}
