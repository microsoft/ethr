package ui

import "github.com/mattn/go-runewidth"

const (
	symbolLeftTop = iota
	symbolHorizontal
	symbolRightTop
	symbolVertical
	symbolLeftBottom
	symbolRightBottom
	symbolMiddleBottom
	symbolMiddleTop
	symbolMiddleLeft
	symbolMiddleRight
	symbolMiddleMiddle
	symbolSpace
	symbolBox1
	symbolBox2
	symbolBox3
	symbolBox4
	symbolUpArrow
	symbolDownArrow
)

var symbols = []rune{'┌', '─', '┐', '│', '└', '┘', '┴', '┬', '├', '┤', '┼', ' ', '░', '▒', '▓', '█', '↑', '↓'}

func init() {
	if runewidth.IsEastAsian() {
		symbols = []rune{'+', '-', '+', '|', '+', '+', '+', '+', '+', '+', '+', ' ', '░', '▒', '▓', '█', '^', 'v'}
	}
}
