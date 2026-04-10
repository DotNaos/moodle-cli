package cli

import (
	"path"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/api"
	"github.com/spf13/cobra"
)

func buildAPICommandRoutes() []api.CommandRoute {
	routes := []api.CommandRoute{}
	walkAPICommands(rootCmd, []string{}, &routes)
	return routes
}

func walkAPICommands(cmd *cobra.Command, names []string, routes *[]api.CommandRoute) {
	if cmd == nil || !cmd.IsAvailableCommand() {
		return
	}

	currentNames := names
	if cmd != rootCmd {
		currentNames = append(append([]string{}, names...), cmd.Name())
		if shouldExposeCommandAsAPI(cmd) {
			*routes = append(*routes, api.CommandRoute{
				APIPath:     "/api/cli/" + path.Join(currentNames...),
				CommandPath: currentNames,
				Summary:     commandSummary(cmd),
				Description: commandDescription(cmd),
				Stream:      isStreamingAPICommandPath(currentNames),
			})
		}
	}

	for _, sub := range cmd.Commands() {
		walkAPICommands(sub, currentNames, routes)
	}
}

func shouldExposeCommandAsAPI(cmd *cobra.Command) bool {
	if cmd == nil || cmd == rootCmd {
		return false
	}
	if isInteractiveOnly(cmd) {
		return false
	}
	return cmd.RunE != nil || cmd.Run != nil
}

func isStreamingAPICommandPath(commandPath []string) bool {
	return len(commandPath) == 1 && strings.EqualFold(commandPath[0], "serve")
}

func commandSummary(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	if strings.TrimSpace(cmd.Short) != "" {
		return cmd.Short
	}
	return "Run " + cmd.CommandPath()
}

func commandDescription(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	if strings.TrimSpace(cmd.Long) != "" {
		return cmd.Long
	}
	return cmd.Short
}
