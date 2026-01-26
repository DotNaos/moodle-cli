package cli

import (
  "fmt"

  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

var deadlinesJSON bool

var deadlinesCmd = &cobra.Command{
  Use:   "deadlines",
  Short: "List deadlines",
  RunE: func(cmd *cobra.Command, args []string) error {
    session, err := moodle.LoadSession(opts.SessionPath)
    if err != nil {
      return fmt.Errorf("load session: %w", err)
    }
    if deadlinesJSON {
      fmt.Println("[]")
      return nil
    }
    fmt.Printf("deadlines: not implemented yet (school=%s)\n", session.SchoolID)
    return nil
  },
}

func init() {
  deadlinesCmd.Flags().BoolVar(&deadlinesJSON, "json", false, "Output JSON")
}
