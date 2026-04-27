package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestEveryMachineCommandHasAPIEndpoint(t *testing.T) {
	routes := buildAPICommandRoutes()
	routeSet := map[string]struct{}{}
	for _, route := range routes {
		routeSet[strings.Join(route.CommandPath, " ")] = struct{}{}
	}

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd == nil || !cmd.IsAvailableCommand() {
			return
		}
		key := strings.Join(commandNamePath(cmd), " ")
		if shouldExposeCommandAsAPI(cmd) {
			if _, ok := routeSet[key]; !ok {
				t.Fatalf("command %s is missing an API endpoint", cmd.CommandPath())
			}
		}
		if isRunnableAPIOptionalCommand(cmd) {
			if _, ok := routeSet[key]; ok {
				t.Fatalf("command %s is explicitly marked as API-optional but still has an API endpoint", cmd.CommandPath())
			}
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}

	walk(rootCmd)
}

func TestAPIRouteCountMatchesMachineCommandCount(t *testing.T) {
	expected := 0
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd == nil || !cmd.IsAvailableCommand() {
			return
		}
		if shouldExposeCommandAsAPI(cmd) {
			expected++
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)

	routes := buildAPICommandRoutes()
	if len(routes) != expected {
		t.Fatalf("expected %d API command routes, got %d", expected, len(routes))
	}
}

func TestLogsRouteUsesStreamingAPI(t *testing.T) {
	for _, route := range buildAPICommandRoutes() {
		if strings.Join(route.CommandPath, " ") != "logs" {
			continue
		}
		if !route.Stream {
			t.Fatalf("expected logs route to be marked as streaming: %#v", route)
		}
		return
	}
	t.Fatal("logs route not found")
}

func TestAPIOptionalCommandsAreExplicitlyExcluded(t *testing.T) {
	for _, excluded := range []string{
		"completion",
		"completion bash",
		"completion fish",
		"completion powershell",
		"completion zsh",
		"serve",
	} {
		cmd, _, err := rootCmd.Find(strings.Split(excluded, " "))
		if err != nil {
			t.Fatalf("find %q: %v", excluded, err)
		}
		if cmd == nil {
			t.Fatalf("command %q not found", excluded)
		}
		if !isAPIOptional(cmd) {
			t.Fatalf("expected %q to be explicitly marked API-optional", excluded)
		}
		if shouldExposeCommandAsAPI(cmd) {
			t.Fatalf("expected %q to stay out of generated API routes", excluded)
		}
	}
}

func commandNamePath(cmd *cobra.Command) []string {
	names := []string{}
	for current := cmd; current != nil && current != rootCmd; current = current.Parent() {
		names = append([]string{current.Name()}, names...)
	}
	return names
}

func isRunnableAPIOptionalCommand(cmd *cobra.Command) bool {
	return cmd != nil &&
		cmd != rootCmd &&
		cmd.IsAvailableCommand() &&
		!isInteractiveOnly(cmd) &&
		isAPIOptional(cmd) &&
		(cmd.RunE != nil || cmd.Run != nil)
}
