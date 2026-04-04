package vt

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

const (
	benchmarkWidth  = 120
	benchmarkHeight = 40

	benchmarkPayload256 = 256
	benchmarkPayload4KB = 4 * 1024
	benchmarkPayload32K = 32 * 1024

	// Reset state and clear scrollback so repeated benchmark iterations stay comparable.
	benchmarkResetPrefix  = "\x1bc\x1b[3J\x1b[H"
	benchmarkFillerPrefix = "\x1b[38;5;117m"
	benchmarkFillerSuffix = "\x1b[0m\r\n"
)

var (
	benchmarkFixtureOnce sync.Once
	benchmarkFixtureData []byte
	benchmarkFixtureErr  error
	benchmarkParserData  []byte

	benchmarkPayloadOnce sync.Once
	benchmarkPayloadData map[int][]byte

	benchmarkCellSink *uv.Cell
	benchmarkIntSink  int
)

var benchmarkLines = []string{
	"\x1b[38;5;81m12:34:56\x1b[0m \x1b[1;38;5;204mINFO\x1b[0m  \x1b[38;5;252mworker-07\x1b[0m connected to \x1b[38;5;114mssh://prod-a\x1b[0m\r\n",
	"\x1b[38;5;240m~/src/amux\x1b[0m on \x1b[38;5;141mmain\x1b[0m \x1b[38;5;214m+2/-1\x1b[0m \x1b[38;5;45mvia go1.24.2\x1b[0m\r\n",
	"\x1b[48;5;236m\x1b[38;5;118m build \x1b[0m \x1b[38;5;252m[##########------]\x1b[0m \x1b[38;5;81m62%\x1b[0m  \x1b[38;5;110m12.4 MiB/s\x1b[0m\r\n",
	"\x1b[38;5;203mWARN\x1b[0m retrying \x1b[38;5;223mGET /api/sessions/42\x1b[0m after \x1b[38;5;214m250ms\x1b[0m from \x1b[38;5;109mus-east-1\x1b[0m\r\n",
	"\x1b[38;5;69muser@host\x1b[0m:\x1b[38;5;111m~/repo/vt\x1b[0m$ \x1b[1;38;5;230mgo test ./... -run TestDraw\x1b[0m\r\n",
	"\x1b[38;5;245m2026-04-04T09:14:12Z\x1b[0m \x1b[38;5;220mTRACE\x1b[0m pane=\x1b[38;5;117m7\x1b[0m cursor=\x1b[38;5;150m(42,11)\x1b[0m repaint=\x1b[38;5;183mfull\x1b[0m\r\n",
}

func BenchmarkParseBytes(b *testing.B) {
	payload := benchmarkParserPayload(b)
	e := NewEmulator(benchmarkWidth, benchmarkHeight)

	b.ReportAllocs()
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e.parseBytes(payload)
	}
}

func BenchmarkCellAt(b *testing.B) {
	payload := benchmarkPayload(b, benchmarkPayload4KB)
	e := benchmarkPreparedEmulator(b, payload)
	x, y := benchmarkStyledCellPosition(b, e)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchmarkCellSink = e.CellAt(x, y)
	}
}

func BenchmarkTouchedIterate(b *testing.B) {
	payload := benchmarkPayload(b, benchmarkPayload4KB)
	e := benchmarkPreparedEmulator(b, payload)
	width := e.Width()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		total := 0
		for y, line := range e.Touched() {
			if line == nil {
				continue
			}
			start, end := dirtySpan(line, width)
			total += y + start + end
		}
		benchmarkIntSink = total
	}
}

func BenchmarkDraw(b *testing.B) {
	payload := benchmarkPayload(b, benchmarkPayload32K)
	e := benchmarkPreparedEmulator(b, payload)
	dst := uv.NewScreenBuffer(benchmarkWidth, benchmarkHeight)
	area := uv.Rect(0, 0, benchmarkWidth, benchmarkHeight)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Draw clears touched state, so each iteration re-invalidates the
		// existing screen before measuring the full redraw path.
		e.scr.invalidate()
		e.Draw(dst, area)
	}
}

