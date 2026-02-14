package components

import (
	"fmt"
	"testing"
)

// BenchmarkLogViewport_AppendLine benchmarks single line appends.
func BenchmarkLogViewport_AppendLine(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lv.AppendLine(fmt.Sprintf("Log line %d: This is a test output message", i))
	}
}

// BenchmarkLogViewport_AppendText benchmarks text appends.
func BenchmarkLogViewport_AppendText(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)
	text := "This is a test line of agent output\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lv.AppendText(text)
	}
}

// BenchmarkLogViewport_SetContent benchmarks content replacement.
func BenchmarkLogViewport_SetContent(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)

	// Generate a large content string
	content := ""
	for i := 0; i < 1000; i++ {
		content += fmt.Sprintf("Log line %d: This is a test output message\n", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lv.SetContent(content)
	}
}

// BenchmarkLogViewport_View benchmarks rendering.
func BenchmarkLogViewport_View(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)

	// Add some content
	for i := 0; i < 100; i++ {
		lv.AppendLine(fmt.Sprintf("Log line %d: This is a test output message", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lv.View()
	}
}

// BenchmarkLogViewport_Write benchmarks io.Writer interface.
func BenchmarkLogViewport_Write(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)
	data := []byte("This is a test line of agent output\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = lv.Write(data)
	}
}

// BenchmarkLogViewport_ManyLines benchmarks handling many lines.
func BenchmarkLogViewport_ManyLines(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)

	// Simulate agent output - many lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			lv.AppendLine(fmt.Sprintf("Line %d.%d: output", i, j))
		}
		lv.Clear()
	}
}

// BenchmarkLogViewport_ScrollingWithManyLines benchmarks scrolling operations.
func BenchmarkLogViewport_ScrollingWithManyLines(b *testing.B) {
	lv := NewLogViewport()
	lv.SetSize(120, 40)

	// Add many lines
	for i := 0; i < 10000; i++ {
		lv.AppendLine(fmt.Sprintf("Log line %d: This is a test output message", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lv.GotoTop()
		for j := 0; j < 100; j++ {
			lv.ScrollDown()
		}
		lv.GotoBottom()
	}
}

