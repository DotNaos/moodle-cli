package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

const APIKeyPrefix = "moodle_"

var ErrUnauthorized = errors.New("authentication required")

func APIKeyFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if key := strings.TrimSpace(r.URL.Query().Get("key")); key != "" {
		return key
	}
	if key := strings.TrimSpace(r.Header.Get("X-Moodle-App-Key")); key != "" {
		return key
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[len("bearer "):])
	}
	return ""
}

func GenerateAPIKey() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return APIKeyPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}

func KeyPrefix(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 14 {
		return key
	}
	return key[:14]
}

func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func HMACAPIKey(key string, secret []byte) string {
	if len(secret) == 0 {
		return HashAPIKey(key)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}

func ConstantTimeEqual(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
