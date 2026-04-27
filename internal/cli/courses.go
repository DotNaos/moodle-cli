package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var coursesCmd = &cobra.Command{
	Use:     "courses",
	Short:   "List your enrolled courses",
	Long:    "List all courses you are enrolled in.\n\nBy default, the output is a table: course ID, full name, and category.\nUse the global output flags to return machine-readable course objects.",
	Example: "  moodle list courses\n  moodle --json list courses\n  moodle --yaml list courses",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureCourseDataClient()
		if err != nil {
			return err
		}

		courses, err := client.FetchCourses()
		if err != nil {
			return err
		}

		return writeCommandOutput(cmd, courses, func(w io.Writer) error {
			for _, course := range courses {
				if _, err := fmt.Fprintf(w, "%d\t%s\t%s\n", course.ID, course.Fullname, course.Category); err != nil {
					return err
				}
			}
			return nil
		})
	},
}
