package cli

import (
	"testing"
	"time"

	"github.com/DotNaos/moodle-services/internal/moodle"
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

func TestRankCurrentLectureResourcesOrdersByNewestTimestamp(t *testing.T) {
	resources := []moodle.Resource{
		{Name: "Folien Teil 1", Type: "resource", FileType: "pdf", UploadedAt: "2026-03-05T20:49:00+01:00"},
		{Name: "Aufgabenblatt 01", Type: "resource", FileType: "pdf"},
		{Name: "Folien Teil 2", Type: "resource", FileType: "pdf", UploadedAt: "2026-03-19T22:00:00+01:00"},
		{Name: "Datei: papa.png", Type: "resource", FileType: "png", UploadedAt: "2026-03-19T21:37:00+01:00"},
		{Name: "Aufgabenblatt 03", Type: "resource", FileType: "pdf", UploadedAt: "2026-03-19T21:35:00+01:00"},
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

func TestSelectBestCurrentLectureMaterialPrefersNewestLecturePDF(t *testing.T) {
	resources := []currentLectureResource{
		{Label: "Data Augmentation", FileType: "pdf", Kind: "other", UploadedAt: "2026-03-20T15:30:00+01:00"},
		{Label: "Einführungsfolien", FileType: "pdf", Kind: "lecture", UploadedAt: "2026-02-16T18:03:00+01:00"},
		{Label: "CNN", FileType: "pdf", Kind: "other", UploadedAt: "2026-03-20T15:20:00+01:00"},
	}
	best := selectBestCurrentLectureMaterial(resources)
	if best == nil || best.Label != "Einführungsfolien" {
		t.Fatalf("expected newest lecture pdf to be chosen, got %#v", best)
	}
}
