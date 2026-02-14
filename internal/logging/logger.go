// Package logging provides structured logging for ralph.
// It supports debug, info, error levels with file rotation and cleanup.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Level represents log severity levels.
type Level int

const (
	// LevelDebug is for detailed debugging information.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// toSlogLevel converts our Level to slog.Level.
func (l Level) toSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Config configures the logger.
type Config struct {
	// Level is the minimum log level to output.
	Level Level
	// LogDir is the directory to write log files (e.g., ".ralph/logs").
	LogDir string
	// MaxLogFiles is the maximum number of log files to keep.
	MaxLogFiles int
	// MaxLogAge is the maximum age of log files before cleanup.
	MaxLogAge time.Duration
	// Console enables logging to stdout/stderr in addition to file.
	Console bool
	// JSONFormat uses JSON output format for structured logs.
	JSONFormat bool
}

// DefaultConfig returns default logging configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:       LevelInfo,
		LogDir:      ".ralph/logs",
		MaxLogFiles: 10,
		MaxLogAge:   7 * 24 * time.Hour, // 7 days
		Console:     false,
		JSONFormat:  false,
	}
}

// Logger is a structured logger for ralph.
type Logger struct {
	slog    *slog.Logger
	config  *Config
	logFile *os.File
	logPath string
	mu      sync.Mutex
}

// New creates a new logger with the given configuration.
// It creates a log file in the configured log directory.
func New(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logger := &Logger{
		config: config,
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	logPath := filepath.Join(config.LogDir, fmt.Sprintf("ralph_%s.log", time.Now().Format("20060102_150405")))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger.logFile = logFile
	logger.logPath = logPath

	// Set up writers
	var writers []io.Writer
	writers = append(writers, logFile)
	if config.Console {
		writers = append(writers, os.Stderr)
	}
	multiWriter := io.MultiWriter(writers...)

	// Create slog handler
	opts := &slog.HandlerOptions{
		Level: config.Level.toSlogLevel(),
	}

	var handler slog.Handler
	if config.JSONFormat {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	logger.slog = slog.New(handler)

	// Run initial cleanup
	go logger.Cleanup()

	return logger, nil
}

// NewNoop creates a no-op logger that discards all output.
// Useful for testing or when logging is disabled.
func NewNoop() *Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	return &Logger{
		slog:   slog.New(handler),
		config: DefaultConfig(),
	}
}

// LogPath returns the path to the current log file.
func (l *Logger) LogPath() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logPath
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// With returns a new logger with the given attributes added.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog:    l.slog.With(args...),
		config:  l.config,
		logFile: l.logFile,
		logPath: l.logPath,
	}
}

// WithContext logs with context values.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values (session ID, task ID, etc.)
	newLogger := l.slog

	if sessionID, ok := ctx.Value(ContextKeySessionID).(string); ok && sessionID != "" {
		newLogger = newLogger.With("session_id", sessionID)
	}
	if taskID, ok := ctx.Value(ContextKeyTaskID).(string); ok && taskID != "" {
		newLogger = newLogger.With("task_id", taskID)
	}

	return &Logger{
		slog:    newLogger,
		config:  l.config,
		logFile: l.logFile,
		logPath: l.logPath,
	}
}

// Context keys for logging.
type contextKey string

const (
	// ContextKeySessionID is the context key for session ID.
	ContextKeySessionID contextKey = "session_id"
	// ContextKeyTaskID is the context key for task ID.
	ContextKeyTaskID contextKey = "task_id"
)

// WithSessionID adds session ID to the context.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ContextKeySessionID, sessionID)
}

// WithTaskID adds task ID to the context.
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, ContextKeyTaskID, taskID)
}

// Writer returns an io.Writer that logs each line at the given level.
// Useful for capturing output from external commands.
func (l *Logger) Writer(level Level) io.Writer {
	return &logWriter{
		logger: l,
		level:  level,
	}
}

// logWriter adapts the logger to io.Writer.
type logWriter struct {
	logger *Logger
	level  Level
	buf    []byte
}

// Write implements io.Writer, logging each complete line.
func (w *logWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := indexOf(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]

		switch w.level {
		case LevelDebug:
			w.logger.Debug(line)
		case LevelInfo:
			w.logger.Info(line)
		case LevelWarn:
			w.logger.Warn(line)
		case LevelError:
			w.logger.Error(line)
		}
	}
	return len(p), nil
}

// Flush writes any remaining buffered data.
func (w *logWriter) Flush() {
	if len(w.buf) > 0 {
		line := string(w.buf)
		w.buf = nil
		switch w.level {
		case LevelDebug:
			w.logger.Debug(line)
		case LevelInfo:
			w.logger.Info(line)
		case LevelWarn:
			w.logger.Warn(line)
		case LevelError:
			w.logger.Error(line)
		}
	}
}

func indexOf(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

// Cleanup removes old log files based on MaxLogFiles and MaxLogAge.
func (l *Logger) Cleanup() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config.LogDir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	// Collect log files with their info
	type logFileInfo struct {
		path    string
		modTime time.Time
	}
	var logFiles []logFileInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Match ralph_*.log pattern
		if len(name) > 6 && name[:6] == "ralph_" && name[len(name)-4:] == ".log" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			logFiles = append(logFiles, logFileInfo{
				path:    filepath.Join(l.config.LogDir, name),
				modTime: info.ModTime(),
			})
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.After(logFiles[j].modTime)
	})

	now := time.Now()
	var removed int

	for i, lf := range logFiles {
		// Skip the current log file
		if lf.path == l.logPath {
			continue
		}

		// Remove if exceeds max count or max age
		shouldRemove := false
		if l.config.MaxLogFiles > 0 && i >= l.config.MaxLogFiles {
			shouldRemove = true
		}
		if l.config.MaxLogAge > 0 && now.Sub(lf.modTime) > l.config.MaxLogAge {
			shouldRemove = true
		}

		if shouldRemove {
			if err := os.Remove(lf.path); err == nil {
				removed++
			}
		}
	}

	if removed > 0 {
		l.slog.Debug("cleaned up old log files", "count", removed)
	}

	return nil
}

// Rotate closes the current log file and creates a new one.
// Useful for long-running sessions that want to start a fresh log.
func (l *Logger) Rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing file
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	// Create new log file
	logPath := filepath.Join(l.config.LogDir, fmt.Sprintf("ralph_%s.log", time.Now().Format("20060102_150405")))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	l.logFile = logFile
	l.logPath = logPath

	// Update slog handler
	var writers []io.Writer
	writers = append(writers, logFile)
	if l.config.Console {
		writers = append(writers, os.Stderr)
	}
	multiWriter := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level: l.config.Level.toSlogLevel(),
	}

	var handler slog.Handler
	if l.config.JSONFormat {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	l.slog = slog.New(handler)

	return nil
}

