package vt

import (
	"fmt"
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

func TestReverseIndexClampsOversizedScrollMarginsAfterShrink(t *testing.T) {
	t.Parallel()

	term := NewEmulator(40, 24)
	term.Resize(40, 13)

	if _, err := term.WriteString("\x1b[1;21r\x1b[H\x1bM"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}

	pos := term.CursorPosition()
	if pos.X != 0 || pos.Y != 0 {
		t.Fatalf("CursorPosition() = (%d, %d), want (0, 0)", pos.X, pos.Y)
	}
	if got := term.CellAt(0, 0).Content; got != " " {
		t.Fatalf("CellAt(0, 0).Content = %q, want blank top cell after reverse index scroll", got)
	}
}

func TestReflowWrappedPositionHandlesEmptyWrappedCounts(t *testing.T) {
	t.Parallel()

	pos := reflowWrappedPosition(nil, reflowPosition{logical: 3, offset: 12}, 20)
	if pos.X != 0 || pos.Y != 0 {
		t.Fatalf("reflowWrappedPosition(nil, ...) = (%d, %d), want (0, 0)", pos.X, pos.Y)
	}
}

func TestResizeShrinkThenWidenKeepsDenseRowsSeparate(t *testing.T) {
	t.Parallel()

	const (
		width       = 214
		shrinkWidth = 80
		height      = 12
	)
	term := NewEmulator(width, height)
	lines := make([]string, 0, 5)
	for i := 1; i <= 5; i++ {
		line := resizeSmearReproLine(i, width)
		lines = append(lines, line)
		if _, err := term.WriteString(line + "\r\n"); err != nil {
			t.Fatalf("WriteString(line %d) error = %v", i, err)
		}
	}

	for i, want := range lines {
		if got := visibleRowText(term, i, width); got != want {
			t.Fatalf("before resize row %d = %q, want %q", i, got, want)
		}
	}

	term.Resize(shrinkWidth, height)
	for i := range lines {
		if screenLineUsesFullWidth(term.scr.buf.Line(i), shrinkWidth) {
			t.Fatalf("after shrink row %d still uses full width", i)
		}
	}
	term.Resize(width, height)

	for i := range lines {
		got := visibleRowText(term, i, width)
		marker := fmt.Sprintf("LINE_%d_BEGIN_", i+1)
		if !strings.HasPrefix(got, marker) || strings.Count(got, "LINE_") != 1 {
			t.Fatalf("after shrink/widen row %d = %q, want separate row beginning %q", i, got, marker)
		}
	}
}

func resizeSmearReproLine(i, width int) string {
	letter := string(rune('A' + i - 1))
	prefix := fmt.Sprintf("LINE_%d_BEGIN_", i)
	suffix := fmt.Sprintf("_END_%d", i)
	return prefix + strings.Repeat(letter, width-len(prefix)-len(suffix)) + suffix
}

func TestAltScreenEntryPreservesHiddenCursorWhenHideArrivesFirst(t *testing.T) {
	t.Parallel()

	term := NewEmulator(40, 24)

	if _, err := term.WriteString("\x1b[?25l\x1b[?1049h"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}

	if !term.Cursor().Hidden {
		t.Fatal("Cursor().Hidden = false after hide-before-alt-screen, want true")
	}
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
