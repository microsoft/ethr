package server

import (
	"fmt"

	"github.com/mattn/go-runewidth"
	"weavelab.xyz/ethr/ui"

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
	t.drawTblRow(ui.Symbols[ui.SymbolVertical], ui.Symbols[ui.SymbolVertical], ui.Symbols[ui.SymbolSpace],
		ui.Symbols[ui.SymbolVertical], tm.ColorDefault, tm.ColorDefault)
	t.cr--

	if len(row) == 0 {
		return
	}

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
		t.printText(t.x+o+alignOffset, t.y+t.cr, w, s, tm.ColorDefault, tm.ColorDefault)
		o += w + 1
		alignOffset = 0
	}

	t.cr++
}

func (t *table) addTblSpr() {
	t.drawTblRow(ui.Symbols[ui.SymbolMiddleLeft], ui.Symbols[ui.SymbolMiddleRight], ui.Symbols[ui.SymbolHorizontal],
		ui.Symbols[ui.SymbolMiddleMiddle], tm.ColorDefault, tm.ColorDefault)
}

func (t *table) addTblHdr() {
	t.drawTblRow(ui.Symbols[ui.SymbolLeftTop], ui.Symbols[ui.SymbolRightTop], ui.Symbols[ui.SymbolHorizontal],
		ui.Symbols[ui.SymbolMiddleTop], tm.ColorDefault, tm.ColorDefault)
}

func (t *table) addTblFtr() {
	t.drawTblRow(ui.Symbols[ui.SymbolLeftBottom], ui.Symbols[ui.SymbolRightBottom], ui.Symbols[ui.SymbolHorizontal],
		ui.Symbols[ui.SymbolMiddleBottom], tm.ColorDefault, tm.ColorDefault)
}

func (t *table) printText(x, y, w int, text string, fg, bg tm.Attribute) {
	textArr := []rune(text)
	for i := 0; i < w; i++ {
		tm.SetCell(x+i, y, ' ', fg, bg)
	}
	xoff := 0
	for i := 0; i < len(textArr); i++ {
		tm.SetCell(x+i+xoff, y, textArr[i], fg, bg)
		if runewidth.RuneWidth(textArr[i]) == 2 {
			xoff++
		}
	}
}
