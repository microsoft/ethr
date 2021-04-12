package ui

import (
	"math"

	tm "github.com/nsf/termbox-go"
)

func PrintUsageBar(x, y, w int, usage, scale uint64, clr tm.Attribute) {
	barw := int(math.Log10(float64((usage + scale - 1) / (scale / 10))))
	if barw > w {
		barw = w
	} else if barw < 0 {
		barw = 0
	}
	for j := 0; j < w; j++ {
		tm.SetCell(x+j, y, symbols[symbolBox3], clr, tm.ColorDefault)
	}
	for j := 0; j < barw; j++ {
		tm.SetCell(x+j, y, symbols[symbolBox3], clr|tm.AttrBold, clr)
	}
}
