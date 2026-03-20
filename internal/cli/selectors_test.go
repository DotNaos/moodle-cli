package cli

import (
	"testing"

	"github.com/DotNaos/moodle-cli/internal/moodle"
)

func TestResolveCourseIDFromCoursesWithCurrentSupportsCurrentAndIndices(t *testing.T) {
	courses := []moodle.Course{
		{ID: 42, Fullname: "Course A"},
		{ID: 99, Fullname: "Course B"},
	}
	current := &currentLectureCourse{ID: 99}

	tests := []struct {
		input string
		want  string
	}{
		{input: "current", want: "99"},
		{input: "0", want: "99"},
		{input: "1", want: "42"},
		{input: "2", want: "99"},
	}

	for _, tt := range tests {
		got, err := resolveCourseIDFromCoursesWithCurrent(courses, tt.input, current)
		if err != nil {
			t.Fatalf("input %q returned error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("input %q expected %q, got %q", tt.input, tt.want, got)
		}
	}
}

func TestResolveCourseIDFromCoursesWithCurrentRejectsOutOfRangeIndex(t *testing.T) {
	courses := []moodle.Course{{ID: 42, Fullname: "Course A"}}
	_, err := resolveCourseIDFromCoursesWithCurrent(courses, "2", nil)
	if err == nil || err.Error() != "course index out of range: 2" {
		t.Fatalf("expected out-of-range error, got %v", err)
	}
}

func TestResolveResourceWithCurrentSupportsCurrentAndIndices(t *testing.T) {
	resources := []moodle.Resource{
		{ID: "folder-1", Name: "Folder", Type: "folder"},
		{ID: "10", Name: "Folien Teil 2", Type: "resource", FileType: "pdf", SectionName: "Thema 2"},
		{ID: "11", Name: "Datei: papa.png", Type: "resource", FileType: "png", SectionName: "Thema 2"},
		{ID: "12", Name: "Aufgabenblatt 03", Type: "resource", FileType: "pdf", SectionName: "Thema 2"},
	}

	current, err := resolveResourceWithCurrent(resources, "current", "12")
	if err != nil {
		t.Fatalf("current selector returned error: %v", err)
	}
	if current.ID != "12" {
		t.Fatalf("expected current material id 12, got %s", current.ID)
	}

	first, err := resolveResourceWithCurrent(resources, "1", "")
	if err != nil {
		t.Fatalf("index 1 returned error: %v", err)
	}
	if first.ID != "10" {
		t.Fatalf("expected first file resource id 10, got %s", first.ID)
	}

	second, err := resolveResourceWithCurrent(resources, "2", "")
	if err != nil {
		t.Fatalf("index 2 returned error: %v", err)
	}
	if second.ID != "11" {
		t.Fatalf("expected second file resource id 11, got %s", second.ID)
	}
}

func TestParentCommandsAcceptDirectSelectors(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "list", fn: func() error { return listCmd.Args(listCmd, []string{"0", "0"}) }},
		{name: "print", fn: func() error { return printCmd.Args(printCmd, []string{"0", "0"}) }},
		{name: "open", fn: func() error { return openCmd.Args(openCmd, []string{"0", "0"}) }},
	}

	for _, tt := range tests {
		if err := tt.fn(); err != nil {
			t.Fatalf("%s command rejected direct selectors: %v", tt.name, err)
		}
	}
}
