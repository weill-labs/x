package vt

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

func TestScrollbackKeepsNewestLinesInOrder(t *testing.T) {
	t.Parallel()

	sb := NewScrollback(3)
	for _, line := range []string{"one", "two", "three", "four"} {
		sb.Push(testScrollbackLine(line, 8))
	}

	if sb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", sb.Len())
	}

	want := []string{"two", "three", "four"}
	for i, line := range want {
		if got := sb.Line(i).String(); got != line {
			t.Fatalf("line %d = %q, want %q", i, got, line)
		}
	}
}

func TestScrollbackSetMaxLinesKeepsNewestLines(t *testing.T) {
	t.Parallel()

	sb := NewScrollback(4)
	for _, line := range []string{"one", "two", "three", "four"} {
		sb.Push(testScrollbackLine(line, 8))
	}

	sb.SetMaxLines(2)

	if sb.Len() != 2 {
		t.Fatalf("expected len 2, got %d", sb.Len())
	}

	want := []string{"three", "four"}
	for i, line := range want {
		if got := sb.Line(i).String(); got != line {
			t.Fatalf("line %d = %q, want %q", i, got, line)
		}
	}
}

func TestScrollbackPushAfterWarmupDoesNotAllocate(t *testing.T) {
	line := testScrollbackLine(strings.Repeat("x", 80), 120)
	sb := NewScrollback(2)
	sb.Push(line)
	sb.Push(line)

	allocs := testing.AllocsPerRun(1000, func() {
		sb.Push(line)
	})

	if allocs != 0 {
		t.Fatalf("expected warmed scrollback push to allocate 0 times, got %.2f", allocs)
	}
}

func BenchmarkScrollbackPushOverflow(b *testing.B) {
	line := testScrollbackLine(strings.Repeat("x", 120), 160)
	sb := NewScrollback(1024)
	for i := 0; i < sb.MaxLines(); i++ {
		sb.Push(line)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sb.Push(line)
	}
}

func testScrollbackLine(content string, width int) uv.Line {
	line := make(uv.Line, max(len(content), width))
	for i := range line {
		line[i] = uv.EmptyCell
	}
	for i := range content {
		line[i] = uv.Cell{
			Content: content[i : i+1],
			Width:   1,
		}
	}
	return line
}
