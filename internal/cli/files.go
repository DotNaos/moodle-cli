package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var filesJSON bool

var filesCmd = &cobra.Command{
	Use:               "files <course-id|name>",
	Short:             "List files and folders in a course",
	Long:              "List all files and folders for a course.\n\nThe course can be specified by ID or name. Output includes resource ID, type, name, and section.",
	Example:           "  moodle list files 12345\n  moodle list files \"Mathematik II (cds-402) FS25\"\n  moodle list files 12345 --json",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeCourseIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courseID, err := resolveCourseID(client, args[0])
		if err != nil {
			return err
		}
		resources, _, err := client.FetchCourseResources(courseID)
		if err != nil {
			return err
		}

		if filesJSON {
			data, err := json.MarshalIndent(resources, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		for _, res := range resources {
			fmt.Printf("%s\t%s\t%s\t%s\n", res.ID, res.Type, res.Name, res.SectionName)
		}
		return nil
	},
}

func init() {
	filesCmd.Flags().BoolVar(&filesJSON, "json", false, "Output JSON")
}
