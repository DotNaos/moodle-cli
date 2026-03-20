package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var printRaw bool
var printCurrentLectureWorkspace string
var printCurrentLectureAt string

var printCmd = &cobra.Command{
	Use:              "print [course] [resource]",
	Short:            "Print Moodle content to stdout",
	Long:             "Print Moodle content to stdout.\n\nUse either the existing subcommands or direct selectors such as `moodle print current current`.",
	TraverseChildren: true,
	Example:          "  moodle print current current\n  moodle print 0 0\n  moodle print course 12345 67890",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if len(args) != 2 {
			return fmt.Errorf("expected either a subcommand or exactly 2 arguments: <course> <resource>")
		}
		return nil
	},
	ValidArgsFunction: completePrintCourseFile,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runPrintSelection(args[0], args[1])
	},
}

var printCourseCmd = &cobra.Command{
	Use:               "course <course-id|name|current|0> <resource-id|name|current|0>",
	Short:             "Print file contents to stdout (PDFs use OCR fallback)",
	Long:              "Print a single file's contents to stdout.\n\nThe course and file can be specified by ID, name, `current`, `0`, or a positive index.\nPDFs are converted to text and automatically fall back to OCR when native extraction looks poor.\nUse --raw to skip cleanup.",
	Example:           "  moodle print course 12345 67890\n  moodle print course current current\n  moodle print course 0 1",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completePrintCourseFile,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrintSelection(args[0], args[1])
	},
}

var printCurrentLectureCmd = &cobra.Command{
	Use:   "current-lecture",
	Short: "Print the best matching material for the current lecture",
	Long: "Resolve the current lecture from the timetable and print the best matching material to stdout.\n\n" +
		"This uses the same current-lecture selection as `moodle list current-lecture` and `moodle open current-lecture`.",
	Example: "  moodle print current-lecture\n" +
		"  moodle print current-lecture --workspace /Users/oli/school\n" +
		"  moodle print current-lecture --at 2026-03-20T11:15:00+01:00",
	Args: cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		now, err := resolveLectureTimeAt(printCurrentLectureAt)
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
		result, err := buildCurrentLectureResult(client, cfg.CalendarURL, now, printCurrentLectureWorkspace)
		if err != nil {
			return err
		}
		if result.Material == nil || strings.TrimSpace(result.Material.URL) == "" {
			if result.Event == nil {
				return fmt.Errorf("no current or upcoming lecture found for today")
			}
			return fmt.Errorf("current lecture matched, but no printable material was found")
		}
		text, err := renderDownloadedResource(client, result.Material.URL, result.Material.FileType, printRaw)
		if err != nil {
			return err
		}
		fmt.Println(text)
		return nil
	},
}

func init() {
	printCmd.PersistentFlags().BoolVar(&printRaw, "raw", false, "Print raw PDF text without cleanup")
	printCurrentLectureCmd.Flags().StringVar(&printCurrentLectureWorkspace, "workspace", "", "Optional workspace root for local file matching")
	printCurrentLectureCmd.Flags().StringVar(&printCurrentLectureAt, "at", "", "Override current time for testing (RFC3339)")
	printCmd.AddCommand(
		printCourseCmd,
		printCurrentLectureCmd,
	)
}

func runPrintSelection(courseArg string, resourceArg string) error {
	client, err := ensureAuthenticatedClient()
	if err != nil {
		return err
	}

	courseID, err := resolveCourseIDWithOptions(client, courseArg, selectorOptions{})
	if err != nil {
		return err
	}
	resources, _, err := client.FetchCourseResources(courseID)
	if err != nil {
		return err
	}
	target, err := resolveResourceWithOptions(client, courseID, resources, resourceArg, selectorOptions{})
	if err != nil {
		return err
	}
	if target.Type != "resource" {
		return fmt.Errorf("resource %s is not a file", target.ID)
	}

	text, err := renderDownloadedResource(client, target.URL, target.FileType, printRaw)
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}

func renderDownloadedResource(client *moodle.Client, url string, fileType string, raw bool) (string, error) {
	result, err := client.DownloadFileToBuffer(url)
	if err != nil {
		return "", err
	}
	if fileType == "pdf" || strings.Contains(strings.ToLower(result.ContentType), "pdf") {
		text, err := moodle.ExtractPDFText(result.Data)
		if err != nil {
			return "", err
		}
		if !raw {
			text = cleanExtractedTextWithTimeout(text, 2*time.Second)
		}
		return text, nil
	}
	return string(result.Data), nil
}

func cleanExtractedTextWithTimeout(input string, timeout time.Duration) string {
	type cleaningResult struct {
		text string
	}
	done := make(chan cleaningResult, 1)
	go func() {
		done <- cleaningResult{text: moodle.CleanExtractedText(input)}
	}()
	select {
	case result := <-done:
		return result.text
	case <-time.After(timeout):
		return strings.TrimSpace(input)
	}
}
