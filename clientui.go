//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type clientUI struct {
}

func (u *clientUI) fini() {
}

func (u *clientUI) printMsg(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	logMsg(s)
	fmt.Println(s)
}

func (u *clientUI) printErr(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	logErr(s)
	fmt.Println(s)
}

func (u *clientUI) printDbg(format string, a ...interface{}) {
	if logDebug {
		s := fmt.Sprintf(format, a...)
		logDbg(s)
		fmt.Println(s)
	}
}

func (u *clientUI) paint() {
}

func (u *clientUI) emitTestResultBegin() {
}

/*
func (u *clientUI) emitTestResult(s []string) {
	fmt.Printf("%-15s %-5s %7s %7s %7s\n", s[0], s[1], s[2], s[3], s[4])
}
*/

func (u *clientUI) emitTestHdr() {
	s := []string{"ServerAddress", "Proto", "Bits/s", "Conn/s", "Pkt/s"}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("%-15s %-5s %7s %7s %7s\n", s[0], s[1], s[2], s[3], s[4])
}

func (u *clientUI) emitLatencyHdr() {
	s := []string{"Avg", "Min", "50%", "90%", "95%", "99%", "99.9%", "99.99%", "Max"}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("%8s %8s %8s %8s %8s %8s %8s %8s %8s\n", s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7], s[8])
}

func (u *clientUI) emitLatencyResults(remote, proto string, avg, min, max, p50, p90, p95, p99, p999, p9999 time.Duration) {
	logLatency(remote, proto, avg, min, max, p50, p90, p95, p99, p999, p9999)
	fmt.Printf("%8s %8s %8s %8s %8s %8s %8s %8s %8s\n",
		durationToString(avg), durationToString(min),
		durationToString(p50), durationToString(p90),
		durationToString(p95), durationToString(p99),
		durationToString(p999), durationToString(p9999),
		durationToString(max))
}

func (u *clientUI) emitTestResultEnd() {
}

func (u *clientUI) emitStats(netStats ethrNetStat) {
}

func (u *clientUI) printTestResults(s []string) {
}

func initClientUI() {
	cli := &clientUI{}
	ui = cli
}

var gInterval uint64

func printTestResult(test *ethrTest, value uint64) {
	if test.testParam.TestID.Type == Bandwidth && (test.testParam.TestID.Protocol == TCP ||
		test.testParam.TestID.Protocol == UDP) {
		if gInterval == 0 {
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
			ui.printMsg("[ ID]   Protocol    Interval      Bits/s")
		}
		cvalue := uint64(0)
		ccount := 0
		test.connListDo(func(ec *ethrConn) {
			value = atomic.SwapUint64(&ec.data, 0)
			ui.printMsg("[%3d]     %-5s    %03d-%03d sec   %7s", ec.fd,
				protoToString(test.testParam.TestID.Protocol),
				gInterval, gInterval+1, bytesToRate(value))
			cvalue += value
			ccount++
		})
		if ccount > 1 {
			ui.printMsg("[SUM]     %-5s    %03d-%03d sec   %7s",
				protoToString(test.testParam.TestID.Protocol),
				gInterval, gInterval+1, bytesToRate(cvalue))
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
		}
		logResults([]string{test.session.remoteAddr, protoToString(test.testParam.TestID.Protocol),
			bytesToRate(cvalue), "", "", ""})
	} else if test.testParam.TestID.Type == Cps {
		if gInterval == 0 {
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
			ui.printMsg("Protocol    Interval      Conn/s")
		}
		ui.printMsg("  %-5s    %03d-%03d sec   %7s",
			protoToString(test.testParam.TestID.Protocol),
			gInterval, gInterval+1, cpsToString(value))
		logResults([]string{test.session.remoteAddr, protoToString(test.testParam.TestID.Protocol),
			"", cpsToString(value), "", ""})
	} else if test.testParam.TestID.Type == Pps {
		if gInterval == 0 {
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
			ui.printMsg("Protocol    Interval      Pkts/s")
		}
		ui.printMsg("  %-5s    %03d-%03d sec   %7s",
			protoToString(test.testParam.TestID.Protocol),
			gInterval, gInterval+1, ppsToString(value))
		logResults([]string{test.session.remoteAddr, protoToString(test.testParam.TestID.Protocol),
			"", "", ppsToString(value), ""})
	} else if test.testParam.TestID.Type == Bandwidth && test.testParam.TestID.Protocol == HTTP {
		if gInterval == 0 {
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
			ui.printMsg("Protocol    Interval      Bits/s")
		}
		ui.printMsg("  %-5s    %03d-%03d sec   %7s",
			protoToString(test.testParam.TestID.Protocol),
			gInterval, gInterval+1, bytesToRate(value))
		logResults([]string{test.session.remoteAddr, protoToString(test.testParam.TestID.Protocol),
			bytesToRate(value), "", "", ""})
	}
	gInterval++
}

func (u *clientUI) emitTestResult(s *ethrSession, proto EthrProtocol) {
	var data uint64
	var testList = []EthrTestType{Bandwidth, Cps, Pps}

	for _, testType := range testList {
		test, found := s.tests[EthrTestID{proto, testType}]
		if found && test.isActive {
			data = atomic.SwapUint64(&test.testResult.data, 0)
			printTestResult(test, data)
		}
	}
}
