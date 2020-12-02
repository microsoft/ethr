//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	tm "github.com/nsf/termbox-go"
)

type ethrTestResultAggregate struct {
	bw, cps, pps    uint64
	cbw, ccps, cpps uint64
}

var gAggregateTestResults = make(map[EthrProtocol]*ethrTestResultAggregate)

//
// Initialization functions.
//
func initServerUI(showUI bool) {
	gAggregateTestResults[TCP] = &ethrTestResultAggregate{}
	gAggregateTestResults[UDP] = &ethrTestResultAggregate{}
	gAggregateTestResults[ICMP] = &ethrTestResultAggregate{}
	if !showUI || !initServerTui() {
		initServerCli()
	}
}

//
// Text based UI
//
type serverTui struct {
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

func initServerTui() bool {
	err := initServerTuiInternal()
	if err != nil {
		fmt.Println("Error: Failed to initialize UI.", err)
		fmt.Println("Using command line view instead of UI")
		return false
	}
	return true
}

func initServerTuiInternal() error {
	err := tm.Init()
	if err != nil {
		return err
	}

	w, h := tm.Size()
	if h < 40 || w < 80 {
		tm.Close()
		s := fmt.Sprintf("Terminal too small (%dwx%dh), must be at least 40hx80w", w, h)
		return errors.New(s)
	}

	tm.SetInputMode(tm.InputEsc | tm.InputMouse)
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	tm.Sync()
	tm.Flush()
	hideCursor()
	blockWindowResize()

	tui := &serverTui{}
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
	tui.res = table{6, []int{13, 5, 7, 7, 7, 8}, 0, 2, 0, justifyRight, noBorder}
	tui.results = make([][]string, 0)
	tui.msg = table{1, []int{tui.msgW}, tui.msgX, tui.msgY, 0, justifyLeft, noBorder}
	tui.msgRing = make([]string, botScnH-1)
	tui.err = table{1, []int{tui.errW}, tui.errX, tui.errY, 0, justifyLeft, noBorder}
	tui.errRing = make([]string, botScnH-1)
	ui = tui

	go func() {
		for {
			switch ev := tm.PollEvent(); ev.Type {
			case tm.EventKey:
				if ev.Key == tm.KeyEsc || ev.Key == tm.KeyCtrlC {
					finiServer()
					os.Exit(0)
				}
			case tm.EventResize:
			}
		}
	}()

	return nil
}

func (u *serverTui) fini() {
	tm.Close()
}

func (u *serverTui) getTitle() string {
	return ""
}

func (u *serverTui) printMsg(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	logInfo(s)
	ss := splitString(s, u.msgW)
	u.ringLock.Lock()
	u.msgRing = u.msgRing[len(ss):]
	u.msgRing = append(u.msgRing, ss...)
	u.ringLock.Unlock()
}

func (u *serverTui) printErr(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	logError(s)
	ss := splitString(s, u.errW)
	u.ringLock.Lock()
	u.errRing = u.errRing[len(ss):]
	u.errRing = append(u.errRing, ss...)
	u.ringLock.Unlock()
}

func (u *serverTui) printDbg(format string, a ...interface{}) {
	if loggingLevel == LogLevelDebug {
		s := fmt.Sprintf(format, a...)
		logDebug(s)
		ss := splitString(s, u.errW)
		u.ringLock.Lock()
		u.errRing = u.errRing[len(ss):]
		u.errRing = append(u.errRing, ss...)
		u.ringLock.Unlock()
	}
}

func (u *serverTui) emitTestResultBegin() {
	u.results = nil
}

func (u *serverTui) emitTestResult(s *ethrSession, proto EthrProtocol, seconds uint64) {
	str := getTestResults(s, proto, seconds)
	if len(str) > 0 {
		ui.printTestResults(str)
	}
}

func (u *serverTui) printTestResults(s []string) {
	// Log before truncation of remote address.
	logResults(s)
	s[0] = truncateStringFromStart(s[0], 13)
	u.results = append(u.results, s)
}

func (u *serverTui) emitTestResultEnd() {
	emitAggregateResults()
}

func (u *serverTui) emitTestHdr() {
	s := []string{"RemoteAddress", "Proto", "Bits/s", "Conn/s", "Pkts/s", "Latency"}
	u.resultHdr = s
}

func (u *serverTui) emitLatencyHdr() {
}

func (u *serverTui) emitLatencyResults(remote, proto string, avg, min, max, p50, p90, p95, p99, p999, p9999 time.Duration) {
	logLatency(remote, proto, avg, min, max, p50, p90, p95, p99, p999, p9999)
}

func (u *serverTui) paint(seconds uint64) {
	tm.Clear(tm.ColorDefault, tm.ColorDefault)
	defer tm.Flush()
	printCenterText(0, 0, u.w, "Ethr (Version: "+gVersion+")", tm.ColorBlack, tm.ColorWhite)
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

var gPrevNetStats ethrNetStat
var gCurNetStats ethrNetStat

func (u *serverTui) emitStats(netStats ethrNetStat) {
	gPrevNetStats = gCurNetStats
	gCurNetStats = netStats
}

//
// Simple command window based output
//
type serverCli struct {
}

func initServerCli() {
	cli := &serverCli{}
	ui = cli
}

func (u *serverCli) fini() {
}

func (u *serverCli) getTitle() string {
	return ""
}

func (u *serverCli) printMsg(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Println(s)
	logInfo(s)
}

func (u *serverCli) printDbg(format string, a ...interface{}) {
	if loggingLevel != LogLevelDebug {
		return
	}
	s := fmt.Sprintf(format, a...)
	fmt.Println(s)
	logDebug(s)
}

func (u *serverCli) printErr(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Println(s)
	logError(s)
}

func (u *serverCli) paint(seconds uint64) {
}

func (u *serverCli) emitTestResultBegin() {
	gSessionLock.RLock()
	l := len(gSessionKeys)
	gSessionLock.RUnlock()
	if l > 1 {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
	}
}

func (u *serverCli) emitTestResult(s *ethrSession, proto EthrProtocol, seconds uint64) {
	str := getTestResults(s, proto, seconds)
	if len(str) > 0 {
		ui.printTestResults(str)
	}
}

func (u *serverCli) emitTestResultEnd() {
	emitAggregateResults()
}

func (u *serverCli) emitTestHdr() {
	s := []string{"RemoteAddress", "Proto", "Bits/s", "Conn/s", "Pkt/s", "Latency"}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("[%13s]  %5s  %7s  %7s  %7s  %8s\n", s[0], s[1], s[2], s[3], s[4], s[5])
}

func (u *serverCli) emitLatencyHdr() {
}

func (u *serverCli) emitLatencyResults(remote, proto string, avg, min, max, p50, p90, p95, p99, p999, p9999 time.Duration) {
	logLatency(remote, proto, avg, min, max, p50, p90, p95, p99, p999, p9999)
}

func (u *serverCli) emitStats(netStats ethrNetStat) {
}

func (u *serverCli) printTestResults(s []string) {
	logResults(s)
	fmt.Printf("[%13s]  %5s  %7s  %7s  %7s  %8s\n", truncateStringFromStart(s[0], 13),
		s[1], s[2], s[3], s[4], s[5])
}

func emitAggregateResults() {
	var protoList = []EthrProtocol{TCP, UDP, ICMP}
	for _, proto := range protoList {
		emitAggregate(proto)
	}
}

func emitAggregate(proto EthrProtocol) {
	str := []string{}
	aggTestResult, _ := gAggregateTestResults[proto]
	if aggTestResult.cbw > 1 || aggTestResult.ccps > 1 || aggTestResult.cpps > 1 {
		str = []string{"[SUM]", protoToString(proto),
			bytesToRate(aggTestResult.bw),
			cpsToString(aggTestResult.cps),
			ppsToString(aggTestResult.pps),
			""}
	}
	aggTestResult.bw = 0
	aggTestResult.cps = 0
	aggTestResult.pps = 0
	aggTestResult.cbw = 0
	aggTestResult.ccps = 0
	aggTestResult.cpps = 0
	if len(str) > 0 {
		ui.printTestResults(str)
	}
}

func getTestResults(s *ethrSession, proto EthrProtocol, seconds uint64) []string {
	var bwTestOn, cpsTestOn, ppsTestOn, latTestOn bool
	var bw, cps, pps, latency uint64
	aggTestResult, _ := gAggregateTestResults[proto]
	test, found := s.tests[EthrTestID{proto, All}]
	if found && test.isActive {
		bwTestOn = true
		bw = atomic.SwapUint64(&test.testResult.bw, 0)
		bw /= seconds
		aggTestResult.bw += bw
		aggTestResult.cbw++

		if proto == TCP {
			cpsTestOn = true
			cps = atomic.SwapUint64(&test.testResult.cps, 0)
			cps /= seconds
			aggTestResult.cps += cps
			aggTestResult.ccps++
		}

		if proto == UDP {
			ppsTestOn = true
			pps = atomic.SwapUint64(&test.testResult.pps, 0)
			pps /= seconds
			aggTestResult.pps += pps
			aggTestResult.cpps++
		}

		if proto == TCP {
			latency = atomic.LoadUint64(&test.testResult.latency)
			if latency > 0 {
				latTestOn = true
			}
		}

		if test.isDormant && !((bwTestOn && bw != 0) || (cpsTestOn && cps != 0) || (ppsTestOn && pps != 0) || (latTestOn && latency != 0)) {
			return []string{}
		}
	}

	if bwTestOn || cpsTestOn || ppsTestOn || latTestOn {
		var bwStr, cpsStr, ppsStr, latStr string = "--  ", "--  ", "--  ", "--  "
		if bwTestOn {
			bwStr = bytesToRate(bw)
		}
		if cpsTestOn {
			cpsStr = cpsToString(cps)
		}
		if ppsTestOn {
			ppsStr = ppsToString(pps)
		}
		if latTestOn {
			latStr = durationToString(time.Duration(latency))
		}
		str := []string{s.remoteIP, protoToString(proto),
			bwStr, cpsStr, ppsStr, latStr}
		return str
	}

	return []string{}
}
