package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var openCurrentLectureWorkspace string
var openCurrentLectureAt string
var moodleDownloadFileToBuffer = func(client *moodle.Client, url string) (moodle.DownloadResult, error) {
	return client.DownloadFileToBuffer(url)
}

type openCommandResult struct {
	Action     string `json:"action" yaml:"action"`
	Target     string `json:"target" yaml:"target"`
	TargetType string `json:"targetType" yaml:"targetType"`
	CourseID   string `json:"courseId,omitempty" yaml:"courseId,omitempty"`
	ResourceID string `json:"resourceId,omitempty" yaml:"resourceId,omitempty"`
}

var openCmd = &cobra.Command{
	Use:     "open [course] [resource]",
	Short:   "Open a course or resource in your browser",
	Long:    "Open Moodle courses or resources in your default browser.\n\nUse either the existing subcommands or direct selectors such as `moodle open current current`.",
	Example: "  moodle open current current\n  moodle open current-resource\n  moodle open 0 0\n  moodle open resource 12345 67890",
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
	ValidArgsFunction: completeOpenDirectArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		args = expandSingleCurrentAlias(args)
		if len(args) == 0 {
			return helpOrMachineError(cmd, "expected either a subcommand or exactly 2 arguments: <course> <resource>")
		}
		result, err := runOpenResourceSelection(args[0], args[1])
		if err != nil {
			return err
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return nil
		})
	},
}

var openCourseCmd = &cobra.Command{
	Use:               "course <course-id|name|current|0>",
	Short:             "Open a course in your browser",
	Long:              "Open a Moodle course in your default browser.\n\nThe course can be specified by ID, name, `current`, `0`, or a positive index.",
	Example:           "  moodle open course 12345\n  moodle open course current\n  moodle open course 0",
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
		courses, err := client.FetchCourses()
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

		if err := openURL(course.ViewURL); err != nil {
			return err
		}
		result := openCommandResult{
			Action:     "open",
			Target:     course.ViewURL,
			TargetType: "course_url",
			CourseID:   courseID,
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return nil
		})
	},
}

var openResourceCmd = &cobra.Command{
	Use:               "resource <course-id|name|current|0> <resource-id|name|current|0>",
	Short:             "Open a resource in your browser",
	Long:              "Open a Moodle resource or folder in your default browser.\n\nThe course and resource can be specified by ID, name, `current`, `0`, or a positive index.",
	Example:           "  moodle open resource 12345 67890\n  moodle open resource current current\n  moodle open resource 0 1",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeOpenResourceArgs,
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

		target, err := resolveResourceWithOptions(client, courseID, resources, args[1], selectorOptions{})
		if err != nil {
			return err
		}
		result, err := openResolvedResource(client, *target)
		if err != nil {
			return err
		}
		result.CourseID = courseID
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return nil
		})
	},
}

var openCurrentLectureCmd = &cobra.Command{
	Use:   "current-lecture",
	Short: "Open the best material for the current lecture",
	Long: "Resolve the current lecture from the timetable and open the best matching material in your browser.\n\n" +
		"If no material can be chosen safely, the matched course page is opened instead.",
	Example: "  moodle open current-lecture\n" +
		"  moodle open current-lecture --workspace /Users/oli/school\n" +
		"  moodle open current-lecture --at 2026-03-20T11:15:00+01:00",
	Args: cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		now, err := resolveLectureTimeAt(openCurrentLectureAt)
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(opts.ConfigPath)
		if err != nil {
			return err
		}
		if cfg.CalendarURL == "" {
			return fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
		}
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}
		result, err := buildCurrentLectureResult(client, cfg.CalendarURL, now, openCurrentLectureWorkspace)
		if err != nil {
			return err
		}
		if result.Material != nil && strings.TrimSpace(result.Material.URL) != "" {
			resource := moodle.Resource{
				ID:       result.Material.ID,
				Name:     result.Material.Label,
				URL:      result.Material.URL,
				Type:     "resource",
				FileType: result.Material.FileType,
			}
			output, err := openResolvedResource(client, resource)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, output, func(w io.Writer) error {
				return nil
			})
		}
		target, err := currentLectureOpenTarget(result)
		if err != nil {
			return err
		}
		if err := openURL(target); err != nil {
			return err
		}
		output := openCommandResult{
			Action:     "open",
			Target:     target,
			TargetType: "course_url",
		}
		if result.Course != nil {
			output.CourseID = fmt.Sprintf("%d", result.Course.ID)
		}
		return writeCommandOutput(cmd, output, func(w io.Writer) error {
			return nil
		})
	},
}

