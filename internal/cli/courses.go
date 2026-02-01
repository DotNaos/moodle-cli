package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var coursesJSON bool

var coursesCmd = &cobra.Command{
	Use:     "courses",
	Short:   "List your enrolled courses",
	Long:    "List all courses you are enrolled in.\n\nBy default, the output is a table: course ID, full name, and category.\nUse --json to return the full course objects.",
	Example: "  moodle list courses\n  moodle list courses --json",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courses, err := client.FetchCourses()
		if err != nil {
			return err
		}

		if coursesJSON {
			data, err := json.MarshalIndent(courses, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		for _, course := range courses {
			fmt.Printf("%d\t%s\t%s\n", course.ID, course.Fullname, course.Category)
		}
		return nil
	},
}

func init() {
	coursesCmd.Flags().BoolVar(&coursesJSON, "json", false, "Output JSON")
}