func BenchmarkEmulatorWrite(b *testing.B) {
	for _, tc := range []struct {
		name string
		size int
	}{
		{name: "256B", size: benchmarkPayload256},
		{name: "4KB", size: benchmarkPayload4KB},
		{name: "32KB", size: benchmarkPayload32K},
	} {
		payload := benchmarkPayload(b, tc.size)

		b.Run(tc.name, func(b *testing.B) {
			e := NewEmulator(benchmarkWidth, benchmarkHeight)

			b.ReportAllocs()
			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if _, err := e.Write(payload); err != nil {
					b.Fatalf("Write(%s): %v", tc.name, err)
				}
			}
		})
	}
}

func benchmarkParserPayload(b testing.TB) []byte {
	b.Helper()

	benchmarkFixtureOnce.Do(func() {
		path := filepath.Join("..", "ansi", "fixtures", "demo.vte")
		benchmarkFixtureData, benchmarkFixtureErr = os.ReadFile(path)
		if benchmarkFixtureErr == nil {
			benchmarkParserData = append([]byte(benchmarkResetPrefix), benchmarkFixtureData...)
		}
	})

	if benchmarkFixtureErr != nil {
		b.Fatalf("read parser fixture: %v", benchmarkFixtureErr)
	}

	return benchmarkParserData
}

func benchmarkPayload(b testing.TB, size int) []byte {
	b.Helper()

	benchmarkPayloadOnce.Do(func() {
		benchmarkPayloadData = map[int][]byte{
			benchmarkPayload256: buildBenchmarkPayload(benchmarkPayload256),
			benchmarkPayload4KB: buildBenchmarkPayload(benchmarkPayload4KB),
			benchmarkPayload32K: buildBenchmarkPayload(benchmarkPayload32K),
		}
	})

	payload, ok := benchmarkPayloadData[size]
	if !ok {
		b.Fatalf("missing payload for size %d", size)
	}

	return payload
}

func buildBenchmarkPayload(size int) []byte {
	minSize := len(benchmarkResetPrefix) + len(benchmarkFillerPrefix) + len(benchmarkFillerSuffix)
	if size < minSize {
		panic("benchmark payload size too small")
	}
	minFiller := len(benchmarkFillerPrefix) + len(benchmarkFillerSuffix)

	var buf strings.Builder
	buf.Grow(size)
	buf.WriteString(benchmarkResetPrefix)

	for i := 0; ; i++ {
		line := benchmarkLines[i%len(benchmarkLines)]
		if buf.Len()+len(line)+minFiller > size {
			break
		}
		buf.WriteString(line)
	}

	remaining := size - buf.Len()
	if remaining > 0 {
		fillLen := remaining - minFiller
		if fillLen < 0 {
			panic("benchmark payload underflow")
		}
		buf.WriteString(benchmarkFillerPrefix)
		if fillLen > 0 {
			buf.WriteString(strings.Repeat("x", fillLen))
		}
		buf.WriteString(benchmarkFillerSuffix)
	}

	if buf.Len() != size {
		panic("benchmark payload size mismatch")
	}

	return []byte(buf.String())
}

func benchmarkPreparedEmulator(b testing.TB, payload []byte) *Emulator {
	b.Helper()

	e := NewEmulator(benchmarkWidth, benchmarkHeight)
	if _, err := e.Write(payload); err != nil {
		b.Fatalf("seed emulator: %v", err)
	}
	return e
}

func benchmarkStyledCellPosition(b testing.TB, e *Emulator) (int, int) {
	b.Helper()

	var fallbackX, fallbackY int
	foundFallback := false

	for y := 0; y < e.Height(); y++ {
		for x := 0; x < e.Width(); x++ {
			cell := e.CellAt(x, y)
			if cell == nil || cell.IsZero() || cell.Content == "" {
				continue
			}
			if cell.Style.Fg != nil || cell.Style.Bg != nil {
				return x, y
			}
			if !foundFallback {
				fallbackX, fallbackY = x, y
				foundFallback = true
			}
		}
	}

	if foundFallback {
		return fallbackX, fallbackY
	}

	b.Fatal("failed to find populated benchmark cell")
	return 0, 0
}
