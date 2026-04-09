package vt

import "github.com/charmbracelet/x/ansi"

type printContentCache struct {
	ascii [ansi.DEL]string
}

func newPrintContentCache() *printContentCache {
	cache := new(printContentCache)
	for r := ansi.SP; r < ansi.DEL; r++ {
		cache.ascii[r] = string(rune(r))
	}
	return cache
}

func (c *printContentCache) printableASCII(r rune) string {
	if c == nil {
		return string(r)
	}
	return c.ascii[r]
}
