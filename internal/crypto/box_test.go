package crypto

import "testing"

func TestBoxEncryptDecryptString(t *testing.T) {
	box, err := NewBox("test-secret")
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	encrypted, err := box.EncryptString("sensitive moodle session")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == "sensitive moodle session" {
		t.Fatalf("encrypted value should not equal plaintext")
	}
	decrypted, err := box.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != "sensitive moodle session" {
		t.Fatalf("expected decrypted plaintext, got %q", decrypted)
	}
}
