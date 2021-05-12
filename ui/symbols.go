package ui

import "github.com/mattn/go-runewidth"

const (
	SymbolLeftTop = iota
	SymbolHorizontal
	SymbolRightTop
	SymbolVertical
	SymbolLeftBottom
	SymbolRightBottom
	SymbolMiddleBottom
	SymbolMiddleTop
	SymbolMiddleLeft
	SymbolMiddleRight
	SymbolMiddleMiddle
	SymbolSpace
	SymbolBox1
	SymbolBox2
	SymbolBox3
	SymbolBox4
	SymbolUpArrow
	SymbolDownArrow
)

var Symbols = []rune{'┌', '─', '┐', '│', '└', '┘', '┴', '┬', '├', '┤', '┼', ' ', '░', '▒', '▓', '█', '↑', '↓'}

func init() {
	if runewidth.IsEastAsian() {
		Symbols = []rune{'+', '-', '+', '|', '+', '+', '+', '+', '+', '+', '+', ' ', '░', '▒', '▓', '█', '^', 'v'}
	}
}
