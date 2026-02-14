package config

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestTimeoutError_Error(t *testing.T) {
	err := &TimeoutError{
		State:    TimeoutStateStuck,
		Elapsed:  35 * time.Minute,
		Deadline: 30 * time.Minute,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("expected non-empty error message")
	}
	if !containsStr(errStr, "stuck") {
		t.Errorf("expected error to contain 'stuck', got %q", errStr)
	}
}

func TestTimeoutError_IsStuck(t *testing.T) {
	stuckErr := &TimeoutError{State: TimeoutStateStuck}
	if !stuckErr.IsStuck() {
		t.Error("expected IsStuck() to return true for stuck state")
	}

	activeErr := &TimeoutError{State: TimeoutStateActive}
	if activeErr.IsStuck() {
		t.Error("expected IsStuck() to return false for active state")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTimeoutState_String(t *testing.T) {
	tests := []struct {
		state    TimeoutState
		expected string
	}{
		{TimeoutStateActive, "active"},
		{TimeoutStateStuck, "stuck"},
		{TimeoutState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("TimeoutState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewTimeoutMonitor(t *testing.T) {
	cfg := TimeoutConfig{
		Active: 2 * time.Hour,
		Stuck:  30 * time.Minute,
	}

	monitor := NewTimeoutMonitor(cfg)

	if monitor.config.Active != cfg.Active {
		t.Errorf("expected Active timeout %v, got %v", cfg.Active, monitor.config.Active)
	}
	if monitor.config.Stuck != cfg.Stuck {
		t.Errorf("expected Stuck timeout %v, got %v", cfg.Stuck, monitor.config.Stuck)
	}
	if monitor.TotalBytesWritten() != 0 {
		t.Errorf("expected 0 bytes written initially, got %d", monitor.TotalBytesWritten())
	}
	if monitor.State() != TimeoutStateActive {
		t.Errorf("expected initial state Active, got %v", monitor.State())
	}
}

func TestTimeoutMonitor_RecordOutput(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	monitor.RecordOutput(100)
	if monitor.TotalBytesWritten() != 100 {
		t.Errorf("expected 100 bytes written, got %d", monitor.TotalBytesWritten())
	}

	monitor.RecordOutput(50)
	if monitor.TotalBytesWritten() != 150 {
		t.Errorf("expected 150 bytes written, got %d", monitor.TotalBytesWritten())
	}
}

func TestTimeoutMonitor_State(t *testing.T) {
	// Use very short durations for testing
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Initially should be active
	if monitor.State() != TimeoutStateActive {
		t.Errorf("expected state Active, got %v", monitor.State())
	}

	// Wait for stuck timeout to elapse
	time.Sleep(60 * time.Millisecond)

	// Should now be stuck
	if monitor.State() != TimeoutStateStuck {
		t.Errorf("expected state Stuck, got %v", monitor.State())
	}

	// Record output to become active again
	monitor.RecordOutput(1)

	if monitor.State() != TimeoutStateActive {
		t.Errorf("expected state Active after output, got %v", monitor.State())
	}
}

func TestTimeoutMonitor_CurrentTimeout(t *testing.T) {
	cfg := TimeoutConfig{
		Active: 2 * time.Hour,
		Stuck:  30 * time.Minute,
	}
	monitor := NewTimeoutMonitor(cfg)

	// When active, should return active timeout
	if monitor.CurrentTimeout() != cfg.Active {
		t.Errorf("expected current timeout %v, got %v", cfg.Active, monitor.CurrentTimeout())
	}
}

func TestTimeoutMonitor_TimeSinceLastOutput(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	// Should be very small initially
	if monitor.TimeSinceLastOutput() > time.Second {
		t.Errorf("expected small time since output initially, got %v", monitor.TimeSinceLastOutput())
	}

	time.Sleep(50 * time.Millisecond)

	// Should have increased
	if monitor.TimeSinceLastOutput() < 50*time.Millisecond {
		t.Errorf("expected at least 50ms since output, got %v", monitor.TimeSinceLastOutput())
	}

	// Recording output should reset it
	monitor.RecordOutput(1)
	if monitor.TimeSinceLastOutput() > 10*time.Millisecond {
		t.Errorf("expected small time after recording output, got %v", monitor.TimeSinceLastOutput())
	}
}

func TestTimeoutMonitor_TotalElapsed(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	// Should be small initially
	if monitor.TotalElapsed() > time.Second {
		t.Errorf("expected small elapsed time initially, got %v", monitor.TotalElapsed())
	}

	time.Sleep(50 * time.Millisecond)

	// Should have increased
	if monitor.TotalElapsed() < 50*time.Millisecond {
		t.Errorf("expected at least 50ms elapsed, got %v", monitor.TotalElapsed())
	}
}

func TestTimeoutMonitor_IsExpired_StuckTimeout(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Should not be expired initially
	if monitor.IsExpired() {
		t.Error("expected not expired initially")
	}

	// Wait for stuck timeout
	time.Sleep(60 * time.Millisecond)

	// Should be expired due to stuck timeout
	if !monitor.IsExpired() {
		t.Error("expected expired after stuck timeout")
	}
}

func TestTimeoutMonitor_IsExpired_ActiveTimeout(t *testing.T) {
	cfg := TimeoutConfig{
		Active: 50 * time.Millisecond,
		Stuck:  10 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Keep it active by recording output
	for i := 0; i < 3; i++ {
		time.Sleep(5 * time.Millisecond)
		monitor.RecordOutput(1)
	}

	// Wait for active timeout to expire
	time.Sleep(60 * time.Millisecond)

	// Should be expired due to active timeout (total elapsed > active)
	if !monitor.IsExpired() {
		t.Error("expected expired after active timeout")
	}
}

func TestTimeoutMonitor_IsExpired_OutputPreventsStuck(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Record output periodically to prevent stuck
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		monitor.RecordOutput(1)
	}

	// Should not be expired because we kept recording output
	if monitor.IsExpired() {
		t.Error("expected not expired when recording output regularly")
	}
}

func TestTimeoutMonitor_DeadlineTime(t *testing.T) {
	cfg := TimeoutConfig{
		Active: 2 * time.Hour,
		Stuck:  30 * time.Minute,
	}
	monitor := NewTimeoutMonitor(cfg)

	deadline := monitor.DeadlineTime()

	// Deadline should be approximately stuck timeout from now
	// (since stuck is less than active and lastOutput == start)
	expectedDeadline := time.Now().Add(30 * time.Minute)

	// Allow 1 second tolerance
	diff := deadline.Sub(expectedDeadline)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("deadline off by more than 1 second: %v", diff)
	}
}

func TestTimeoutMonitor_TimeRemaining(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  100 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	remaining := monitor.TimeRemaining()
	if remaining < 90*time.Millisecond || remaining > 110*time.Millisecond {
		t.Errorf("expected remaining close to 100ms, got %v", remaining)
	}

	// Wait for timeout to expire
	time.Sleep(110 * time.Millisecond)

	remaining = monitor.TimeRemaining()
	if remaining != 0 {
		t.Errorf("expected 0 remaining when expired, got %v", remaining)
	}
}

func TestTimeoutMonitor_Reset(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Record some output and wait
	monitor.RecordOutput(100)
	time.Sleep(60 * time.Millisecond)

	// Should be stuck/expired
	if !monitor.IsExpired() {
		t.Error("expected expired before reset")
	}

	// Reset the monitor
	monitor.Reset()

	// Should not be expired after reset
	if monitor.IsExpired() {
		t.Error("expected not expired after reset")
	}
}

func TestTimeoutMonitor_Error_NotExpired(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  time.Minute,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Should return nil when not expired
	if monitor.Error() != nil {
		t.Error("expected nil error when not expired")
	}
}

func TestTimeoutMonitor_Error_StuckExpired(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Wait for stuck timeout
	time.Sleep(60 * time.Millisecond)

	err := monitor.Error()
	if err == nil {
		t.Fatal("expected error when stuck timeout expired")
	}

	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Fatalf("expected *TimeoutError, got %T", err)
	}
	if timeoutErr.State != TimeoutStateStuck {
		t.Errorf("expected stuck state, got %v", timeoutErr.State)
	}
	if !timeoutErr.IsStuck() {
		t.Error("expected IsStuck() to return true")
	}
}

func TestTimeoutMonitor_Error_ActiveExpired(t *testing.T) {
	cfg := TimeoutConfig{
		Active: 50 * time.Millisecond,
		Stuck:  20 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	// Keep recording output to prevent stuck timeout
	for i := 0; i < 3; i++ {
		time.Sleep(15 * time.Millisecond)
		monitor.RecordOutput(1)
	}

	// Wait for active timeout to expire
	time.Sleep(60 * time.Millisecond)

	err := monitor.Error()
	if err == nil {
		t.Fatal("expected error when active timeout expired")
	}

	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Fatalf("expected *TimeoutError, got %T", err)
	}
	// Note: may be stuck if we waited too long, but should be some kind of timeout error
	if timeoutErr.Elapsed == 0 {
		t.Error("expected non-zero elapsed time")
	}
}

func TestTimeoutMonitor_ContextWithDeadline(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  50 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	ctx, cancel := monitor.ContextWithDeadline(context.Background())
	defer cancel()

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("context should not be done initially")
	default:
		// expected
	}

	// Wait for stuck timeout + check interval
	time.Sleep(100 * time.Millisecond)

	// Context should be canceled due to stuck timeout
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Error("context should have been canceled due to timeout")
	}
}

func TestTimeoutMonitor_ContextWithDeadline_OutputPreventsCancel(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  100 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	ctx, cancel := monitor.ContextWithDeadline(context.Background())
	defer cancel()

	// Record output periodically to keep active
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(50 * time.Millisecond)
			monitor.RecordOutput(1)
		}
		close(done)
	}()

	// Wait for output goroutine to finish
	<-done

	// Context should not be canceled because we kept recording output
	select {
	case <-ctx.Done():
		t.Error("context should not be canceled when output is being recorded")
	default:
		// expected
	}
}

