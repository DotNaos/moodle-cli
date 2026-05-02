package cli

import (
	"fmt"
	"io"

	ver "github.com/DotNaos/moodle-services/internal/version"
	"github.com/spf13/cobra"
)

type versionResult struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	BuildDate string `json:"buildDate" yaml:"buildDate"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Show the current CLI version, commit, and build date.",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result := versionResult{
			Version:   ver.Version(),
			Commit:    ver.Commit(),
			BuildDate: ver.BuildDate(),
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			if _, err := fmt.Fprintf(w, "version: %s\n", result.Version); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "commit: %s\n", result.Commit); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "buildDate: %s\n", result.BuildDate); err != nil {
				return err
			}
			return nil
		})
	},
}
