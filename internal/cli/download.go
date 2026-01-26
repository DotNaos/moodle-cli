package cli

import (
  "fmt"

  "github.com/spf13/cobra"
)

var downloadZip bool
var downloadFiles bool

var downloadCmd = &cobra.Command{
  Use:   "download course <id>",
  Short: "Download a course",
  Args:  cobra.MinimumNArgs(2),
  RunE: func(cmd *cobra.Command, args []string) error {
    if args[0] != "course" {
      return fmt.Errorf("expected 'course' subcommand")
    }
    if downloadZip && downloadFiles {
      return fmt.Errorf("choose either --zip or --files")
    }
    fmt.Printf("download course %s: not implemented yet\n", args[1])
    return nil
  },
}

func init() {
  downloadCmd.Flags().BoolVar(&downloadZip, "zip", false, "Download as zip")
  downloadCmd.Flags().BoolVar(&downloadFiles, "files", false, "Download as files")
}
