package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/DotNaos/moodle-services/internal/update"
	ver "github.com/DotNaos/moodle-services/internal/version"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

type updateCheckResult struct {
	CurrentVersion string `json:"currentVersion" yaml:"currentVersion"`
	LatestTag      string `json:"latestTag,omitempty" yaml:"latestTag,omitempty"`
	NeedsUpdate    bool   `json:"needsUpdate" yaml:"needsUpdate"`
	Status         string `json:"status" yaml:"status"`
}

type updateCommandResult struct {
	Updated        bool   `json:"updated" yaml:"updated"`
	InstalledTag   string `json:"installedTag,omitempty" yaml:"installedTag,omitempty"`
	ExecutablePath string `json:"executablePath,omitempty" yaml:"executablePath,omitempty"`
	Status         string `json:"status" yaml:"status"`
	Warning        string `json:"warning,omitempty" yaml:"warning,omitempty"`
}

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
					result := updateCheckResult{
						CurrentVersion: ver.Version(),
						NeedsUpdate:    false,
						Status:         "no_stable_release",
					}
					return writeCommandOutput(cmd, result, func(w io.Writer) error {
						_, err := fmt.Fprintln(w, "no stable release published yet")
						return err
					})
				}
				return err
			}
			result := updateCheckResult{
				CurrentVersion: availability.CurrentVersion,
				LatestTag:      availability.LatestTag,
				NeedsUpdate:    availability.NeedsUpdate,
				Status:         "up_to_date",
			}
			if availability.NeedsUpdate {
				result.Status = "update_available"
			}
			return writeCommandOutput(cmd, result, func(w io.Writer) error {
				if availability.NeedsUpdate {
					_, err := fmt.Fprintf(w, "update available: %s -> %s\n", ver.Version(), availability.LatestTag)
					return err
				}
				_, err := fmt.Fprintf(w, "up to date: %s\n", availability.CurrentVersion)
				return err
			})
		}

		executablePath, err := os.Executable()
		if err != nil {
			return err
		}

		result, err := client.Update(ctx, executablePath, ver.Version())
		if err != nil {
			if errors.Is(err, update.ErrNoStableRelease) {
				output := updateCommandResult{
					Updated:        false,
					ExecutablePath: executablePath,
					Status:         "no_stable_release",
				}
				return writeCommandOutput(cmd, output, func(w io.Writer) error {
					_, err := fmt.Fprintln(w, "no stable release published yet")
					return err
				})
			}
			return err
		}

		if !result.Updated {
			output := updateCommandResult{
				Updated:        false,
				InstalledTag:   result.InstalledTag,
				ExecutablePath: executablePath,
				Status:         "up_to_date",
			}
			return writeCommandOutput(cmd, output, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "already up to date: %s\n", ver.Version())
				return err
			})
		}

		output := updateCommandResult{
			Updated:        true,
			InstalledTag:   result.InstalledTag,
			ExecutablePath: executablePath,
			Status:         "updated",
		}
		if err := saveUpdateStateAfterInstall(opts.StatePath, result.InstalledTag); err != nil {
			output.Warning = fmt.Sprintf("could not update state file: %v", err)
		}
		return writeCommandOutput(cmd, output, func(w io.Writer) error {
			if _, err := fmt.Fprintf(w, "updated %s to %s\n", executablePath, result.InstalledTag); err != nil {
				return err
			}
			if output.Warning != "" {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", output.Warning); err != nil {
					return err
				}
			}
			return nil
		})
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
