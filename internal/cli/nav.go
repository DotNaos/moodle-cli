package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

var navOpen bool
var navPrint bool
var navWorkspace string
var navAt string

type navActionResult struct {
	Path   string     `json:"path" yaml:"path"`
	Action string     `json:"action" yaml:"action"`
	Node   navSummary `json:"node" yaml:"node"`
	Target string     `json:"target,omitempty" yaml:"target,omitempty"`
	Text   string     `json:"text,omitempty" yaml:"text,omitempty"`
}

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
		summary, err := service.Summary(path, node)
		if err != nil {
			return err
		}
		if navOpen {
			target, err := service.Open(node)
			if err != nil {
				return err
			}
			result := navActionResult{
				Path:   path,
				Action: "open",
				Node:   summary,
				Target: target,
			}
			return writeCommandOutput(cmd, result, func(w io.Writer) error {
				return nil
			})
		}
		if navPrint {
			text, err := service.Print(node)
			if err != nil {
				return err
			}
			result := navActionResult{
				Path:   path,
				Action: "print",
				Node:   summary,
				Text:   text,
			}
			return writeCommandOutput(cmd, result, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, text)
				return err
			})
		}
		return writeCommandOutput(cmd, summary, func(w io.Writer) error {
			if _, err := fmt.Fprintf(w, "%s (%s)\n", summary.Title, summary.Kind); err != nil {
				return err
			}
			if summary.Subtitle != "" {
				if _, err := fmt.Fprintln(w, summary.Subtitle); err != nil {
					return err
				}
			}
			if summary.Preview != "" {
				if _, err := fmt.Fprintln(w, summary.Preview); err != nil {
					return err
				}
			}
			if len(summary.Children) == 0 {
				return nil
			}
			if _, err := fmt.Fprintln(w, "Children:"); err != nil {
				return err
			}
			for _, child := range summary.Children {
				line := fmt.Sprintf("%d. %s", child.Index, child.Title)
				if child.Subtitle != "" {
					line += " — " + child.Subtitle
				}
				if _, err := fmt.Fprintln(w, line); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func init() {
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
