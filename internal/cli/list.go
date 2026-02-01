package cli

import "github.com/spf13/cobra"

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List courses, files, or timetable entries",
	Long:    "List data from Moodle.\n\nUse one of the subcommands:\n  - list courses\n  - list files <course>\n  - list timetable",
	Example: "  moodle list courses\n  moodle list files 12345\n  moodle list timetable",
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	listCmd.AddCommand(
		coursesCmd,
		filesCmd,
		timetableCmd,
	)
}
