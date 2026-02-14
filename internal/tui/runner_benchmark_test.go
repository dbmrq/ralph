package tui

import (
	"bytes"
	"strings"
	"testing"
)

// BenchmarkTUIOutputWriter_Write benchmarks the TUI output writer.
func BenchmarkTUIOutputWriter_Write(b *testing.B) {
	writer := NewTUIOutputWriter(nil) // nil program, just buffer operations
	data := []byte("This is a test line of agent output that simulates real output\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkTUIOutputWriter_PartialLines benchmarks partial line buffering.
func BenchmarkTUIOutputWriter_PartialLines(b *testing.B) {
	writer := NewTUIOutputWriter(nil)
	data := []byte("partial line without newline")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
		if i%10 == 0 {
			writer.Flush()
		}
	}
}

// BenchmarkTUIOutputWriter_MultipleLines benchmarks multi-line writes.
func BenchmarkTUIOutputWriter_MultipleLines(b *testing.B) {
	writer := NewTUIOutputWriter(nil)
	lines := []string{
		"Line 1: Starting process...\n",
		"Line 2: Processing files...\n",
		"Line 3: Running tests...\n",
		"Line 4: Complete\n",
	}
	data := []byte(strings.Join(lines, ""))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkLineWriter_Write benchmarks the line writer.
func BenchmarkLineWriter_Write(b *testing.B) {
	lineCount := 0
	writer := NewLineWriter(func(line string) {
		lineCount++
	})
	data := []byte("This is a test line of agent output that simulates real output\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkLineWriter_PartialLines benchmarks partial line handling.
func BenchmarkLineWriter_PartialLines(b *testing.B) {
	writer := NewLineWriter(func(line string) {})
	data := []byte("partial")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = writer.Write(data)
	}
}

// BenchmarkBufferReadString simulates the current buffer read pattern.
func BenchmarkBufferReadString(b *testing.B) {
	data := "line1\nline2\nline3\npartial"
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.WriteString(data)
		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				buf.WriteString(line)
				break
			}
			_ = line
		}
	}
}

// BenchmarkManualLineScan simulates optimized line scanning.
func BenchmarkManualLineScan(b *testing.B) {
	data := []byte("line1\nline2\nline3\npartial")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		remaining := data
		for {
			idx := bytes.IndexByte(remaining, '\n')
			if idx < 0 {
				break
			}
			_ = remaining[:idx]
			remaining = remaining[idx+1:]
		}
	}
}

