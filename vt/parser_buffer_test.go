package vt

import (
	"strings"
	"testing"
)

func TestNewEmulatorCapsParserDataBuffer(t *testing.T) {
	const wantParserDataSize = 128 * 1024

	e := NewEmulator(80, 24)
	if got := cap(e.parser.Data()); got != wantParserDataSize {
		t.Fatalf("parser data buffer cap = %d, want %d", got, wantParserDataSize)
	}

	oversizedTitle := "\x1b]0;" + strings.Repeat("x", wantParserDataSize*2) + "\a"
	if _, err := e.Write([]byte(oversizedTitle)); err != nil {
		t.Fatalf("Write(oversized OSC title): %v", err)
	}
	if got := cap(e.parser.Data()); got != wantParserDataSize {
		t.Fatalf("parser data buffer cap after oversized OSC = %d, want %d", got, wantParserDataSize)
	}
	if got := len(e.parser.Data()); got > wantParserDataSize {
		t.Fatalf("parser data length after oversized OSC = %d, want <= %d", got, wantParserDataSize)
	}
}
