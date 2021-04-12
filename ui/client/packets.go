package client

import (
	"fmt"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintPacketsPerSecond(test *session.Test, result client.TestResult, showHeader bool, printCount uint64) {
	if showHeader {
		u.printPacketsDivider()
		u.printPacketsHeader()
	}
	// TODO make results self contained
	//bw := atomic.SwapUint64(&test.testResult.bw, 0)
	//pps := atomic.SwapUint64(&test.testResult.pps, 0)
	switch r := result.Body.(type) {
	case payloads.BandwidthPayload:
		u.printPacketsResult(test.ID.Protocol, r, printCount)
		//logResults([]string{test.session.remoteIP, protoToString(test.testID.Protocol),
		//	bytesToRate(bw), "", ppsToString(pps), ""})
	default:
		u.printUnknownResultType()
	}
}

func (u *UI) printPacketsHeader() {
	fmt.Println("Protocol    Interval      Bits/s    Pkts/s")
}

func (u *UI) printPacketsDivider() {
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - -")
}

func (u *UI) printPacketsResult(protocol ethr.Protocol, body payloads.BandwidthPayload, printCount uint64) {
	fmt.Printf("  %-5s    %03d-%03d sec   %7s   %7s\n", ethr.ProtocolToString(protocol), printCount, printCount+1, ui.BytesToRate(body.TotalBandwidth), ui.PpsToString(body.PacketsPerSecond))
}
