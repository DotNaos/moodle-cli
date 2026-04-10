package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestValidateOutputFlagsRejectsConflicts(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true
	outputYAML = true

	err := validateOutputFlags()
	if err == nil {
		t.Fatal("expected conflicting output flags to fail")
	}
	if !strings.Contains(err.Error(), "choose exactly one") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCurrentOutputFormatTreatsYMLAsYAML(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputYML = true
	if got := currentOutputFormat(); got != outputFormatYAML {
		t.Fatalf("expected yml alias to select yaml, got %s", got)
	}
}

func TestInteractiveCommandsRejectMachineOutput(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true

	for _, cmd := range []*cobra.Command{rootCmd, tuiCmd} {
		err := ensureMachineOutputAllowed(cmd)
		if err == nil {
			t.Fatalf("expected %s to reject machine output", cmd.CommandPath())
		}
		if !strings.Contains(err.Error(), "does not support machine-readable output") {
			t.Fatalf("unexpected error for %s: %v", cmd.CommandPath(), err)
		}
	}
}

func TestCommandsExposeGlobalMachineFlags(t *testing.T) {
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd != rootCmd {
			for _, flagName := range []string{"json", "yaml", "yml"} {
				if cmd.InheritedFlags().Lookup(flagName) == nil && cmd.Flags().Lookup(flagName) == nil {
					t.Fatalf("command %s is missing --%s", cmd.CommandPath(), flagName)
				}
			}
		}
		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}
	walk(rootCmd)
}

func TestVersionCommandOutputsJSON(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true
	var output bytes.Buffer
	versionCmd.SetOut(&output)
	t.Cleanup(func() { versionCmd.SetOut(nil) })

	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload versionResult
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON output, got %q (%v)", output.String(), err)
	}
	if payload.Version == "" {
		t.Fatalf("expected version field, got %#v", payload)
	}
}

func TestVersionCommandOutputsYAML(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputYAML = true
	var output bytes.Buffer
	versionCmd.SetOut(&output)
	t.Cleanup(func() { versionCmd.SetOut(nil) })

	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload versionResult
	if err := yaml.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("expected YAML output, got %q (%v)", output.String(), err)
	}
	if payload.Version == "" {
		t.Fatalf("expected version field, got %#v", payload)
	}
}

func TestServeStatusEventsWriteStructuredOutput(t *testing.T) {
	tests := []struct {
		name   string
		format outputFormat
		verify func(t *testing.T, data []byte)
	}{
		{
			name:   "json",
			format: outputFormatJSON,
			verify: func(t *testing.T, data []byte) {
				lines := strings.Split(strings.TrimSpace(string(data)), "\n")
				if len(lines) != 4 {
					t.Fatalf("expected 4 NDJSON lines, got %d: %q", len(lines), string(data))
				}
				var event serveEvent
				if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
					t.Fatalf("invalid json event: %v", err)
				}
				if event.Type != "starting" {
					t.Fatalf("unexpected event: %#v", event)
				}
			},
		},
		{
			name:   "yaml",
			format: outputFormatYAML,
			verify: func(t *testing.T, data []byte) {
				docs := strings.Split(strings.TrimSpace(string(data)), "---\n")
				count := 0
				for _, doc := range docs {
					if strings.TrimSpace(doc) == "" {
						continue
					}
					count++
					var event serveEvent
					if err := yaml.Unmarshal([]byte(doc), &event); err != nil {
						t.Fatalf("invalid yaml event: %v", err)
					}
					if count == 1 && event.Type != "starting" {
						t.Fatalf("unexpected first event: %#v", event)
					}
				}
				if count != 4 {
					t.Fatalf("expected 4 YAML event docs, got %d: %q", count, string(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset := saveOutputFlagState()
			defer reset()

			switch tt.format {
			case outputFormatJSON:
				outputJSON = true
			case outputFormatYAML:
				outputYAML = true
			}

			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&output)

			events := []serveEvent{
				{Type: "starting", Addr: ":8080"},
				{Type: "ready", Addr: "127.0.0.1:8080"},
				{Type: "shutdown", Signal: "terminated"},
				{Type: "fatal", Error: "boom"},
			}
			for _, event := range events {
				if err := emitServeStatus(cmd, event); err != nil {
					t.Fatalf("emitServeStatus(%s): %v", event.Type, err)
				}
			}

			tt.verify(t, output.Bytes())
		})
	}
}

func saveOutputFlagState() func() {
	prevJSON := outputJSON
	prevYAML := outputYAML
	prevYML := outputYML
	return func() {
		outputJSON = prevJSON
		outputYAML = prevYAML
		outputYML = prevYML
	}
}
