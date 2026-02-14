package version

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BinaryDownloadURL is the URL pattern for downloading release binaries.
const BinaryDownloadURL = "https://github.com/%s/releases/download/%s/%s"

// Updater handles downloading and installing updates.
type Updater struct {
	HTTPClient *http.Client
	Repo       string
}

// NewUpdater creates a new updater.
func NewUpdater() *Updater {
	return &Updater{
		HTTPClient: http.DefaultClient,
		Repo:       GitHubRepo,
	}
}

// GetArchiveName returns the archive name for the current platform.
func GetArchiveName(version string) string {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Map OS names to match goreleaser output
	switch osName {
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	case "windows":
		osName = "Windows"
	}

	// Map arch names to match goreleaser output
	switch archName {
	case "amd64":
		archName = "x86_64"
	case "386":
		archName = "i386"
	}

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	return fmt.Sprintf("ralph_%s_%s_%s.%s", version, osName, archName, ext)
}

// Download downloads the release archive for the given version.
func (u *Updater) Download(ctx context.Context, version, destDir string) (string, error) {
	archiveName := GetArchiveName(version)
	url := fmt.Sprintf(BinaryDownloadURL, u.Repo, version, archiveName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	destPath := filepath.Join(destDir, archiveName)
	file, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return destPath, nil
}

// Extract extracts the binary from the archive.
func Extract(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

// extractTarGz extracts a .tar.gz archive.
func extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var binaryPath string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Look for the ralph binary
		name := filepath.Base(header.Name)
		if name == "ralph" || name == "ralph.exe" {
			binaryPath = filepath.Join(destDir, name)
			outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return "", err
			}
			outFile.Close()
			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("ralph binary not found in archive")
	}
	return binaryPath, nil
}

// extractZip extracts a .zip archive.
func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var binaryPath string
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "ralph" || name == "ralph.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}

			binaryPath = filepath.Join(destDir, name)
			outFile, err := os.OpenFile(binaryPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				rc.Close()
				return "", err
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()
			if err != nil {
				return "", err
			}
			break
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("ralph binary not found in archive")
	}
	return binaryPath, nil
}

// InstallBinary installs the binary to the specified path.
func InstallBinary(binaryPath, installPath string) error {
	// Check if we need sudo (we don't have write permission)
	dir := filepath.Dir(installPath)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("install directory does not exist: %s", dir)
	}

	// Read the binary
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}

	// Write to install path
	if err := os.WriteFile(installPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary (may need sudo): %w", err)
	}

	return nil
}

// GetCurrentExecutable returns the path to the current ralph executable.
func GetCurrentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
