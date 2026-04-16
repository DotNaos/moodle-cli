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
		if shouldExposeCommandAsAPI(cmd) {
			key := strings.Join(commandNamePath(cmd), " ")
			if _, ok := routeSet[key]; !ok {
				t.Fatalf("command %s is missing an API endpoint", cmd.CommandPath())
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

func commandNamePath(cmd *cobra.Command) []string {
	names := []string{}
	for current := cmd; current != nil && current != rootCmd; current = current.Parent() {
		names = append([]string{current.Name()}, names...)
	}
	return names
}
