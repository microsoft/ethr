package ui

import (
	"fmt"

	tm "github.com/nsf/termbox-go"
)

type table struct {
	ccount  int
	cwidth  []int
	x       int
	y       int
	cr      int
	justify int
	border  int
}

const (
	tableJustifyLeft = iota
	tableJustifyRight
	tableJustifyCenter
)

const (
	tableBorder = iota
	tableNoBorder
)

func (t *table) drawTblRow(ledge, redge, middle, spr rune, fg, bg tm.Attribute) {
	twidth := t.ccount + 1
	for _, w := range t.cwidth {
		twidth += w
	}

	for i := 0; i < twidth; i++ {
		tm.SetCell(t.x+i, t.y+t.cr, middle, fg, bg)
	}

	if t.border == tableBorder {
		tm.SetCell(t.x, t.y+t.cr, ledge, fg, bg)
		tm.SetCell(t.x+twidth, t.y+t.cr, redge, fg, bg)
	}

	o := 0
	for c, w := range t.cwidth {
		o += w + 1
		if c < t.ccount-1 {
			tm.SetCell(t.x+o, t.y+t.cr, spr, fg, bg)
		}
	}
	t.cr++
}

func (t *table) addTblRow(row []string) {
	t.drawTblRow(symbols[symbolVertical], symbols[symbolVertical], symbols[symbolSpace],
		symbols[symbolVertical], tm.ColorDefault, tm.ColorDefault)
	t.cr--

	o := 1
	alignOffset := 0
	for i := 0; i < t.ccount; i++ {
		w := t.cwidth[i]
		var s string
		if t.justify == tableJustifyLeft {
			s = fmt.Sprintf("%-*s", w, row[i])
			if i == 0 && t.border == tableNoBorder {
				alignOffset = -1
			}
		} else {
			s = fmt.Sprintf("%*s", w, row[i])
		}
		printText(t.x+o+alignOffset, t.y+t.cr, w, s, tm.ColorDefault, tm.ColorDefault)
		o += w + 1
		alignOffset = 0
	}

	t.cr++
}

func (t *table) addTblSpr() {
	t.drawTblRow(symbols[symbolMiddleLeft], symbols[symbolMiddleRight], symbols[symbolHorizontal],
		symbols[symbolMiddleMiddle], tm.ColorDefault, tm.ColorDefault)
}

func (t *table) addTblHdr() {
	t.drawTblRow(symbols[symbolLeftTop], symbols[symbolRightTop], symbols[symbolHorizontal],
		symbols[symbolMiddleTop], tm.ColorDefault, tm.ColorDefault)
}

func (t *table) addTblFtr() {
	t.drawTblRow(symbols[symbolLeftBottom], symbols[symbolRightBottom], symbols[symbolHorizontal],
		symbols[symbolMiddleBottom], tm.ColorDefault, tm.ColorDefault)
}
