package moodle

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const BootstrapPayloadVersion = "v1"

type BootstrapPayload struct {
	Version       string        `json:"version"`
	CreatedAt     time.Time     `json:"createdAt"`
	MobileSession MobileSession `json:"mobileSession"`
	Image         string        `json:"image,omitempty"`
}

func NewBootstrapPayload(session MobileSession, image string) BootstrapPayload {
	return BootstrapPayload{
		Version:       BootstrapPayloadVersion,
		CreatedAt:     time.Now(),
		MobileSession: session,
		Image:         image,
	}
}

func EncodeBootstrapPayload(payload BootstrapPayload) (string, error) {
	if payload.Version == "" {
		payload.Version = BootstrapPayloadVersion
	}
	if payload.CreatedAt.IsZero() {
		payload.CreatedAt = time.Now()
	}
	if err := validateBootstrapPayload(payload); err != nil {
		return "", err
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return payload.Version + "." + base64.RawURLEncoding.EncodeToString(data) + "." + hex.EncodeToString(sum[:]), nil
}

func DecodeBootstrapPayload(encoded string) (BootstrapPayload, error) {
	parts := strings.Split(strings.TrimSpace(encoded), ".")
	if len(parts) != 3 {
		return BootstrapPayload{}, fmt.Errorf("bootstrap payload has invalid format")
	}
	if parts[0] != BootstrapPayloadVersion {
		return BootstrapPayload{}, fmt.Errorf("unsupported bootstrap payload version %q", parts[0])
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return BootstrapPayload{}, fmt.Errorf("decode bootstrap payload: %w", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != parts[2] {
		return BootstrapPayload{}, fmt.Errorf("bootstrap payload checksum mismatch")
	}

	var payload BootstrapPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return BootstrapPayload{}, fmt.Errorf("parse bootstrap payload: %w", err)
	}
	if err := validateBootstrapPayload(payload); err != nil {
		return BootstrapPayload{}, err
	}
	return payload, nil
}

func validateBootstrapPayload(payload BootstrapPayload) error {
	if payload.Version != BootstrapPayloadVersion {
		return fmt.Errorf("unsupported bootstrap payload version %q", payload.Version)
	}
	if payload.MobileSession.SiteURL == "" {
		return fmt.Errorf("bootstrap payload missing site URL")
	}
	if payload.MobileSession.UserID == 0 {
		return fmt.Errorf("bootstrap payload missing user id")
	}
	if payload.MobileSession.Token == "" {
		return fmt.Errorf("bootstrap payload missing mobile token")
	}
	return nil
}
