package ui

import (
	"github.com/mattn/go-runewidth"
	tm "github.com/nsf/termbox-go"
)

func printHLineText(x, y int, w int, text string) {
	for i := 0; i < w; i++ {
		tm.SetCell(x+i, y, symbols[symbolHorizontal], tm.ColorWhite, tm.ColorDefault)
	}
	offset := (w - runewidth.StringWidth(text)) / 2
	textArr := []rune(text)
	xoff := 0
	for i := 0; i < len(text); i++ {
		tm.SetCell(x+offset+i+xoff, y, textArr[i], tm.ColorWhite, tm.ColorDefault)
		if runewidth.RuneWidth(textArr[i]) == 2 {
			xoff++
		}
	}
}

func printVLine(x, y int, h int) {
	tm.SetCell(x, y, symbols[symbolMiddleTop], tm.ColorWhite, tm.ColorDefault)
	for i := 1; i < h; i++ {
		tm.SetCell(x, y+i, symbols[symbolVertical], tm.ColorWhite, tm.ColorDefault)
	}
}

func printText(x, y, w int, text string, fg, bg tm.Attribute) {
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

func printCenterText(x, y, w int, text string, fg, bg tm.Attribute) {
	offset := (w - runewidth.StringWidth(text)) / 2
	textArr := []rune(text)
	for i := 0; i < w; i++ {
		tm.SetCell(x+i, y, ' ', fg, bg)
	}
	xoff := 0
	for i := 0; i < len(textArr); i++ {
		tm.SetCell(x+offset+i+xoff, y, textArr[i], fg, bg)
		if runewidth.RuneWidth(textArr[i]) == 2 {
			xoff++
		}
	}
}
