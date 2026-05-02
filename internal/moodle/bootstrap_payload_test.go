package moodle

import (
	"strings"
	"testing"
	"time"
)

func TestBootstrapPayloadRoundTrip(t *testing.T) {
	session := MobileSession{
		SchoolID:     "fhgr",
		SiteURL:      "https://moodle.example.test",
		UserID:       42,
		Token:        "test-token",
		PrivateToken: "test-private-token",
		CreatedAt:    time.Unix(100, 0).UTC(),
	}
	payload := NewBootstrapPayload(session, "ghcr.io/example/moodle-services:test")

	encoded, err := EncodeBootstrapPayload(payload)
	if err != nil {
		t.Fatalf("EncodeBootstrapPayload: %v", err)
	}
	decoded, err := DecodeBootstrapPayload(encoded)
	if err != nil {
		t.Fatalf("DecodeBootstrapPayload: %v", err)
	}

	if decoded.MobileSession.Token != session.Token {
		t.Fatalf("unexpected token %q", decoded.MobileSession.Token)
	}
	if decoded.MobileSession.PrivateToken != session.PrivateToken {
		t.Fatalf("unexpected private token %q", decoded.MobileSession.PrivateToken)
	}
	if decoded.Image != payload.Image {
		t.Fatalf("unexpected image %q", decoded.Image)
	}
}

func TestDecodeBootstrapPayloadRejectsTampering(t *testing.T) {
	payload := NewBootstrapPayload(MobileSession{
		SiteURL:   "https://moodle.example.test",
		UserID:    42,
		Token:     "test-token",
		CreatedAt: time.Unix(100, 0).UTC(),
	}, "")

	encoded, err := EncodeBootstrapPayload(payload)
	if err != nil {
		t.Fatalf("EncodeBootstrapPayload: %v", err)
	}
	parts := strings.Split(encoded, ".")
	replacement := "A"
	if parts[1][0] == 'A' {
		replacement = "B"
	}
	parts[1] = replacement + parts[1][1:]
	tampered := strings.Join(parts, ".")
	if _, err := DecodeBootstrapPayload(tampered); err == nil {
		t.Fatalf("expected tampered payload to fail")
	}
}

func TestEncodeBootstrapPayloadRequiresToken(t *testing.T) {
	_, err := EncodeBootstrapPayload(NewBootstrapPayload(MobileSession{
		SiteURL: "https://moodle.example.test",
		UserID:  42,
	}, ""))
	if err == nil {
		t.Fatalf("expected missing token to fail")
	}
}
