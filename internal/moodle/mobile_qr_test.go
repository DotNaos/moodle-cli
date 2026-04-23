package moodle

import "testing"

func TestParseMobileQRLinkAutoLogin(t *testing.T) {
	link, err := ParseMobileQRLink("moodlemobile://https://moodle.fhgr.ch?qrlogin=abcdef1234567890abcdef1234567890&userid=22388")
	if err != nil {
		t.Fatalf("expected link to parse, got %v", err)
	}

	if link.SiteURL != "https://moodle.fhgr.ch" {
		t.Fatalf("unexpected site URL %q", link.SiteURL)
	}
	if link.QRLoginKey != "abcdef1234567890abcdef1234567890" {
		t.Fatalf("unexpected QR key %q", link.QRLoginKey)
	}
	if link.UserID != 22388 {
		t.Fatalf("unexpected user ID %d", link.UserID)
	}
	if !link.IsAutoLogin {
		t.Fatal("expected auto-login QR link")
	}
}

func TestParseMobileQRLinkRejectsWrongScheme(t *testing.T) {
	_, err := ParseMobileQRLink("https://moodle.fhgr.ch?qrlogin=abc&userid=1")
	if err == nil {
		t.Fatal("expected wrong scheme to fail")
	}
}

func TestRedactSecret(t *testing.T) {
	got := RedactSecret("abcdef1234567890abcdef1234567890")
	want := "abcdef**********************7890"
	if got != want {
		t.Fatalf("unexpected redaction: got %q want %q", got, want)
	}
}
