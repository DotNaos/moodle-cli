package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:               "files <course-id|name|current|0>",
	Short:             "List files and folders in a course",
	Long:              "List all files and folders for a course.\n\nThe course can be specified by ID, name, `current`, `0`, or a positive index. Output includes resource ID, type, name, and section.",
	Example:           "  moodle list files 12345\n  moodle list files current\n  moodle list files 0\n  moodle --json list files 12345",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeCourseIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courseID, err := resolveCourseIDWithOptions(client, args[0], selectorOptions{})
		if err != nil {
			return err
		}
		resources, _, err := client.FetchCourseResources(courseID)
		if err != nil {
			return err
		}

		return writeCommandOutput(cmd, resources, func(w io.Writer) error {
			for _, res := range resources {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", res.ID, res.Type, res.Name, res.SectionName); err != nil {
					return err
				}
			}
			return nil
		})
	},
}
