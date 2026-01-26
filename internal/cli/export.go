package cli

import (
  "fmt"

  "github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
  Use:   "export course <id>",
  Short: "Export a course",
  Args:  cobra.MinimumNArgs(2),
  RunE: func(cmd *cobra.Command, args []string) error {
    if args[0] != "course" {
      return fmt.Errorf("expected 'course' subcommand")
    }
    fmt.Printf("export course %s (format=%s): not implemented yet\n", args[1], exportFormat)
    return nil
  },
}

func init() {
  exportCmd.Flags().StringVar(&exportFormat, "format", "folder", "Export format: folder|zip")
}
