package logging

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGlobal(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	// Should return no-op logger when not initialized
	logger := Global()
	if logger == nil {
		t.Fatal("Global() returned nil")
	}

	// Should not panic
	logger.Info("test message")
}

func TestSetGlobal(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	tmpDir := t.TempDir()

	config := &Config{
		Level:  LevelInfo,
		LogDir: tmpDir,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	SetGlobal(logger)

	got := Global()
	if got != logger {
		t.Error("Global() should return the logger set by SetGlobal()")
	}

	// Clean up
	SetGlobal(nil)
}

func TestInitGlobal(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	tmpDir := t.TempDir()

	config := &Config{
		Level:  LevelInfo,
		LogDir: tmpDir,
	}

	err := InitGlobal(config)
	if err != nil {
		t.Fatalf("InitGlobal() error = %v", err)
	}
	defer CloseGlobal()

	// Global should now be initialized
	logger := Global()
	if logger == nil {
		t.Fatal("Global() returned nil after InitGlobal()")
	}

	logger.Info("test message")

	// Verify log file was created
	entries, _ := os.ReadDir(tmpDir)
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Log file should have been created")
	}
}

func TestCloseGlobal(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	tmpDir := t.TempDir()

	config := &Config{
		Level:  LevelInfo,
		LogDir: tmpDir,
	}

	err := InitGlobal(config)
	if err != nil {
		t.Fatalf("InitGlobal() error = %v", err)
	}

	err = CloseGlobal()
	if err != nil {
		t.Fatalf("CloseGlobal() error = %v", err)
	}

	// After close, global should be nil (will return noop on next call)
	globalMu.RLock()
	isNil := globalLogger == nil
	globalMu.RUnlock()
	if !isNil {
		t.Error("globalLogger should be nil after CloseGlobal()")
	}
}

func TestCloseGlobalWhenNil(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	// Should not error when nil
	err := CloseGlobal()
	if err != nil {
		t.Errorf("CloseGlobal() with nil logger should not error: %v", err)
	}
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	tmpDir := t.TempDir()

	config := &Config{
		Level:  LevelDebug,
		LogDir: tmpDir,
	}

	_ = InitGlobal(config)
	defer CloseGlobal()

	// Use convenience functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Wait for writes
	time.Sleep(50 * time.Millisecond)

	// Find and read log file
	entries, _ := os.ReadDir(tmpDir)
	var logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = filepath.Join(tmpDir, e.Name())
			break
		}
	}

	if logPath == "" {
		t.Fatal("No log file found")
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	for _, msg := range []string{"debug message", "info message", "warn message", "error message"} {
		if !strings.Contains(contentStr, msg) {
			t.Errorf("Log should contain %q", msg)
		}
	}
}

func TestGlobalWith(t *testing.T) {
	// Reset global state
	globalMu.Lock()
	globalLogger = nil
	globalOnce = sync.Once{}
	globalMu.Unlock()

	tmpDir := t.TempDir()

	config := &Config{
		Level:  LevelInfo,
		LogDir: tmpDir,
	}

	_ = InitGlobal(config)
	defer CloseGlobal()

	logger := With("component", "test")
	logger.Info("with test")

	// Give time for write
	time.Sleep(50 * time.Millisecond)

	// Verify attribute is present
	entries, _ := os.ReadDir(tmpDir)
	var logPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			logPath = filepath.Join(tmpDir, e.Name())
			break
		}
	}

	content, _ := os.ReadFile(logPath)
	if !strings.Contains(string(content), "component") {
		t.Error("Log should contain 'component' attribute")
	}
}
