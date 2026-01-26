package cli

import (
  "encoding/json"
  "errors"
  "fmt"

  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

var filesJSON bool

var filesCmd = &cobra.Command{
  Use:   "files <course-id>",
  Short: "List files for a course",
  Args:  cobra.ExactArgs(1),
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

    resources, _, err := client.FetchCourseResources(args[0])
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
