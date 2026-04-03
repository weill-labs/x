package vt

import (
	"strings"
	"testing"
	"time"
)

func visibleRowText(term *Emulator, row, width int) string {
	var b strings.Builder
	for x := 0; x < width; x++ {
		cell := term.CellAt(x, row)
		if cell == nil || cell.Width == 0 {
			continue
		}
		if cell.Content == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(cell.Content)
	}
	return strings.TrimRight(b.String(), " ")
}

func TestSynchronizedOutputBuffersScreenChangesUntilReset(t *testing.T) {
	t.Parallel()

	term := NewEmulator(40, 24)

	if _, err := term.WriteString("old"); err != nil {
		t.Fatalf("WriteString(initial) error = %v", err)
	}

	if _, err := term.WriteString("\x1b[?2026h\x1b[2J\x1b[Hnew"); err != nil {
		t.Fatalf("WriteString(begin sync output) error = %v", err)
	}

	if got := visibleRowText(term, 0, 20); got != "old" {
		t.Fatalf("visibleRowText() before reset = %q, want %q", got, "old")
	}
	if pos := term.CursorPosition(); pos.X != 3 || pos.Y != 0 {
		t.Fatalf("CursorPosition() before reset = (%d, %d), want (3, 0)", pos.X, pos.Y)
	}

	if _, err := term.WriteString(" text"); err != nil {
		t.Fatalf("WriteString(buffered payload) error = %v", err)
	}
	if got := visibleRowText(term, 0, 20); got != "old" {
		t.Fatalf("visibleRowText() while buffered = %q, want %q", got, "old")
	}

	if _, err := term.WriteString("\x1b[?2026l"); err != nil {
		t.Fatalf("WriteString(end sync output) error = %v", err)
	}

	if got := visibleRowText(term, 0, 20); got != "new text" {
		t.Fatalf("visibleRowText() after reset = %q, want %q", got, "new text")
	}
	if pos := term.CursorPosition(); pos.X != 8 || pos.Y != 0 {
		t.Fatalf("CursorPosition() after reset = (%d, %d), want (8, 0)", pos.X, pos.Y)
	}
}

func TestSynchronizedOutputFlushesBufferedScreenChangesAfterTimeout(t *testing.T) {
	t.Parallel()

	term := NewEmulator(40, 24)

	if _, err := term.WriteString("old"); err != nil {
		t.Fatalf("WriteString(initial) error = %v", err)
	}
	if _, err := term.WriteString("\x1b[?2026h\x1b[2J\x1b[Hnew"); err != nil {
		t.Fatalf("WriteString(begin sync output) error = %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	if got := visibleRowText(term, 0, 20); got != "new" {
		t.Fatalf("visibleRowText() after timeout = %q, want %q", got, "new")
	}
	if pos := term.CursorPosition(); pos.X != 3 || pos.Y != 0 {
		t.Fatalf("CursorPosition() after timeout = (%d, %d), want (3, 0)", pos.X, pos.Y)
	}

	if _, err := term.WriteString("!"); err != nil {
		t.Fatalf("WriteString(after timeout) error = %v", err)
	}
	if got := visibleRowText(term, 0, 20); got != "new!" {
		t.Fatalf("visibleRowText() after timeout follow-up write = %q, want %q", got, "new!")
	}
}
