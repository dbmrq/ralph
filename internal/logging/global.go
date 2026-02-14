// Package logging provides structured logging for ralph.
// This file provides a global logger instance for convenience.
package logging

import (
	"sync"
)

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
	globalOnce   sync.Once
)

// Global returns the global logger instance.
// If not initialized, returns a no-op logger.
func Global() *Logger {
	globalMu.RLock()
	l := globalLogger
	globalMu.RUnlock()

	if l != nil {
		return l
	}

	// Return no-op logger if not initialized
	globalOnce.Do(func() {
		globalMu.Lock()
		if globalLogger == nil {
			globalLogger = NewNoop()
		}
		globalMu.Unlock()
	})

	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// SetGlobal sets the global logger instance.
func SetGlobal(l *Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = l
}

// Debug logs a debug message using the global logger.
func Debug(msg string, args ...any) {
	Global().Debug(msg, args...)
}

// Info logs an info message using the global logger.
func Info(msg string, args ...any) {
	Global().Info(msg, args...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, args ...any) {
	Global().Warn(msg, args...)
}

// Error(msg string, args ...any) logs an error message using the global logger.
func Error(msg string, args ...any) {
	Global().Error(msg, args...)
}

// With returns a new logger with the given attributes added.
func With(args ...any) *Logger {
	return Global().With(args...)
}

// InitGlobal initializes the global logger with the given configuration.
// This should be called early in application startup.
// If config is nil, default configuration is used.
func InitGlobal(config *Config) error {
	l, err := New(config)
	if err != nil {
		return err
	}
	SetGlobal(l)
	return nil
}

// CloseGlobal closes the global logger.
// This should be called during application shutdown.
func CloseGlobal() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalLogger != nil {
		err := globalLogger.Close()
		globalLogger = nil
		return err
	}
	return nil
}
