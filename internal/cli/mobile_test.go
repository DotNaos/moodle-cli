package cli

import "testing"

func TestInspectMobileQRLink(t *testing.T) {
	result, err := inspectMobileQRLink("moodlemobile://https://moodle.fhgr.ch?qrlogin=abcdef1234567890abcdef1234567890&userid=22388")
	if err != nil {
		t.Fatalf("expected inspect to succeed, got %v", err)
	}

	if result.Kind != "qr-auto-login" {
		t.Fatalf("unexpected kind %q", result.Kind)
	}
	if result.SiteURL != "https://moodle.fhgr.ch" {
		t.Fatalf("unexpected site URL %q", result.SiteURL)
	}
	if result.UserID != 22388 {
		t.Fatalf("unexpected user ID %d", result.UserID)
	}
	if result.TokenWSFunction != "tool_mobile_get_tokens_for_qr_login" {
		t.Fatalf("unexpected token function %q", result.TokenWSFunction)
	}
	if result.TokenEndpoint != "https://moodle.fhgr.ch/lib/ajax/service-nologin.php?info=tool_mobile_get_tokens_for_qr_login&lang=de_ch" {
		t.Fatalf("unexpected token endpoint %q", result.TokenEndpoint)
	}
	if result.PublicConfigEndpoint != "https://moodle.fhgr.ch/lib/ajax/service-nologin.php?info=tool_mobile_get_public_config&lang=en" {
		t.Fatalf("unexpected public config endpoint %q", result.PublicConfigEndpoint)
	}
}
