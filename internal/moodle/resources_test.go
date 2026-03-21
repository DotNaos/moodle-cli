package moodle

import "testing"

func TestParseResourcesIncludesUploadedAt(t *testing.T) {
	html := `
  <li class="activity resource modtype_resource">
    <div data-activityname="Data Augmentation">
      <a href="https://moodle.fhgr.ch/mod/resource/view.php?id=956991">
        <span class="instancename">Data Augmentation <span class="accesshide">Datei</span></span>
      </a>
      <img src="https://moodle.fhgr.ch/theme/image.php/boost_union/core/1773924003/f/pdf?filtericon=1">
      <span class="activitybadge badge rounded-pill badge-none">PDF</span>
      <span class="resourcelinkdetails">646.3 KB · Hochgeladen 20.03.2026 15:30</span>
    </div>
  </li>`

	resources := ParseResources(html, "22585", "https://moodle.fhgr.ch")
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].UploadedAt != "2026-03-20T15:30:00+01:00" {
		t.Fatalf("expected uploadedAt to be parsed, got %q", resources[0].UploadedAt)
	}
}

func TestParseUploadedAtReturnsEmptyWhenMissing(t *testing.T) {
	if got := parseUploadedAt("646.3 KB"); got != "" {
		t.Fatalf("expected empty timestamp, got %q", got)
	}
}

func TestParseUploadedAtSupportsEnglishLabel(t *testing.T) {
	got := parseUploadedAt("646.3 KB · Uploaded 20.03.2026 15:30")
	if got != "2026-03-20T15:30:00+01:00" {
		t.Fatalf("expected english timestamp, got %q", got)
	}
}
