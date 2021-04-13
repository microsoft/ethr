package server

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"weavelab.xyz/ethr/config"

	tm "github.com/nsf/termbox-go"
)

type Tui struct {
	h, w                               int
	resX, resY, resW                   int
	latX, latY, latW                   int
	topVSplitX, topVSplitY, topVSplitH int
	statX, statY, statW                int
	msgX, msgY, msgW                   int
	botVSplitX, botVSplitY, botVSplitH int
	errX, errY, errW                   int
	res                                table
	results                            [][]string
	resultHdr                          []string
	msg                                table
	msgRing                            []string
	err                                table
	errRing                            []string
	ringLock                           sync.RWMutex
}

func InitTui() (*Tui, error) {
	err := tm.Init()
	if err != nil {
		return nil, err
	}

	w, h := tm.Size()
	if h < 40 || w < 80 {
		tm.Close()
		s := fmt.Sprintf("Terminal too small (%dwx%dh), must be at least 40hx80w", w, h)
		return nil, errors.New(s)
	}

	tm.SetInputMode(tm.InputEsc | tm.InputMouse)
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	tm.Sync()
	tm.Flush()
	hideCursor()

	tui := Tui{}
	botScnH := 8
	statScnW := 26
	tui.h = h
	tui.w = w
	tui.resX = 0
	tui.resY = 2
	tui.resW = w - statScnW
	tui.latX = 0
	tui.latY = h - botScnH
	tui.latW = w
	tui.topVSplitX = tui.resW
	tui.topVSplitY = 1
	tui.topVSplitH = h - botScnH
	tui.statX = tui.topVSplitX + 1
	tui.statY = 2
	tui.statW = statScnW
	tui.msgX = 0
	tui.msgY = h - botScnH + 1
	tui.msgW = (w+1)/2 + 1
	tui.botVSplitX = tui.msgW
	tui.botVSplitY = h - botScnH
	tui.botVSplitH = botScnH
	tui.errX = tui.botVSplitX + 1
	tui.errY = h - botScnH + 1
	tui.errW = w - tui.msgW - 1
	tui.res = table{6, []int{13, 5, 7, 7, 7, 8}, 0, 2, 0, tableJustifyRight, tableNoBorder}
	tui.results = make([][]string, 0)
	tui.msg = table{1, []int{tui.msgW}, tui.msgX, tui.msgY, 0, tableJustifyLeft, tableNoBorder}
	tui.msgRing = make([]string, botScnH-1)
	tui.err = table{1, []int{tui.errW}, tui.errX, tui.errY, 0, tableJustifyLeft, tableNoBorder}
	tui.errRing = make([]string, botScnH-1)
	//ui = tui

	go func() {
		for {
			switch ev := tm.PollEvent(); ev.Type {
			case tm.EventKey:
				if ev.Key == tm.KeyEsc || ev.Key == tm.KeyCtrlC {
					tm.Close()
					os.Exit(0)
				}
			case tm.EventResize:
			}
		}
	}()

	return &tui, nil
}

func hideCursor() {
	tm.SetCursor(0, 0)
}

func (t *Tui) Paint(seconds uint64) {
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	defer tm.Flush()
	printCenterText(0, 0, u.w, "Ethr (Version: "+config.Version+")", tm.ColorBlack, tm.ColorWhite)
	printHLineText(u.resX, u.resY-1, u.resW, "Test Results")
	printHLineText(u.statX, u.statY-1, u.statW, "Statistics")
	printVLine(u.topVSplitX, u.topVSplitY, u.topVSplitH)

	printHLineText(u.msgX, u.msgY-1, u.msgW, "Messages")
	printHLineText(u.errX, u.errY-1, u.errW, "Errors")

	u.ringLock.Lock()
	u.msg.cr = 0
	for _, s := range u.msgRing {
		u.msg.addTblRow([]string{s})
	}

	u.err.cr = 0
	for _, s := range u.errRing {
		u.err.addTblRow([]string{s})
	}
	u.ringLock.Unlock()

	printVLine(u.botVSplitX, u.botVSplitY, u.botVSplitH)

	u.res.cr = 0
	if u.resultHdr != nil {
		u.res.addTblHdr()
		u.res.addTblRow(u.resultHdr)
		u.res.addTblSpr()
	}
	for _, s := range u.results {
		u.res.addTblRow(s)
		u.res.addTblSpr()
	}

	if len(gPrevNetStats.netDevStats) == 0 {
		return
	}

	x := u.statX
	w := u.statW
	y := u.statY
	for _, ns := range gCurNetStats.netDevStats {
		nsDiff := getNetDevStatDiff(ns, gPrevNetStats, seconds)
		// TODO: Log the network adapter stats in file as well.
		printText(x, y, w, fmt.Sprintf("if: %s", ns.interfaceName), tm.ColorWhite, tm.ColorBlack)
		y++
		printText(x, y, w, fmt.Sprintf("Tx %sbps", bytesToRate(nsDiff.txBytes)), tm.ColorWhite, tm.ColorBlack)
		bw := nsDiff.txBytes * 8
		printUsageBar(x+14, y, 10, bw, KILO, tm.ColorYellow)
		y++
		printText(x, y, w, fmt.Sprintf("Rx %sbps", bytesToRate(nsDiff.rxBytes)), tm.ColorWhite, tm.ColorBlack)
		bw = nsDiff.rxBytes * 8
		printUsageBar(x+14, y, 10, bw, KILO, tm.ColorGreen)
		y++
		printText(x, y, w, fmt.Sprintf("Tx %spps", numberToUnit(nsDiff.txPkts)), tm.ColorWhite, tm.ColorBlack)
		printUsageBar(x+14, y, 10, nsDiff.txPkts, 10, tm.ColorWhite)
		y++
		printText(x, y, w, fmt.Sprintf("Rx %spps", numberToUnit(nsDiff.rxPkts)), tm.ColorWhite, tm.ColorBlack)
		printUsageBar(x+14, y, 10, nsDiff.rxPkts, 10, tm.ColorCyan)
		y++
		printText(x, y, w, "-------------------------", tm.ColorDefault, tm.ColorDefault)
		y++
	}
	printText(x, y, w,
		fmt.Sprintf("Tcp Retrans: %s",
			numberToUnit((gCurNetStats.tcpStats.segRetrans-gPrevNetStats.tcpStats.segRetrans)/seconds)),
		tm.ColorDefault, tm.ColorDefault)
}
