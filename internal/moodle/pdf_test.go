package moodle

import (
	"errors"
	"testing"
)

func TestShouldAttemptOCR(t *testing.T) {
	if !shouldAttemptOCR("", errors.New("native extraction failed")) {
		t.Fatalf("expected OCR attempt when native extraction failed")
	}
	if !shouldAttemptOCR("", nil) {
		t.Fatalf("expected OCR attempt for empty native text")
	}
	if !shouldAttemptOCR("short text only", nil) {
		t.Fatalf("expected OCR attempt for very short text")
	}

	goodText := "This is a readable sentence. " +
		"This is another readable sentence with enough words to exceed the threshold significantly. " +
		"The extractor should keep this without OCR because the content quality is high and consistent. " +
		"Adding more words here keeps the text long enough to avoid triggering OCR heuristics."
	if shouldAttemptOCR(goodText, nil) {
		t.Fatalf("did not expect OCR attempt for readable text")
	}
}

func TestShouldPreferOCR(t *testing.T) {
	if !shouldPreferOCR("", errors.New("native failed"), "valid ocr output", nil) {
		t.Fatalf("expected OCR preference when native extraction failed")
	}
	if shouldPreferOCR("native text", nil, "", nil) {
		t.Fatalf("did not expect OCR preference for empty OCR output")
	}

	nativePoor := "l l l l l l l l"
	ocrGood := "This output contains clear readable words and punctuation."
	if !shouldPreferOCR(nativePoor, nil, ocrGood, nil) {
		t.Fatalf("expected OCR preference for clearly better OCR output")
	}

	nativeGood := "This sentence is clear and already readable without OCR."
	ocrWorse := "Th1s se nte nce i$ noisy"
	if shouldPreferOCR(nativeGood, nil, ocrWorse, nil) {
		t.Fatalf("did not expect OCR preference for worse OCR output")
	}
}

func TestSelectTesseractLanguage(t *testing.T) {
	out := "List of available languages in \"/opt/homebrew/share/tessdata\" (2):\ndeu\neng\n"
	if got := selectTesseractLanguage(out, "", nil); got != "deu+eng" {
		t.Fatalf("expected deu+eng, got %q", got)
	}

	out = "List of available languages\neng\n"
	if got := selectTesseractLanguage(out, "", nil); got != "eng" {
		t.Fatalf("expected eng, got %q", got)
	}

	if got := selectTesseractLanguage("", "", errors.New("failed")); got != "" {
		t.Fatalf("expected empty language on error, got %q", got)
	}
}

func TestPageIndexFromPath(t *testing.T) {
	if got := pageIndexFromPath("/tmp/page-12.png"); got != 12 {
		t.Fatalf("expected 12, got %d", got)
	}
	if got := pageIndexFromPath("/tmp/page.png"); got <= 1000 {
		t.Fatalf("expected large fallback index, got %d", got)
	}
}
