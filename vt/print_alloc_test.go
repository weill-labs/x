package vt

import (
	"strings"
	"testing"
)

func TestHandlePrintASCIIAfterWarmupDoesNotAllocate(t *testing.T) {
	e := NewEmulator(8, 1)

	e.handlePrint('A')
	e.scr.setCursor(0, 0, false)
	e.atPhantom = false

	allocs := testing.AllocsPerRun(1000, func() {
		e.scr.setCursor(0, 0, false)
		e.atPhantom = false
		e.handlePrint('A')
	})

	if allocs != 0 {
		t.Fatalf("expected warmed ASCII print to allocate 0 times, got %.2f", allocs)
	}
}

func BenchmarkHandlePrintASCII(b *testing.B) {
	e := NewEmulator(128, 1)
	line := strings.Repeat("A", 80)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.scr.setCursor(0, 0, false)
		e.atPhantom = false
		for _, r := range line {
			e.handlePrint(r)
		}
	}
}
