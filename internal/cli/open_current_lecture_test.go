package cli

import "testing"

func TestCurrentLectureOpenTargetPrefersMaterial(t *testing.T) {
	result := currentLectureResult{
		Material: &currentLectureResource{URL: "https://example.com/resource"},
		Course:   &currentLectureCourse{URL: "https://example.com/course"},
	}
	url, err := currentLectureOpenTarget(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/resource" {
		t.Fatalf("expected resource URL, got %q", url)
	}
}

func TestCurrentLectureOpenTargetFallsBackToCourse(t *testing.T) {
	result := currentLectureResult{
		Course: &currentLectureCourse{URL: "https://example.com/course"},
	}
	url, err := currentLectureOpenTarget(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/course" {
		t.Fatalf("expected course URL, got %q", url)
	}
}
