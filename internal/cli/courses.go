package cli

import (
  "encoding/json"
  "errors"
  "fmt"

  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

var coursesJSON bool

var coursesCmd = &cobra.Command{
  Use:   "courses",
  Short: "List courses",
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
