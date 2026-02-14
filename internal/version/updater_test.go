package version

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetArchiveName(t *testing.T) {
	name := GetArchiveName("v1.0.0")

	// Should contain version without 'v' prefix
	if !containsString(name, "1.0.0") {
		t.Errorf("GetArchiveName() = %q, should contain version", name)
	}

	// Should have correct extension
	if runtime.GOOS == "windows" {
		if !containsString(name, ".zip") {
			t.Errorf("GetArchiveName() = %q, should have .zip extension on Windows", name)
		}
	} else {
		if !containsString(name, ".tar.gz") {
			t.Errorf("GetArchiveName() = %q, should have .tar.gz extension", name)
		}
	}
}

func TestNewUpdater(t *testing.T) {
	u := NewUpdater()
	if u == nil {
		t.Fatal("NewUpdater() should not return nil")
	}
	if u.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
	if u.Repo != GitHubRepo {
		t.Errorf("Repo = %q, want %q", u.Repo, GitHubRepo)
	}
}

func TestExtractTarGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	destDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(destDir, 0755)

	// Create a test tar.gz archive with a ralph binary
	if err := createTestTarGz(archivePath); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	// Extract
	binaryPath, err := extractTarGz(archivePath, destDir)
	if err != nil {
		t.Fatalf("extractTarGz() error = %v", err)
	}

	if filepath.Base(binaryPath) != "ralph" {
		t.Errorf("binaryPath = %q, want ralph binary", binaryPath)
	}

	// Verify file exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Extracted binary should exist")
	}
}

func TestExtractZip(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")
	destDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(destDir, 0755)

	// Create a test zip archive with a ralph binary
	if err := createTestZip(archivePath); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	// Extract
	binaryPath, err := extractZip(archivePath, destDir)
	if err != nil {
		t.Fatalf("extractZip() error = %v", err)
	}

	if filepath.Base(binaryPath) != "ralph" {
		t.Errorf("binaryPath = %q, want ralph binary", binaryPath)
	}

	// Verify file exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Error("Extracted binary should exist")
	}
}

func TestExtract(t *testing.T) {
	tmpDir := t.TempDir()
	destDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(destDir, 0755)

	t.Run("tar.gz", func(t *testing.T) {
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		createTestTarGz(archivePath)

		_, err := Extract(archivePath, destDir)
		if err != nil {
			t.Errorf("Extract() error = %v", err)
		}
	})

	t.Run("zip", func(t *testing.T) {
		archivePath := filepath.Join(tmpDir, "test.zip")
		createTestZip(archivePath)

		extractDir := filepath.Join(tmpDir, "extracted_zip")
		os.MkdirAll(extractDir, 0755)

		_, err := Extract(archivePath, extractDir)
		if err != nil {
			t.Errorf("Extract() error = %v", err)
		}
	})
}

func TestInstallBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source binary
	srcPath := filepath.Join(tmpDir, "ralph_src")
	if err := os.WriteFile(srcPath, []byte("binary content"), 0755); err != nil {
		t.Fatal(err)
	}

	// Install to dest
	destPath := filepath.Join(tmpDir, "ralph_dest")
	if err := InstallBinary(srcPath, destPath); err != nil {
		t.Fatalf("InstallBinary() error = %v", err)
	}

	// Verify dest exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("Installed binary should exist")
	}

	// Verify content
	content, _ := os.ReadFile(destPath)
	if string(content) != "binary content" {
		t.Error("Binary content should match")
	}
}

func TestGetCurrentExecutable(t *testing.T) {
	exe, err := GetCurrentExecutable()
	if err != nil {
		t.Fatalf("GetCurrentExecutable() error = %v", err)
	}
	if exe == "" {
		t.Error("GetCurrentExecutable() should return non-empty path")
	}
}

func TestInstallBinary_MissingDir(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "ralph_src")
	os.WriteFile(srcPath, []byte("binary"), 0755)

	destPath := filepath.Join(tmpDir, "nonexistent", "ralph")
	err := InstallBinary(srcPath, destPath)
	if err == nil {
		t.Error("InstallBinary() should error when directory doesn't exist")
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func createTestTarGz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add ralph binary
	content := []byte("#!/bin/bash\necho 'ralph binary'")
	header := &tar.Header{
		Name: "ralph",
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	return nil
}

func createTestZip(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	// Add ralph binary
	w, err := zw.Create("ralph")
	if err != nil {
		return err
	}

	content := []byte("#!/bin/bash\necho 'ralph binary'")
	if _, err := w.Write(content); err != nil {
		return err
	}

	return nil
}

