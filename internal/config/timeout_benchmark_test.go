package config

import (
	"io"
	"testing"
	"time"
)

// BenchmarkTimeoutMonitor_RecordOutput benchmarks the output recording hot path.
func BenchmarkTimeoutMonitor_RecordOutput(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.RecordOutput(1024)
	}
}

// BenchmarkTimeoutMonitor_IsExpired benchmarks the expiry check.
func BenchmarkTimeoutMonitor_IsExpired(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.IsExpired()
	}
}

// BenchmarkTimeoutMonitor_State benchmarks state checking.
func BenchmarkTimeoutMonitor_State(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = monitor.State()
	}
}

// BenchmarkTimeoutMonitor_ConcurrentAccess benchmarks concurrent read/write access.
func BenchmarkTimeoutMonitor_ConcurrentAccess(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				monitor.RecordOutput(100)
			} else {
				_ = monitor.IsExpired()
			}
			i++
		}
	})
}

// BenchmarkMonitoredWriter_Write benchmarks the monitored writer hot path.
func BenchmarkMonitoredWriter_Write(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)
	writer := NewMonitoredWriter(io.Discard, monitor)
	data := []byte("This is a test line of agent output that simulates real output\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkMonitoredWriter_SmallWrites benchmarks many small writes.
func BenchmarkMonitoredWriter_SmallWrites(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)
	writer := NewMonitoredWriter(io.Discard, monitor)
	data := []byte("x")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkMonitoredWriter_LargeWrites benchmarks large buffer writes.
func BenchmarkMonitoredWriter_LargeWrites(b *testing.B) {
	cfg := TimeoutConfig{Active: 2 * time.Hour, Stuck: 30 * time.Minute}
	monitor := NewTimeoutMonitor(cfg)
	writer := NewMonitoredWriter(io.Discard, monitor)
	data := make([]byte, 64*1024) // 64KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}
