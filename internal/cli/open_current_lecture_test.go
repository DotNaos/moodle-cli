package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/DotNaos/moodle-services/internal/moodle"
)

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

func TestDownloadResourceToTempFileWritesNamedFile(t *testing.T) {
	client := &moodle.Client{}
	resource := moodle.Resource{
		ID:       "1",
		Name:     "Folien Teil 2",
		URL:      "https://example.com/mod/resource/view.php?id=1&redirect=1",
		Type:     "resource",
		FileType: "pdf",
	}

	original := moodleDownloadFileToBuffer
	moodleDownloadFileToBuffer = func(client *moodle.Client, url string) (moodle.DownloadResult, error) {
		return moodle.DownloadResult{Data: []byte("hello"), ContentType: "application/pdf"}, nil
	}
	defer func() { moodleDownloadFileToBuffer = original }()

	path, err := downloadResourceToTempFile(client, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "Folien Teil 2.pdf") {
		t.Fatalf("expected pdf filename, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected temp file to exist: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected downloaded contents, got %q", string(data))
	}
}

func TestDownloadResourceToTempFileInfersExtensionFromContentType(t *testing.T) {
	client := &moodle.Client{}
	resource := moodle.Resource{
		ID:   "2",
		Name: "Folien Teil 1",
		URL:  "https://example.com/mod/resource/view.php?id=2&redirect=1",
		Type: "resource",
	}

	original := moodleDownloadFileToBuffer
	moodleDownloadFileToBuffer = func(client *moodle.Client, url string) (moodle.DownloadResult, error) {
		return moodle.DownloadResult{Data: []byte("hello"), ContentType: "application/pdf"}, nil
	}
	defer func() { moodleDownloadFileToBuffer = original }()

	path, err := downloadResourceToTempFile(client, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "Folien Teil 1.pdf") {
		t.Fatalf("expected inferred pdf filename, got %q", path)
	}
}

func TestDownloadResourceToTempFileIgnoresPseudoExtensionFromTitle(t *testing.T) {
	client := &moodle.Client{}
	resource := moodle.Resource{
		ID:       "3",
		Name:     "Folien Teil 1 (Update 05.03.26)",
		URL:      "https://example.com/mod/resource/view.php?id=3&redirect=1",
		Type:     "resource",
		FileType: "pdf",
	}

	original := moodleDownloadFileToBuffer
	moodleDownloadFileToBuffer = func(client *moodle.Client, url string) (moodle.DownloadResult, error) {
		return moodle.DownloadResult{Data: []byte("hello"), ContentType: "application/pdf"}, nil
	}
	defer func() { moodleDownloadFileToBuffer = original }()

	path, err := downloadResourceToTempFile(client, resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "Folien Teil 1 (Update 05.03.26).pdf") {
		t.Fatalf("expected pdf filename despite dotted title, got %q", path)
	}
}
