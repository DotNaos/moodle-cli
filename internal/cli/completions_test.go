package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
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
			name:           "after file and course, returns current selector",
			args:           []string{"file", "123"},
			expectContains: "current",
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

func TestCompleteOpenTargets(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectContains string
		expectEmpty    bool
	}{
		{
			name:           "no args returns course keyword",
			args:           []string{},
			expectContains: "course",
		},
		{
			name:           "no args returns resource keyword",
			args:           []string{},
			expectContains: "resource",
		},
		{
			name:        "after one arg returns empty",
			args:        []string{"resource"},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, directive := completeOpenTargets(nil, tt.args, "")

			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
			}

			if tt.expectEmpty {
				if len(results) != 0 {
					t.Errorf("expected empty results, got %v", results)
				}
				return
			}

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
		})
	}
}

func TestCompleteOpenResourceArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectEmpty bool
	}{
		{
			name:        "after course and resource returns empty",
			args:        []string{"123", "456"},
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, directive := completeOpenResourceArgs(nil, tt.args, "")

			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
			}

			if tt.expectEmpty && len(results) != 0 {
				t.Errorf("expected empty results, got %v", results)
			}
		})
	}
}

func TestCompleteCourseIDsIncludesCurrentSelectors(t *testing.T) {
	results, directive := completeCourseIDs(nil, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected no-file-comp directive, got %v", directive)
	}
	for _, want := range []string{"current", "0"} {
		found := false
		for _, result := range results {
			if len(result) >= len(want) && result[:len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %q in course completions, got %v", want, results)
		}
	}
}

func TestUpdateCurrentCourseCompletionDescriptions(t *testing.T) {
	out := []string{
		formatCompValue("current", "Current lecture course"),
		formatCompValue("0", "Current lecture course"),
	}
	client := &moodle.Client{}
	original := currentLectureResultForCompletion
	currentLectureResultForCompletion = func(client *moodle.Client) (currentLectureResult, error) {
		return currentLectureResult{
			Event:  &moodle.CalendarEvent{Summary: "Deep Learning"},
			Course: &currentLectureCourse{ID: 22585, Title: "Deep Learning (cds-108) FS26"},
		}, nil
	}
	defer func() { currentLectureResultForCompletion = original }()

	updateCurrentCourseCompletionDescriptions(out, client)

	if out[0] != "current\tDeep Learning -> Deep Learning (cds-108) FS26" {
		t.Fatalf("unexpected current completion: %q", out[0])
	}
	if out[1] != "0\tDeep Learning -> Deep Learning (cds-108) FS26" {
		t.Fatalf("unexpected zero completion: %q", out[1])
	}
}

func TestUpdateCurrentResourceCompletionDescriptions(t *testing.T) {
	out := []string{
		formatCompValue("current", "Current or top-ranked material"),
		formatCompValue("0", "Current or top-ranked material"),
	}
	result := currentLectureResult{
		Course:   &currentLectureCourse{ID: 22585, Title: "Deep Learning (cds-108) FS26"},
		Material: &currentLectureResource{ID: "948787", Label: "Einführungsfolien"},
	}

	updateCurrentResourceCompletionDescriptions(out, "22585", result)

	if out[0] != "current\tEinführungsfolien (current material)" {
		t.Fatalf("unexpected current resource completion: %q", out[0])
	}
	if out[1] != "0\tEinführungsfolien (current material)" {
		t.Fatalf("unexpected zero resource completion: %q", out[1])
	}
}

func TestBrowserOpenCommand(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		url      string
		wantErr  string
		wantPath string
		wantArgs []string
	}{
		{
			name:     "darwin uses open",
			goos:     "darwin",
			url:      "https://example.com/course",
			wantPath: "open",
			wantArgs: []string{"open", "https://example.com/course"},
		},
		{
			name:     "linux uses xdg-open",
			goos:     "linux",
			url:      "https://example.com/course",
			wantPath: "xdg-open",
			wantArgs: []string{"xdg-open", "https://example.com/course"},
		},
		{
			name:     "windows uses rundll32",
			goos:     "windows",
			url:      "https://example.com/course",
			wantPath: "rundll32",
			wantArgs: []string{"rundll32", "url.dll,FileProtocolHandler", "https://example.com/course"},
		},
		{
			name:    "unknown os errors",
			goos:    "plan9",
			url:     "https://example.com/course",
			wantErr: "opening URLs is not supported on plan9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := browserOpenCommand(tt.goos, tt.url)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertExecCommand(t, cmd, tt.wantPath, tt.wantArgs)
		})
	}
}

func TestFindCourseByID(t *testing.T) {
	courses := []moodle.Course{
		{ID: 42, Fullname: "Course A", ViewURL: "https://example.com/course/view.php?id=42"},
		{ID: 99, Fullname: "Course B", ViewURL: "https://example.com/course/view.php?id=99"},
	}

	course, err := findCourseByID(courses, "99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if course.ViewURL != "https://example.com/course/view.php?id=99" {
		t.Fatalf("expected course URL to be returned, got %q", course.ViewURL)
	}
}

func TestFindCourseByIDNotFound(t *testing.T) {
	_, err := findCourseByID([]moodle.Course{{ID: 42}}, "99")
	if err == nil || err.Error() != "course not found: 99" {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestResolveResourceReturnsURL(t *testing.T) {
	resources := []moodle.Resource{
		{ID: "folder-12", Name: "Slides", URL: "https://example.com/mod/folder/view.php?id=12"},
		{ID: "34", Name: "Sheet 1", URL: "https://example.com/mod/resource/view.php?id=34&redirect=1"},
	}

	resource, err := resolveResource(resources, "Slides")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resource.URL != "https://example.com/mod/folder/view.php?id=12" {
		t.Fatalf("expected resource URL to be returned, got %q", resource.URL)
	}
}

func TestValidArgsForCommandsWithPositionalArgs(t *testing.T) {
	// Commands that accept positional args must have ValidArgsFunction
	commandsWithArgs := []*cobra.Command{
		listCmd,
		filesCmd,    // list files <course-id|name>
		printCmd,    // print course <course-id|name> <resource-id|name>
		downloadCmd, // download file <course-id|name> <resource-id|name>
		exportCmd,   // export course <course-id|name>
		openCmd,
		openCourseCmd,
		openResourceCmd,
	}

	for _, cmd := range commandsWithArgs {
		t.Run(cmd.Name(), func(t *testing.T) {
			if cmd.ValidArgsFunction == nil {
				t.Errorf("command %q accepts positional args but has no ValidArgsFunction", cmd.Name())
			}
		})
	}
}

func assertExecCommand(t *testing.T, cmd *exec.Cmd, wantPath string, wantArgs []string) {
	t.Helper()

	if normalizeCommandName(cmd.Path) != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, cmd.Path)
	}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("expected args %v, got %v", wantArgs, cmd.Args)
	}
	for i := range wantArgs {
		got := cmd.Args[i]
		if i == 0 {
			got = normalizeCommandName(got)
		}
		if got != wantArgs[i] {
			t.Fatalf("expected args %v, got %v", wantArgs, cmd.Args)
		}
	}
}

func normalizeCommandName(value string) string {
	base := filepath.Base(value)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	return strings.ToLower(base)
}
