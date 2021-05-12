package client

import (
	"fmt"

	"weavelab.xyz/ethr/ethr"
)

type UI struct {
	Title               string
	ShowConnectionStats bool
	Logger              ethr.Logger

	lastPrintSeconds    uint64
	currentPrintSeconds uint64
}

func NewUI(title string, connectionStats bool, logger ethr.Logger) *UI {
	return &UI{
		Title:               title,
		ShowConnectionStats: connectionStats,
		Logger:              logger,
	}
}

func (u *UI) PrintTestHeader() {
	s := []string{"ServerAddress", "Proto", "Bits/s", "Conn/s", "Pkt/s"}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("%-15s %-5s %7s %7s %7s\n", s[0], s[1], s[2], s[3], s[4])
}
