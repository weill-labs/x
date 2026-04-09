package vt

import uv "github.com/charmbracelet/ultraviolet"

// scrollbackRing keeps scrollback lines in FIFO order without shifting the
// outer slice as the buffer fills.
type scrollbackRing struct {
	lines []uv.Line
	start int
	len   int
}

func newScrollbackRing(maxLines int) scrollbackRing {
	return scrollbackRing{
		lines: make([]uv.Line, maxLines),
	}
}

func (r *scrollbackRing) push(line uv.Line) {
	if len(r.lines) == 0 {
		return
	}

	index := 0
	if r.len == len(r.lines) {
		index = r.start
		r.start = (r.start + 1) % len(r.lines)
	} else {
		index = (r.start + r.len) % len(r.lines)
		r.len++
	}

	r.lines[index] = cloneLineInto(r.lines[index], line)
}

func (r *scrollbackRing) resize(maxLines int) {
	if maxLines == len(r.lines) {
		return
	}

	next := make([]uv.Line, maxLines)
	keep := min(r.len, maxLines)
	for i := range keep {
		next[i] = r.line(r.len - keep + i)
	}

	r.lines = next
	r.start = 0
	r.len = keep
}

func (r *scrollbackRing) line(index int) uv.Line {
	if index < 0 || index >= r.len || len(r.lines) == 0 {
		return nil
	}
	return r.lines[(r.start+index)%len(r.lines)]
}

func (r *scrollbackRing) orderedLines() []uv.Line {
	lines := make([]uv.Line, r.len)
	for i := range r.len {
		lines[i] = r.line(i)
	}
	return lines
}

func (r *scrollbackRing) clear() {
	r.start = 0
	r.len = 0
}

// cloneLineInto reuses a ring slot's backing array once it is large enough for
// the next trimmed line.
func cloneLineInto(dst uv.Line, src uv.Line) uv.Line {
	if cap(dst) < len(src) {
		dst = make(uv.Line, len(src))
	} else {
		dst = dst[:len(src)]
	}
	copy(dst, src)
	return dst
}
