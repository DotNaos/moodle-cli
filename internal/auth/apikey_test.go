package auth

import "testing"

func TestGenerateAPIKeyAndHash(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if got := KeyPrefix(key); len(got) == 0 || got == key {
		t.Fatalf("expected shortened prefix for generated key, got %q", got)
	}
	hash := HMACAPIKey(key, []byte("secret"))
	if hash == "" || hash == HMACAPIKey(key, []byte("other")) {
		t.Fatalf("expected keyed hash to depend on secret")
	}
	if !ConstantTimeEqual(HashAPIKey(key), HashAPIKey(key)) {
		t.Fatalf("expected same hashes to compare equal")
	}
}
