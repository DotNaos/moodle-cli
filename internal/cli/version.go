package cli

import (
	"fmt"

	ver "github.com/DotNaos/moodle-cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Show the current CLI version, commit, and build date.",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "version: %s\n", ver.Version())
		fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", ver.Commit())
		fmt.Fprintf(cmd.OutOrStdout(), "buildDate: %s\n", ver.BuildDate())
		return nil
	},
}
