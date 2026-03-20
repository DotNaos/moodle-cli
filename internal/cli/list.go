package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var listSelectionJSON bool
var listSelectionWorkspace string
var listSelectionAt string

var listCmd = &cobra.Command{
	Use:     "list [course] [resource]",
	Short:   "List courses, files, or timetable entries",
	Long:    "List data from Moodle.\n\nUse either the subcommands or a course/resource selector pair such as `moodle list current current`.",
	Example: "  moodle list courses\n  moodle list files 12345\n  moodle list timetable\n  moodle list current current\n  moodle list current-course\n  moodle list 0 0",
	Args: func(cmd *cobra.Command, args []string) error {
		args = expandSingleCurrentAlias(args)
		if len(args) == 0 {
			return nil
		}
		if len(args) != 2 {
			return fmt.Errorf("expected either a subcommand or exactly 2 arguments: <course> <resource>")
		}
		return nil
	},
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completeListSelectionArgs(nil, nil, "")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		args = expandSingleCurrentAlias(args)
		if len(args) == 0 {
			return cmd.Help()
		}
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}
		result, err := buildSelectedCourseResult(client, args[0], args[1], listSelectionWorkspace, listSelectionAt)
		if err != nil {
			return err
		}
		if listSelectionJSON {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		renderCurrentLectureText(result)
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listSelectionJSON, "json", false, "Output JSON")
	listCmd.Flags().StringVar(&listSelectionWorkspace, "workspace", "", "Optional workspace root for local file matching")
	listCmd.Flags().StringVar(&listSelectionAt, "at", "", "Override current time for testing (RFC3339)")
	listCmd.AddCommand(
		coursesCmd,
		currentLectureCmd,
		filesCmd,
		timetableCmd,
	)
}
