package vt

import (
	"slices"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

type countingScreen struct {
	uv.ScreenBuffer
	setCalls  int
	fillAreas []uv.Rectangle
}

func newCountingScreen(width, height int) *countingScreen {
	return &countingScreen{
		ScreenBuffer: uv.NewScreenBuffer(width, height),
	}
}

func (s *countingScreen) SetCell(x, y int, c *uv.Cell) {
	s.setCalls++
	s.ScreenBuffer.SetCell(x, y, c)
}

func (s *countingScreen) FillArea(cell *uv.Cell, area uv.Rectangle) {
	s.fillAreas = append(s.fillAreas, area)
	s.ScreenBuffer.FillArea(cell, area)
}

func TestEmulatorDrawUsesDirtySpans(t *testing.T) {
	e := NewEmulator(10, 4)
	e.scr.ClearTouched()

	want := &uv.Cell{Content: "X", Width: 1}
	e.SetCell(5, 2, want)

	dst := newCountingScreen(10, 4)
	e.Draw(dst, uv.Rect(0, 0, 10, 4))

	if dst.setCalls != 1 {
		t.Fatalf("expected 1 SetCell call, got %d", dst.setCalls)
	}
	if len(dst.fillAreas) != 0 {
		t.Fatalf("expected no FillArea calls, got %d", len(dst.fillAreas))
	}

	got := dst.CellAt(5, 2)
	if got == nil || !got.Equal(want) {
		t.Fatalf("expected drawn cell %v at 5,2, got %#v", want, got)
	}

	for y, line := range e.Touched() {
		if line != nil {
			t.Fatalf("expected touched state to be cleared after draw, line %d = %#v", y, line)
		}
	}

	dst = newCountingScreen(10, 4)
	e.Draw(dst, uv.Rect(0, 0, 10, 4))
	if dst.setCalls != 0 {
		t.Fatalf("expected no SetCell calls on redraw without changes, got %d", dst.setCalls)
	}
	if len(dst.fillAreas) != 0 {
		t.Fatalf("expected no FillArea calls on redraw without changes, got %d", len(dst.fillAreas))
	}
}

func TestEmulatorDrawClearsDirtyBlankCellsWithoutFillArea(t *testing.T) {
	e := NewEmulator(10, 4)
	e.SetCell(5, 2, &uv.Cell{Content: "X", Width: 1})
	e.scr.ClearTouched()
	e.SetCell(5, 2, nil)

	dst := newCountingScreen(10, 4)
	dst.SetCell(5, 2, &uv.Cell{Content: "Y", Width: 1})
	dst.setCalls = 0

	e.Draw(dst, uv.Rect(0, 0, 10, 4))

	if len(dst.fillAreas) != 0 {
		t.Fatalf("expected no FillArea calls, got %d", len(dst.fillAreas))
	}
	if dst.setCalls != 1 {
		t.Fatalf("expected 1 SetCell call for cleared cell, got %d", dst.setCalls)
	}
	want := uv.EmptyCell
	want.Style.Bg = e.defaultBg
	if got := dst.CellAt(5, 2); got == nil || !got.Equal(&want) {
		t.Fatalf("expected cleared destination cell %#v at 5,2, got %#v", want, got)
	}
}

func TestScreenEachTouchedLineVisitsDirtyRowsOnly(t *testing.T) {
	s := NewScreen(8, 6)
	s.ClearTouched()
	s.SetCell(2, 1, &uv.Cell{Content: "A", Width: 1})
	s.SetCell(4, 4, &uv.Cell{Content: "B", Width: 1})

	var got []int
	s.eachTouchedLine(func(y int, line *uv.LineData) {
		if line == nil {
			t.Fatalf("eachTouchedLine(%d) received nil line", y)
		}
		got = append(got, y)
	})

	if want := []int{1, 4}; !slices.Equal(got, want) {
		t.Fatalf("eachTouchedLine rows = %v, want %v", got, want)
	}

	s.ClearTouched()
	got = got[:0]
	s.eachTouchedLine(func(y int, line *uv.LineData) {
		got = append(got, y)
	})
	if len(got) != 0 {
		t.Fatalf("eachTouchedLine after ClearTouched = %v, want none", got)
	}
}

func TestScreenCellShiftsTouchRestOfLine(t *testing.T) {
	s := NewScreen(10, 1)
	s.ClearTouched()
	s.cur.X, s.cur.Y = 2, 0

	s.InsertCell(1)
	if got := s.Touched()[0]; got == nil || got.FirstCell != 2 || got.LastCell != 10 {
		t.Fatalf("expected insert-cell dirty range [2,10), got %#v", got)
	}

	s.ClearTouched()
	s.DeleteCell(1)
	if got := s.Touched()[0]; got == nil || got.FirstCell != 2 || got.LastCell != 10 {
		t.Fatalf("expected delete-cell dirty range [2,10), got %#v", got)
	}
}

func TestAltScreenReturnInvalidatesMainScreen(t *testing.T) {
	e := NewEmulator(6, 2)
	e.scr.ClearTouched()

	want := &uv.Cell{Content: "M", Width: 1}
	e.SetCell(1, 0, want)

	// Consume the initial main-screen damage so the redraw below depends on the
	// alt-screen return path invalidating the main screen again.
	e.Draw(newCountingScreen(6, 2), uv.Rect(0, 0, 6, 2))

	if _, err := e.WriteString("\x1b[?1049h"); err != nil {
		t.Fatalf("enter alt screen: %v", err)
	}
	e.scr.ClearTouched()
	if _, err := e.WriteString("\x1b[?1049l"); err != nil {
		t.Fatalf("leave alt screen: %v", err)
	}

	dst := newCountingScreen(6, 2)
	e.Draw(dst, uv.Rect(0, 0, 6, 2))

	if len(dst.fillAreas) != 0 {
		t.Fatalf("expected no FillArea calls, got %d", len(dst.fillAreas))
	}
	if dst.setCalls != 12 {
		t.Fatalf("expected full redraw via 12 SetCell calls, got %d", dst.setCalls)
	}

	got := dst.CellAt(1, 0)
	if got == nil || !got.Equal(want) {
		t.Fatalf("expected main-screen cell %v at 1,0 after alt-screen return, got %#v", want, got)
	}
	blank := uv.EmptyCell
	blank.Style.Bg = e.defaultBg
	if got := dst.CellAt(0, 1); got == nil || !got.Equal(&blank) {
		t.Fatalf("expected cleared second line after alt-screen return, got %#v", got)
	}
}

func TestTouchRowForcesDrawOnCleanRow(t *testing.T) {
	t.Parallel()

	e := NewEmulator(6, 3)

	// Write content to row 1 and draw it once to establish baseline.
	e.SetCell(0, 1, &uv.Cell{Content: "A", Width: 1})
	dst := newCountingScreen(6, 3)
	e.Draw(dst, uv.Rect(0, 0, 6, 3))

	// dst now has "A" at (0,1). Overwrite it so we can detect a redraw.
	dst.SetCell(0, 1, &uv.Cell{Content: "Z", Width: 1})
	dst.setCalls = 0

	// After Draw, touched state is cleared. A second Draw should be a no-op.
	e.Draw(dst, uv.Rect(0, 0, 6, 3))
	if dst.setCalls != 0 {
		t.Fatalf("expected no SetCell calls on clean screen, got %d", dst.setCalls)
	}
	// dst still shows "Z" because Draw skipped the clean row.
	if got := dst.CellAt(0, 1); got == nil || got.Content != "Z" {
		t.Fatalf("expected dst to retain Z on clean row, got %#v", got)
	}

	// TouchRow marks row 1 dirty from outside the emulator. This is
	// the API compositors use to ensure overlay rows get redrawn.
	e.TouchRow(1)

	dst.setCalls = 0
	e.Draw(dst, uv.Rect(0, 0, 6, 3))

	// Draw should have written to row 1, restoring "A" from the emulator.
	if dst.setCalls == 0 {
		t.Fatalf("expected SetCell calls after TouchRow, got 0")
	}
	if got := dst.CellAt(0, 1); got == nil || got.Content != "A" {
		t.Fatalf("expected dst to show A after TouchRow + Draw, got %#v", got)
	}
}
