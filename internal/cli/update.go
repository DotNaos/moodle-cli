package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/DotNaos/moodle-cli/internal/update"
	ver "github.com/DotNaos/moodle-cli/internal/version"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install a newer release",
	Long:  "Check GitHub Releases for a newer stable version and install it automatically when available.",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client := update.NewClient()
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		if updateCheckOnly {
			availability, _, err := client.Check(ctx, ver.Version())
			if err != nil {
				if errors.Is(err, update.ErrNoStableRelease) {
					fmt.Fprintln(cmd.OutOrStdout(), "no stable release published yet")
					return nil
				}
				return err
			}
			if availability.NeedsUpdate {
				fmt.Fprintf(cmd.OutOrStdout(), "update available: %s -> %s\n", ver.Version(), availability.LatestTag)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "up to date: %s\n", availability.CurrentVersion)
			return nil
		}

		executablePath, err := os.Executable()
		if err != nil {
			return err
		}

		result, err := client.Update(ctx, executablePath, ver.Version())
		if err != nil {
			if errors.Is(err, update.ErrNoStableRelease) {
				fmt.Fprintln(cmd.OutOrStdout(), "no stable release published yet")
				return nil
			}
			return err
		}

		if !result.Updated {
			fmt.Fprintf(cmd.OutOrStdout(), "already up to date: %s\n", ver.Version())
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "updated %s to %s\n", executablePath, result.InstalledTag)
		if err := saveUpdateStateAfterInstall(opts.StatePath, result.InstalledTag); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update state file: %v\n", err)
		}
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Only check for a newer version without installing it")
}

func saveUpdateStateAfterInstall(path string, tag string) error {
	state, err := update.LoadState(path)
	if err != nil {
		return err
	}
	state.LastUpdateCheckAt = time.Now()
	state.LastNotifiedTag = tag
	return update.SaveState(path, state)
}
