package cli

import "testing"

func TestNormalizeBrowserURLRemovesResourceRedirectFlag(t *testing.T) {
	input := "https://moodle.fhgr.ch/mod/resource/view.php?id=956877&redirect=1"
	got := normalizeBrowserURL(input)
	want := "https://moodle.fhgr.ch/mod/resource/view.php?id=956877"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeBrowserURLLeavesOtherURLsUntouched(t *testing.T) {
	input := "https://moodle.fhgr.ch/course/view.php?id=22583"
	got := normalizeBrowserURL(input)
	if got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}
