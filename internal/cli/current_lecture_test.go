package cli

import (
	"testing"
	"time"

	"github.com/DotNaos/moodle-cli/internal/moodle"
)

func TestSelectCurrentLectureEventPrefersActive(t *testing.T) {
	now := time.Date(2026, 3, 20, 9, 30, 0, 0, time.FixedZone("CET", 3600))
	events := []moodle.CalendarEvent{
		{Summary: "Earlier", Start: now.Add(-2 * time.Hour), End: now.Add(-90 * time.Minute)},
		{Summary: "Active", Start: now.Add(-15 * time.Minute), End: now.Add(30 * time.Minute)},
		{Summary: "Later", Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour)},
	}
	event := selectCurrentLectureEvent(events, now)
	if event == nil || event.Summary != "Active" {
		t.Fatalf("expected active event, got %#v", event)
	}
}

func TestSelectCurrentLectureEventFallsBackToNextToday(t *testing.T) {
	now := time.Date(2026, 3, 20, 8, 0, 0, 0, time.FixedZone("CET", 3600))
	events := []moodle.CalendarEvent{
		{Summary: "Next", Start: now.Add(45 * time.Minute), End: now.Add(2 * time.Hour)},
		{Summary: "Tomorrow", Start: now.Add(24 * time.Hour), End: now.Add(25 * time.Hour)},
	}
	event := selectCurrentLectureEvent(events, now)
	if event == nil || event.Summary != "Next" {
		t.Fatalf("expected next event today, got %#v", event)
	}
}

func TestMatchCourseForLecture(t *testing.T) {
	courses := []moodle.Course{
		{ID: 1, Fullname: "High Performance Computing (cds-301) FS26"},
		{ID: 2, Fullname: "Algorithmen des wissenschaftlichen Rechnens (cds-116) FS26"},
	}
	course, matched := matchCourseForLecture(courses, "Algorithmen des wissenschaftlichen Rechnens")
	if course == nil || course.ID != 2 {
		t.Fatalf("expected course 2, got %#v", course)
	}
	if matched == "" {
		t.Fatalf("expected a non-empty matched title")
	}
}

func TestRankCurrentLectureResourcesPrefersLectureSlides(t *testing.T) {
	resources := []moodle.Resource{
		{Name: "Folien Teil 1", Type: "resource", FileType: "pdf", SectionName: "Thema 1"},
		{Name: "Aufgabenblatt 01", Type: "resource", FileType: "pdf", SectionName: "Thema 1"},
		{Name: "Folien Teil 2", Type: "resource", FileType: "pdf", SectionName: "Thema 2"},
		{Name: "Datei: papa.png", Type: "resource", FileType: "png", SectionName: "Thema 2"},
		{Name: "Aufgabenblatt 03", Type: "resource", FileType: "pdf", SectionName: "Thema 2"},
	}
	ranked := rankCurrentLectureResources(resources, map[string]string{})
	if len(ranked) != 5 {
		t.Fatalf("expected 5 ranked resources, got %d", len(ranked))
	}
	expected := []string{"Folien Teil 2", "Datei: papa.png", "Aufgabenblatt 03", "Folien Teil 1", "Aufgabenblatt 01"}
	for index, label := range expected {
		if ranked[index].Label != label {
			t.Fatalf("expected rank %d to be %q, got %q", index, label, ranked[index].Label)
		}
	}
}
