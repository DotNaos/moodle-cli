package moodle

import (
	"strings"
	"testing"
)

func TestDisplayCourseNameRemovesTrailingCodes(t *testing.T) {
	got := DisplayCourseName("Algorithmen des wissenschaftlichen Rechnens (cds-116) FS26", nil)
	if got != "Algorithmen des wissenschaftlichen Rechnens" {
		t.Fatalf("unexpected cleaned course name: %q", got)
	}
}

func TestRenderCoursePageReaderBuildsSectionReaderView(t *testing.T) {
	htmlContent := `
    <li id="section-1" data-id="1" data-sectionname="Thema 1: Sparse Grids">
      <div class="summarytext"><p>Einfuhrung in die Woche.</p></div>
      <li class="activity label modtype_label">
        <div class="contentwithoutlink"><p>Bitte zuerst das Aufgabenblatt lesen.</p></div>
      </li>
      <li class="activity resource modtype_resource">
        <div data-activityname="Folien Teil 1">
          <a href="https://example.com/mod/resource/view.php?id=100"></a>
          <span class="activitybadge">PDF</span>
          <span class="resourcelinkdetails">Hochgeladen 20.03.2026 15:30</span>
        </div>
      </li>
    </li>
    <li id="section-2" data-id="2" data-sectionname="Thema 2: Tensorfaktorisierung">
      <li class="activity resource modtype_resource">
        <div data-activityname="Aufgabenblatt 02">
          <a href="https://example.com/mod/resource/view.php?id=101"></a>
          <span class="activitybadge">PDF</span>
        </div>
      </li>
    </li>
  `

	got := RenderCoursePageReader(htmlContent, "42", "https://example.com")

	wantContains := []string{
		"Thema 1: Sparse Grids",
		"Einfuhrung in die Woche.",
		"Bitte zuerst das Aufgabenblatt lesen.",
		"- Folien Teil 1 · PDF · 2026-03-20T15:30:00+01:00",
		"Thema 2: Tensorfaktorisierung",
		"- Aufgabenblatt 02 · PDF",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in preview, got %q", want, got)
		}
	}
}
