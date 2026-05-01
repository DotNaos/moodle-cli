package cli

import (
	"bytes"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

type completionResult struct {
	Shell  string `json:"shell" yaml:"shell"`
	Script string `json:"script" yaml:"script"`
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate the autocompletion script for the specified shell",
	Long:  "Generate the autocompletion script for moodle for the specified shell.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return helpOrMachineError(cmd, "expected a shell name such as bash, zsh, fish, or powershell")
	},
}

func newCompletionShellCmd(shell string, generator func(io.Writer) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Generate the autocompletion script for %s", shell),
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var script bytes.Buffer
			if err := generator(&script); err != nil {
				return err
			}
			result := completionResult{
				Shell:  shell,
				Script: script.String(),
			}
			return writeCommandOutput(cmd, result, func(w io.Writer) error {
				_, err := io.Copy(w, &script)
				return err
			})
		},
	}
	markAPIOptional(cmd)
	return cmd
}

func init() {
	markAPIOptional(completionCmd)
	completionCmd.AddCommand(
		newCompletionShellCmd("bash", func(w io.Writer) error {
			return rootCmd.GenBashCompletionV2(w, true)
		}),
		newCompletionShellCmd("fish", func(w io.Writer) error {
			return rootCmd.GenFishCompletion(w, true)
		}),
		newCompletionShellCmd("powershell", func(w io.Writer) error {
			return rootCmd.GenPowerShellCompletionWithDesc(w)
		}),
		newCompletionShellCmd("zsh", func(w io.Writer) error {
			return rootCmd.GenZshCompletion(w)
		}),
	)
}
