package vt

import (
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
	if len(dst.fillAreas) != 1 {
		t.Fatalf("expected 1 FillArea call, got %d", len(dst.fillAreas))
	}
	if got := dst.fillAreas[0]; got != uv.Rect(5, 2, 1, 1) {
		t.Fatalf("expected dirty fill area %v, got %v", uv.Rect(5, 2, 1, 1), got)
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
