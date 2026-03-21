package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var navJSON bool
var navOpen bool
var navPrint bool
var navWorkspace string
var navAt string

var navCmd = &cobra.Command{
	Use:   "nav <path>",
	Short: "Resolve a Moodle navigation path",
	Long: "Resolve a slash-separated Moodle navigation path without starting the interactive TUI.\n\n" +
		"Examples:\n" +
		"  moodle nav current\n" +
		"  moodle nav current/items/current\n" +
		"  moodle nav semesters/FS26/courses/1/sections/1/items/1 --open",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeNavPath,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}
		service, err := newNavService(client, selectorOptions{Workspace: navWorkspace, At: navAt})
		if err != nil {
			return err
		}
		path := strings.TrimSpace(args[0])
		node, err := service.ResolvePath(path)
		if err != nil {
			return err
		}
		if navOpen {
			return service.Open(node)
		}
		if navPrint {
			text, err := service.Print(node)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		}
		summary, err := service.Summary(path, node)
		if err != nil {
			return err
		}
		if navJSON {
			fmt.Println(renderNavSummary(summary))
			return nil
		}
		fmt.Printf("%s (%s)\n", summary.Title, summary.Kind)
		if summary.Subtitle != "" {
			fmt.Println(summary.Subtitle)
		}
		if summary.Preview != "" {
			fmt.Println(summary.Preview)
		}
		if len(summary.Children) == 0 {
			return nil
		}
		fmt.Println("Children:")
		for _, child := range summary.Children {
			line := fmt.Sprintf("%d. %s", child.Index, child.Title)
			if child.Subtitle != "" {
				line += " — " + child.Subtitle
			}
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	navCmd.Flags().BoolVar(&navJSON, "json", false, "Output JSON")
	navCmd.Flags().BoolVar(&navOpen, "open", false, "Open the resolved node if possible")
	navCmd.Flags().BoolVar(&navPrint, "print", false, "Print the resolved node if possible")
	navCmd.Flags().StringVar(&navWorkspace, "workspace", "", "Optional workspace root for current-course helpers")
	navCmd.Flags().StringVar(&navAt, "at", "", "Override current time for testing (RFC3339)")
}

func completeNavPath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{
		formatCompValue("current", "Current lecture view"),
		formatCompValue("today", "Today’s timetable"),
		formatCompValue("semesters", "Semester browser"),
	}, cobra.ShellCompDirectiveNoFileComp
}
