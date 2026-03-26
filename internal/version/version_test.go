package version

import "testing"

func TestParseSemverKeepsPrerelease(t *testing.T) {
	parsed, err := ParseSemver("v1.2.3-rc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Prerelease != "rc1" {
		t.Fatalf("expected prerelease rc1, got %q", parsed.Prerelease)
	}
}

func TestCompareTreatsPrereleaseAsOlderThanStable(t *testing.T) {
	result, err := Compare("v1.2.3-rc1", "v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result >= 0 {
		t.Fatalf("expected prerelease to be older, got %d", result)
	}
}

func TestCompareStableBeatsPrerelease(t *testing.T) {
	result, err := Compare("v1.2.3", "v1.2.3-rc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result <= 0 {
		t.Fatalf("expected stable to be newer, got %d", result)
	}
}
