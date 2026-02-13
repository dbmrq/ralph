// Package config provides configuration and timeout monitoring for ralph.
package config

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// TimeoutError represents an error that occurred due to a timeout.
type TimeoutError struct {
	State    TimeoutState
	Elapsed  time.Duration
	Deadline time.Duration
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout: agent %s after %v (limit: %v)",
		e.State.String(), e.Elapsed.Round(time.Second), e.Deadline.Round(time.Second))
}

// IsStuck returns true if the timeout was due to stuck detection.
func (e *TimeoutError) IsStuck() bool {
	return e.State == TimeoutStateStuck
}

// TimeoutState represents the current activity state of the agent.
type TimeoutState int

const (
	// TimeoutStateActive indicates the agent is actively producing output.
	TimeoutStateActive TimeoutState = iota
	// TimeoutStateStuck indicates no output has been produced recently.
	TimeoutStateStuck
)

// String returns a string representation of the timeout state.
func (s TimeoutState) String() string {
	switch s {
	case TimeoutStateActive:
		return "active"
	case TimeoutStateStuck:
		return "stuck"
	default:
		return "unknown"
	}
}

// TimeoutMonitor monitors output activity to implement the smart timeout system.
// It tracks when output was last written and determines if the agent is
// actively working (applying active timeout) or stuck (applying stuck timeout).
type TimeoutMonitor struct {
	mu sync.RWMutex

	config TimeoutConfig

	// startTime is when monitoring began.
	startTime time.Time

	// lastOutputTime is when output was last written.
	lastOutputTime time.Time

	// totalBytesWritten tracks total output bytes.
	totalBytesWritten int64
}

// NewTimeoutMonitor creates a new timeout monitor with the given configuration.
func NewTimeoutMonitor(config TimeoutConfig) *TimeoutMonitor {
	now := time.Now()
	return &TimeoutMonitor{
		config:         config,
		startTime:      now,
		lastOutputTime: now,
	}
}

// RecordOutput records that output was written at the current time.
// This resets the stuck timeout countdown.
func (m *TimeoutMonitor) RecordOutput(bytesWritten int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastOutputTime = time.Now()
	m.totalBytesWritten += int64(bytesWritten)
}

// State returns the current timeout state (active or stuck).
func (m *TimeoutMonitor) State() TimeoutState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stateAt(time.Now())
}

// stateAt returns the state at a given time. Must hold at least read lock.
func (m *TimeoutMonitor) stateAt(now time.Time) TimeoutState {
	if now.Sub(m.lastOutputTime) >= m.config.Stuck {
		return TimeoutStateStuck
	}
	return TimeoutStateActive
}

// CurrentTimeout returns the applicable timeout based on current state.
// Returns active timeout if agent is producing output, stuck timeout otherwise.
func (m *TimeoutMonitor) CurrentTimeout() time.Duration {
	if m.State() == TimeoutStateStuck {
		return m.config.Stuck
	}
	return m.config.Active
}

// TimeSinceLastOutput returns the duration since last output was recorded.
func (m *TimeoutMonitor) TimeSinceLastOutput() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.lastOutputTime)
}

// TotalElapsed returns the total elapsed time since monitoring started.
func (m *TimeoutMonitor) TotalElapsed() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.startTime)
}

// TotalBytesWritten returns the total bytes written since monitoring started.
func (m *TimeoutMonitor) TotalBytesWritten() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalBytesWritten
}

// IsExpired checks if the current timeout has expired based on activity state.
// When active: checks if total elapsed exceeds active timeout.
// When stuck: checks if time since last output exceeds stuck timeout.
func (m *TimeoutMonitor) IsExpired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isExpiredAt(time.Now())
}

// isExpiredAt checks expiration at a specific time. Must hold at least read lock.
func (m *TimeoutMonitor) isExpiredAt(now time.Time) bool {
	// Check stuck timeout first - if no output for stuck duration, we're expired
	if now.Sub(m.lastOutputTime) >= m.config.Stuck {
		return true
	}
	// Check active timeout - if total elapsed exceeds active timeout
	if now.Sub(m.startTime) >= m.config.Active {
		return true
	}
	return false
}

// Error returns a TimeoutError if the monitor has expired, nil otherwise.
// This is useful for generating a descriptive error message when a timeout occurs.
func (m *TimeoutMonitor) Error() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	if !m.isExpiredAt(now) {
		return nil
	}

	// Determine which timeout triggered expiration
	stuckElapsed := now.Sub(m.lastOutputTime)
	activeElapsed := now.Sub(m.startTime)

	if stuckElapsed >= m.config.Stuck {
		return &TimeoutError{
			State:    TimeoutStateStuck,
			Elapsed:  stuckElapsed,
			Deadline: m.config.Stuck,
		}
	}

	return &TimeoutError{
		State:    TimeoutStateActive,
		Elapsed:  activeElapsed,
		Deadline: m.config.Active,
	}
}

// DeadlineTime returns the absolute time when the current timeout will expire.
// This accounts for the smart timeout system - returning whichever comes first:
// the stuck deadline (lastOutput + stuck) or the active deadline (start + active).
func (m *TimeoutMonitor) DeadlineTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stuckDeadline := m.lastOutputTime.Add(m.config.Stuck)
	activeDeadline := m.startTime.Add(m.config.Active)

	if stuckDeadline.Before(activeDeadline) {
		return stuckDeadline
	}
	return activeDeadline
}

// TimeRemaining returns the duration until the current timeout expires.
// Returns 0 if already expired.
func (m *TimeoutMonitor) TimeRemaining() time.Duration {
	deadline := m.DeadlineTime()
	remaining := time.Until(deadline)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset resets the monitor to its initial state (useful for retries).
func (m *TimeoutMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.startTime = now
	m.lastOutputTime = now
	m.totalBytesWritten = 0
}

// ContextWithDeadline returns a context that will be canceled when the timeout
// expires. The context is dynamically updated based on output activity.
// The returned cancel function should be called when done to free resources.
func (m *TimeoutMonitor) ContextWithDeadline(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if m.IsExpired() {
					cancel()
					return
				}
			}
		}
	}()

	return ctx, cancel
}

// MonitoredWriter wraps an io.Writer and automatically records output activity
// to the timeout monitor.
type MonitoredWriter struct {
	w       io.Writer
	monitor *TimeoutMonitor
}

// NewMonitoredWriter creates a writer that records output activity.
func NewMonitoredWriter(w io.Writer, monitor *TimeoutMonitor) *MonitoredWriter {
	return &MonitoredWriter{
		w:       w,
		monitor: monitor,
	}
}

// Write implements io.Writer, recording output activity before delegating.
func (mw *MonitoredWriter) Write(p []byte) (n int, err error) {
	n, err = mw.w.Write(p)
	if n > 0 {
		mw.monitor.RecordOutput(n)
	}
	return n, err
}

// Monitor returns the associated timeout monitor.
func (mw *MonitoredWriter) Monitor() *TimeoutMonitor {
	return mw.monitor
}

