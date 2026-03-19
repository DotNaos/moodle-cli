package cli

import (
	"fmt"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:     "open",
	Short:   "Open a course or resource in your browser",
	Long:    "Open Moodle courses or resources in your default browser.\n\nUse one of the subcommands:\n  - open course <course>\n  - open resource <course> <resource>",
	Example: "  moodle open course 12345\n  moodle open resource 12345 67890",
}

var openCourseCmd = &cobra.Command{
	Use:               "course <course-id|name>",
	Short:             "Open a course in your browser",
	Long:              "Open a Moodle course in your default browser.\n\nThe course can be specified by ID or name.",
	Example:           "  moodle open course 12345\n  moodle open course \"Mathematik II (cds-402) FS25\"",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeCourseIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courses, err := client.FetchCourses()
		if err != nil {
			return err
		}

		courseID, err := resolveCourseIDFromCourses(courses, args[0])
		if err != nil {
			return err
		}

		course, err := findCourseByID(courses, courseID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(course.ViewURL) == "" {
			return fmt.Errorf("course %s has no view URL", courseID)
		}

		return openURL(course.ViewURL)
	},
}

var openResourceCmd = &cobra.Command{
	Use:               "resource <course-id|name> <resource-id|name>",
	Short:             "Open a resource in your browser",
	Long:              "Open a Moodle resource or folder in your default browser.\n\nThe course and resource can be specified by ID or name.",
	Example:           "  moodle open resource 12345 67890\n  moodle open resource \"Mathematik II (cds-402) FS25\" \"Übungsblatt Analysis 1\"",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeOpenResourceArgs,
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

		target, err := resolveResource(resources, args[1])
		if err != nil {
			return err
		}
		if strings.TrimSpace(target.URL) == "" {
			return fmt.Errorf("resource %s has no URL", target.ID)
		}

		return openURL(target.URL)
	},
}

func init() {
	openCmd.AddCommand(
		openCourseCmd,
		openResourceCmd,
	)
}

func findCourseByID(courses []moodle.Course, courseID string) (*moodle.Course, error) {
	for i := range courses {
		if fmt.Sprintf("%d", courses[i].ID) == courseID {
			return &courses[i], nil
		}
	}
	return nil, fmt.Errorf("course not found: %s", courseID)
}