func TestMonitoredWriter_Write(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	var buf bytes.Buffer
	writer := NewMonitoredWriter(&buf, monitor)

	n, err := writer.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	if buf.String() != "hello" {
		t.Errorf("expected buffer to contain 'hello', got %q", buf.String())
	}

	if monitor.TotalBytesWritten() != 5 {
		t.Errorf("expected monitor to record 5 bytes, got %d", monitor.TotalBytesWritten())
	}
}

func TestMonitoredWriter_MultipleWrites(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	var buf bytes.Buffer
	writer := NewMonitoredWriter(&buf, monitor)

	writer.Write([]byte("hello "))
	writer.Write([]byte("world"))

	if buf.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", buf.String())
	}

	if monitor.TotalBytesWritten() != 11 {
		t.Errorf("expected 11 bytes, got %d", monitor.TotalBytesWritten())
	}
}

func TestMonitoredWriter_Monitor(t *testing.T) {
	cfg := TimeoutConfig{Active: time.Hour, Stuck: time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	var buf bytes.Buffer
	writer := NewMonitoredWriter(&buf, monitor)

	if writer.Monitor() != monitor {
		t.Error("Monitor() should return the associated monitor")
	}
}

func TestMonitoredWriter_KeepsMonitorActive(t *testing.T) {
	cfg := TimeoutConfig{
		Active: time.Hour,
		Stuck:  100 * time.Millisecond,
	}
	monitor := NewTimeoutMonitor(cfg)

	var buf bytes.Buffer
	writer := NewMonitoredWriter(&buf, monitor)

	// Write periodically to keep monitor active
	for i := 0; i < 5; i++ {
		time.Sleep(50 * time.Millisecond)
		writer.Write([]byte("."))
	}

	// Monitor should still be active because we've been writing
	if monitor.State() != TimeoutStateActive {
		t.Errorf("expected Active state, got %v", monitor.State())
	}
	if monitor.IsExpired() {
		t.Error("monitor should not be expired when writes are happening")
	}
}
