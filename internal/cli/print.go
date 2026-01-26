package cli

import (
  "errors"
  "fmt"
  "strings"

  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

var printRaw bool

var printCmd = &cobra.Command{
  Use:   "print <course-id> <resource-id>",
  Short: "Print file contents (PDFs extracted to text)",
  Args:  cobra.ExactArgs(2),
  ValidArgsFunction: completeCourseOrResourceIDs,
  RunE: func(cmd *cobra.Command, args []string) error {
    session, err := moodle.LoadSession(opts.SessionPath)
    if err != nil {
      return fmt.Errorf("load session: %w", err)
    }
    client, err := moodle.NewClient(session)
    if err != nil {
      return err
    }
    if err := client.ValidateSession(); err != nil {
      if errors.Is(err, moodle.ErrSessionExpired) {
        return fmt.Errorf("session expired, please run 'moodle login' again")
      }
      return err
    }

    courseID := args[0]
    resourceID := args[1]
    resources, _, err := client.FetchCourseResources(courseID)
    if err != nil {
      return err
    }

    var target *moodle.Resource
    for i := range resources {
      if resources[i].ID == resourceID {
        target = &resources[i]
        break
      }
    }
    if target == nil {
      return fmt.Errorf("resource not found: %s", resourceID)
    }
    if target.Type != "resource" {
      return fmt.Errorf("resource %s is not a file", resourceID)
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
