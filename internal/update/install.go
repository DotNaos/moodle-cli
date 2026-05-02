package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type InstallResult struct {
	InstalledTag string
	TargetPath   string
	Updated      bool
}

func (c *Client) Update(ctx context.Context, targetPath string, currentVersion string) (InstallResult, error) {
	availability, release, err := c.Check(ctx, currentVersion)
	if err != nil {
		return InstallResult{}, err
	}
	if !availability.NeedsUpdate {
		return InstallResult{InstalledTag: availability.LatestTag, TargetPath: targetPath, Updated: false}, nil
	}

	archiveName, err := CurrentArchiveAssetName()
	if err != nil {
		return InstallResult{}, err
	}
	archiveAsset, err := FindAsset(release, archiveName)
	if err != nil {
		return InstallResult{}, err
	}
	checksumAsset, err := ChecksumAsset(release)
	if err != nil {
		return InstallResult{}, err
	}

	tempDir, err := os.MkdirTemp("", "moodle-update-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, archiveAsset.Name)
	checksumPath := filepath.Join(tempDir, checksumAsset.Name)

	if err := c.downloadToFile(ctx, archiveAsset.BrowserDownloadURL, archivePath); err != nil {
		return InstallResult{}, err
	}
	if err := c.downloadToFile(ctx, checksumAsset.BrowserDownloadURL, checksumPath); err != nil {
		return InstallResult{}, err
	}
	if err := verifyChecksumFile(archivePath, checksumPath, archiveAsset.Name); err != nil {
		return InstallResult{}, err
	}

	extractedBinaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		return InstallResult{}, err
	}

	if err := replaceExecutable(targetPath, extractedBinaryPath); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		InstalledTag: availability.LatestTag,
		TargetPath:   targetPath,
		Updated:      true,
	}, nil
}

func (c *Client) downloadToFile(ctx context.Context, url string, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "moodle-services updater")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func verifyChecksumFile(archivePath string, checksumPath string, archiveName string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}

	expected, err := checksumForAsset(string(data), archiveName)
	if err != nil {
		return err
	}

	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	return nil
}

func checksumForAsset(content string, assetName string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", assetName)
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractBinary(archivePath string, destDir string) (string, error) {
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		return extractTarGzBinary(archivePath, destDir)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZipBinary(archivePath, destDir)
	default:
		return "", fmt.Errorf("unsupported archive format: %s", archivePath)
	}
}

func extractTarGzBinary(archivePath string, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(header.Name)
		if base != "moodle" && base != "moodle.exe" {
			continue
		}

		destPath := filepath.Join(destDir, base)
		if err := writeReaderToFile(destPath, tarReader, 0o755); err != nil {
			return "", err
		}
		return destPath, nil
	}

	return "", fmt.Errorf("binary not found in archive")
}

func extractZipBinary(archivePath string, destDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, file := range reader.File {
		base := filepath.Base(file.Name)
		if base != "moodle.exe" && base != "moodle" {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return "", err
		}
		destPath := filepath.Join(destDir, base)
		writeErr := writeReaderToFile(destPath, handle, 0o755)
		closeErr := handle.Close()
		if writeErr != nil {
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return destPath, nil
	}

	return "", fmt.Errorf("binary not found in archive")
}

func writeReaderToFile(path string, reader io.Reader, mode os.FileMode) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func replaceExecutable(targetPath string, sourcePath string) error {
	if runtime.GOOS == "windows" {
		return replaceExecutableWindows(targetPath, sourcePath)
	}
	return replaceExecutableUnix(targetPath, sourcePath)
}

func replaceExecutableUnix(targetPath string, sourcePath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(targetPath), ".moodle-update-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, bytes.NewReader(data)); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0o755); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

func replaceExecutableWindows(targetPath string, sourcePath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	pendingPath := targetPath + ".new"
	if err := copyFile(sourcePath, pendingPath, 0o755); err != nil {
		return err
	}

	scriptPath := filepath.Join(filepath.Dir(targetPath), "moodle-update.cmd")
	script := fmt.Sprintf(`@echo off
ping 127.0.0.1 -n 2 > nul
move /Y "%s" "%s" > nul
del "%s"
`, pendingPath, targetPath, "%~f0")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return err
	}

	cmd := exec.Command("cmd", "/C", "start", "", "/b", scriptPath)
	return cmd.Start()
}

func copyFile(src string, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}
