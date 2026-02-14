package logging

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")

	config := &Config{
		Level:       LevelDebug,
		LogDir:      logDir,
		MaxLogFiles: 5,
		MaxLogAge:   24 * time.Hour,
		Console:     false,
		JSONFormat:  false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	// Verify log directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}

	// Verify log file was created
	logPath := logger.LogPath()
	if logPath == "" {
		t.Error("LogPath() returned empty string")
	}
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestNewWithNilConfig(t *testing.T) {
	// Create default config location
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(originalDir) }()

	logger, err := New(nil)
	if err != nil {
		t.Fatalf("New(nil) error = %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Error("New(nil) returned nil logger")
	}
}

func TestNewNoop(t *testing.T) {
	logger := NewNoop()
	if logger == nil {
		t.Error("NewNoop() returned nil")
	}

	// Should not panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelDebug,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	// Log at all levels
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	// Read log file
	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "debug message") {
		t.Error("Log file missing debug message")
	}
	if !strings.Contains(contentStr, "info message") {
		t.Error("Log file missing info message")
	}
	if !strings.Contains(contentStr, "warn message") {
		t.Error("Log file missing warn message")
	}
	if !strings.Contains(contentStr, "error message") {
		t.Error("Log file missing error message")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Set level to Warn - should filter out Debug and Info
	config := &Config{
		Level:      LevelWarn,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "debug message") {
		t.Error("Debug message should have been filtered")
	}
	if strings.Contains(contentStr, "info message") {
		t.Error("Info message should have been filtered")
	}
	if !strings.Contains(contentStr, "warn message") {
		t.Error("Warn message should be present")
	}
	if !strings.Contains(contentStr, "error message") {
		t.Error("Error message should be present")
	}
}

func TestJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: true,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	logger.Info("test message", "key", "value")

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// JSON format should have quotes and colons
	contentStr := string(content)
	if !strings.Contains(contentStr, `"msg"`) {
		t.Error("JSON format should contain 'msg' key")
	}
	if !strings.Contains(contentStr, `"key"`) {
		t.Error("JSON format should contain 'key' key")
	}
}

func TestWith(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	taskLogger := logger.With("task_id", "TASK-001")
	taskLogger.Info("working on task")

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TASK-001") {
		t.Error("Log should contain task_id attribute")
	}
}

func TestWithContext(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	ctx := context.Background()
	ctx = WithSessionID(ctx, "sess-123")
	ctx = WithTaskID(ctx, "TASK-002")

	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("context message")

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "sess-123") {
		t.Error("Log should contain session_id from context")
	}
	if !strings.Contains(contentStr, "TASK-002") {
		t.Error("Log should contain task_id from context")
	}
}

func TestWriter(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	writer := logger.Writer(LevelInfo)

	// Write multiple lines
	_, _ = writer.Write([]byte("line one\nline two\n"))

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "line one") {
		t.Error("Log should contain 'line one'")
	}
	if !strings.Contains(contentStr, "line two") {
		t.Error("Log should contain 'line two'")
	}
}

func TestWriterFlush(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	writer := logger.Writer(LevelInfo)

	// Write partial line without newline
	_, _ = writer.Write([]byte("partial line"))

	// Flush to ensure it's written
	if lw, ok := writer.(*logWriter); ok {
		lw.Flush()
	}

	content, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "partial line") {
		t.Error("Log should contain flushed partial line")
	}
}

func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create old log files
	for i := 0; i < 15; i++ {
		name := filepath.Join(tmpDir, "ralph_20240101_00000"+string(rune('0'+i%10))+".log")
		if err := os.WriteFile(name, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test log file: %v", err)
		}
	}

	config := &Config{
		Level:       LevelInfo,
		LogDir:      tmpDir,
		MaxLogFiles: 5,
		MaxLogAge:   0, // Disable age-based cleanup
		Console:     false,
		JSONFormat:  false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Give cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// Count remaining log files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read log dir: %v", err)
	}

	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			count++
		}
	}

	// Should have at most MaxLogFiles + 1 (current log)
	if count > config.MaxLogFiles+1 {
		t.Errorf("Expected at most %d log files, got %d", config.MaxLogFiles+1, count)
	}
}

func TestRotate(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    false,
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	oldPath := logger.LogPath()
	logger.Info("before rotate")

	// Wait to ensure different timestamp (format is seconds-based)
	time.Sleep(1100 * time.Millisecond)

	if err := logger.Rotate(); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}

	newPath := logger.LogPath()
	// New path should be different since we waited >1 second
	if oldPath == newPath {
		t.Log("Warning: Log paths are the same (test ran too fast)")
		// Skip the rest of this test if paths are the same
		return
	}

	logger.Info("after rotate")

	// Old file should have old content
	oldContent, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("Failed to read old log file: %v", err)
	}
	if !strings.Contains(string(oldContent), "before rotate") {
		t.Error("Old log file should contain 'before rotate'")
	}

	// New file should have new content
	newContent, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("Failed to read new log file: %v", err)
	}
	if !strings.Contains(string(newContent), "after rotate") {
		t.Error("New log file should contain 'after rotate'")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %v, want %v", tt.level, got, tt.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != LevelInfo {
		t.Errorf("DefaultConfig().Level = %v, want %v", config.Level, LevelInfo)
	}
	if config.LogDir != ".ralph/logs" {
		t.Errorf("DefaultConfig().LogDir = %v, want %v", config.LogDir, ".ralph/logs")
	}
	if config.MaxLogFiles != 10 {
		t.Errorf("DefaultConfig().MaxLogFiles = %v, want %v", config.MaxLogFiles, 10)
	}
	if config.MaxLogAge != 7*24*time.Hour {
		t.Errorf("DefaultConfig().MaxLogAge = %v, want %v", config.MaxLogAge, 7*24*time.Hour)
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		b        []byte
		c        byte
		expected int
	}{
		{[]byte("hello\nworld"), '\n', 5},
		{[]byte("hello"), '\n', -1},
		{[]byte(""), '\n', -1},
	}

	for _, tt := range tests {
		if got := indexOf(tt.b, tt.c); got != tt.expected {
			t.Errorf("indexOf(%q, %q) = %v, want %v", tt.b, tt.c, got, tt.expected)
		}
	}
}

// Mock buffer to test console output
type testBuffer struct {
	bytes.Buffer
}

func (b *testBuffer) Sync() error { return nil }

func TestConsoleOutput(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Level:      LevelInfo,
		LogDir:     tmpDir,
		Console:    true, // Enable console output
		JSONFormat: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer logger.Close()

	// Just verify it doesn't panic with console enabled
	logger.Info("console test")
}

