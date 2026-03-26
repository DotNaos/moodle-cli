package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestChecksumForAsset(t *testing.T) {
	content := "abc123  moodle_linux_amd64.tar.gz\n"
	got, err := checksumForAsset(content, "moodle_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc123" {
		t.Fatalf("expected abc123, got %q", got)
	}
}

func TestVerifyChecksumFile(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "moodle_linux_amd64.tar.gz")
	checksumPath := filepath.Join(tempDir, "checksums.txt")

	content := []byte("archive-data")
	if err := os.WriteFile(archivePath, content, 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	sum := sha256.Sum256(content)
	checksumLine := hex.EncodeToString(sum[:]) + "  moodle_linux_amd64.tar.gz\n"
	if err := os.WriteFile(checksumPath, []byte(checksumLine), 0o600); err != nil {
		t.Fatalf("write checksum: %v", err)
	}

	if err := verifyChecksumFile(archivePath, checksumPath, "moodle_linux_amd64.tar.gz"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractTarGzBinary(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "moodle_linux_amd64.tar.gz")
	if err := writeTarGzArchive(archivePath, "moodle", []byte("binary")); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	binaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		t.Fatalf("extract binary: %v", err)
	}
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("expected extracted binary, got %q", string(data))
	}
}

func TestExtractZipBinary(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "moodle_windows_amd64.zip")
	if err := writeZipArchive(archivePath, "moodle.exe", []byte("binary")); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	binaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		t.Fatalf("extract binary: %v", err)
	}
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	if string(data) != "binary" {
		t.Fatalf("expected extracted binary, got %q", string(data))
	}
}

func writeTarGzArchive(path string, name string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	header := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(data)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = tarWriter.Write(data)
	return err
}

func writeZipArchive(path string, name string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	entry, err := zipWriter.Create(name)
	if err != nil {
		zipWriter.Close()
		return err
	}
	if _, err := entry.Write(data); err != nil {
		zipWriter.Close()
		return err
	}
	return zipWriter.Close()
}
