package cli

import (
	"testing"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	"github.com/spf13/cobra"
)

func TestAllCommandsHaveCompletions(t *testing.T) {
	var check func(cmd *cobra.Command)
	check = func(cmd *cobra.Command) {
		// Skip root command itself, only check subcommands
		if cmd.Parent() != nil && cmd.ValidArgsFunction == nil && !cmd.HasSubCommands() {
			t.Errorf("command %q missing ValidArgsFunction", cmd.CommandPath())
		}
		for _, sub := range cmd.Commands() {
			check(sub)
		}
	}
	check(rootCmd)
}

func TestFlagCompletionsRegistered(t *testing.T) {
	tests := []struct {
		cmdPath  string
		flagName string
		cmd      *cobra.Command
	}{
		{"moodle login", "school", loginCmd},
		{"moodle config set", "school", configSetCmd},
	}

	for _, tt := range tests {
		t.Run(tt.cmdPath+" --"+tt.flagName, func(t *testing.T) {
			flag := tt.cmd.Flag(tt.flagName)
			if flag == nil {
				t.Fatalf("flag %q not found on command %q", tt.flagName, tt.cmdPath)
			}

			completionFunc, found := tt.cmd.GetFlagCompletionFunc(tt.flagName)
			if !found || completionFunc == nil {
				t.Errorf("flag %q on command %q has no completion function registered", tt.flagName, tt.cmdPath)
				return
			}

			results, _ := completionFunc(tt.cmd, nil, "")
			if len(results) == 0 {
				t.Errorf("flag %q on command %q completion returned no results", tt.flagName, tt.cmdPath)
			}
		})
	}
}

func TestCompleteSchoolIDs(t *testing.T) {
	results, directive := completeSchoolIDs(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	if len(results) != len(moodle.Schools) {
		t.Errorf("expected %d schools, got %d", len(moodle.Schools), len(results))
	}

	// Verify each school ID is in the results
	for _, school := range moodle.Schools {
		found := false
		for _, r := range results {
			if len(r) >= len(school.ID) && r[:len(school.ID)] == school.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("school %q not found in completions", school.ID)
		}
	}
}

func TestCompleteDownloadFile(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectContains string
		expectEmpty    bool
	}{
		{
			name:           "no args returns file keyword",
			args:           []string{},
			expectContains: "file",
		},
		{
			name:        "after file and course, returns empty",
			args:        []string{"file", "123"},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, directive := completeDownloadFile(nil, tt.args, "")

			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
			}

			if tt.expectEmpty {
				if len(results) != 0 {
					t.Errorf("expected empty results, got %v", results)
				}
				return
			}

			if tt.expectContains != "" {
				found := false
				for _, r := range results {
					if len(r) >= len(tt.expectContains) && r[:len(tt.expectContains)] == tt.expectContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %q in results, got %v", tt.expectContains, results)
				}
			}
		})
	}
}

func TestValidArgsForCommandsWithPositionalArgs(t *testing.T) {
	// Commands that accept positional args must have ValidArgsFunction
	commandsWithArgs := []*cobra.Command{
		filesCmd,    // list files <course-id|name>
		printCmd,    // print course <course-id|name> <resource-id|name>
		downloadCmd, // download file <course-id|name> <resource-id|name>
		exportCmd,   // export course <course-id|name>
	}

	for _, cmd := range commandsWithArgs {
		t.Run(cmd.Name(), func(t *testing.T) {
			if cmd.ValidArgsFunction == nil {
				t.Errorf("command %q accepts positional args but has no ValidArgsFunction", cmd.Name())
			}
		})
	}
}