func init() {
	openCurrentLectureCmd.Flags().StringVar(&openCurrentLectureWorkspace, "workspace", "", "Optional workspace root for local file matching")
	openCurrentLectureCmd.Flags().StringVar(&openCurrentLectureAt, "at", "", "Override current time for testing (RFC3339)")
	openCmd.AddCommand(
		openCourseCmd,
		openCurrentLectureCmd,
		openResourceCmd,
	)
}

func runOpenResourceSelection(courseArg string, resourceArg string) (openCommandResult, error) {
	client, err := ensureAuthenticatedClient()
	if err != nil {
		return openCommandResult{}, err
	}
	courseID, err := resolveCourseIDWithOptions(client, courseArg, selectorOptions{})
	if err != nil {
		return openCommandResult{}, err
	}
	resources, _, err := client.FetchCourseResources(courseID)
	if err != nil {
		return openCommandResult{}, err
	}
	target, err := resolveResourceWithOptions(client, courseID, resources, resourceArg, selectorOptions{})
	if err != nil {
		return openCommandResult{}, err
	}
	result, err := openResolvedResource(client, *target)
	if err != nil {
		return openCommandResult{}, err
	}
	result.CourseID = courseID
	return result, nil
}

func currentLectureOpenTarget(result currentLectureResult) (string, error) {
	if result.Material != nil && strings.TrimSpace(result.Material.URL) != "" {
		return result.Material.URL, nil
	}
	if result.Course != nil && strings.TrimSpace(result.Course.URL) != "" {
		return result.Course.URL, nil
	}
	if result.Event == nil {
		return "", fmt.Errorf("no current or upcoming lecture found for today")
	}
	return "", fmt.Errorf("current lecture matched, but no openable material or course URL was found")
}

func findCourseByID(courses []moodle.Course, courseID string) (*moodle.Course, error) {
	for i := range courses {
		if fmt.Sprintf("%d", courses[i].ID) == courseID {
			return &courses[i], nil
		}
	}
	return nil, fmt.Errorf("course not found: %s", courseID)
}

func openResolvedResource(client *moodle.Client, resource moodle.Resource) (openCommandResult, error) {
	if strings.TrimSpace(resource.URL) == "" {
		return openCommandResult{}, fmt.Errorf("resource %s has no URL", resource.ID)
	}
	if resource.Type != "resource" {
		if err := openURL(resource.URL); err != nil {
			return openCommandResult{}, err
		}
		return openCommandResult{
			Action:     "open",
			Target:     resource.URL,
			TargetType: "resource_url",
			ResourceID: resource.ID,
		}, nil
	}
	path, err := downloadResourceToTempFile(client, resource)
	if err != nil {
		return openCommandResult{}, err
	}
	if err := openURL(path); err != nil {
		return openCommandResult{}, err
	}
	return openCommandResult{
		Action:     "open",
		Target:     path,
		TargetType: "local_file",
		ResourceID: resource.ID,
	}, nil
}

func downloadResourceToTempFile(client *moodle.Client, resource moodle.Resource) (string, error) {
	result, err := moodleDownloadFileToBuffer(client, resource.URL)
	if err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "moodle-open-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(tempDir, buildDownloadedResourceFilename(resource, result.ContentType))
	if err := os.WriteFile(path, result.Data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
