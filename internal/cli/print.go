package cli

import (
	"fmt"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

var printRaw bool

var printCmd = &cobra.Command{
	Use:               "print course <course-id|name> <resource-id|name>",
	Short:             "Print file contents to stdout (PDFs use OCR fallback)",
	Long:              "Print a single file's contents to stdout.\n\nThe course and file can be specified by ID or name.\nPDFs are converted to text and automatically fall back to OCR when native extraction looks poor.\nUse --raw to skip cleanup.",
	Example:           "  moodle print course 12345 67890\n  moodle print course \"Mathematik II (cds-402) FS25\" \"Übungsblatt Analysis 1\"",
	Args:              cobra.ExactArgs(3),
	ValidArgsFunction: completePrintCourseFile,
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "course" {
			return fmt.Errorf("expected 'course' subcommand")
		}
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		courseID, err := resolveCourseID(client, args[1])
		if err != nil {
			return err
		}
		resources, _, err := client.FetchCourseResources(courseID)
		if err != nil {
			return err
		}
		target, err := resolveResource(resources, args[2])
		if err != nil {
			return err
		}
		if target.Type != "resource" {
			return fmt.Errorf("resource %s is not a file", target.ID)
		}

		result, err := client.DownloadFileToBuffer(target.URL)
		if err != nil {
			return err
		}

		if target.FileType == "pdf" || strings.Contains(strings.ToLower(result.ContentType), "pdf") {
			text, err := moodle.ExtractPDFText(result.Data)
			if err != nil {
				return err
			}
			if !printRaw {
				text = moodle.CleanExtractedText(text)
			}
			fmt.Println(text)
			return nil
		}

		fmt.Println(string(result.Data))
		return nil
	},
}

func init() {
	printCmd.Flags().BoolVar(&printRaw, "raw", false, "Print raw PDF text without cleanup")
}
