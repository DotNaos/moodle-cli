package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedPrefix = "v1"

type Box struct {
	aead cipher.AEAD
}

func NewBox(secret string) (Box, error) {
	key := normalizeKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return Box{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Box{}, err
	}
	return Box{aead: aead}, nil
}

func (b Box) EncryptString(value string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := b.aead.Seal(nil, nonce, []byte(value), nil)
	return encryptedPrefix + ":" + base64.RawURLEncoding.EncodeToString(nonce) + ":" + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (b Box) DecryptString(value string) (string, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[0] != encryptedPrefix {
		return "", fmt.Errorf("encrypted value has unsupported format")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	plaintext, err := b.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plaintext), nil
}

func normalizeKey(secret string) []byte {
	trimmed := strings.TrimSpace(secret)
	if data, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(data) == 32 {
		return data
	}
	if data, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(data) == 32 {
		return data
	}
	sum := sha256.Sum256([]byte(trimmed))
	return sum[:]
}
