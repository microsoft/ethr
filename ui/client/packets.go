package client

import (
	"fmt"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintPacketsPerSecond(test *session.Test, result *session.TestResult) {
	switch r := result.Body.(type) {
	case payloads.BandwidthPayload:
		u.printPacketsResult(test.ID.Protocol, r)
		u.Logger.TestResult(ethr.TestTypePacketsPerSecond, result.Success, test.ID.Protocol, test.RemoteIP, test.RemotePort, r)
	default:
		u.printUnknownResultType()
	}
}

func (u *UI) PrintPacketsPerSecondHeader() {
	fmt.Println("Protocol    Interval      Bits/s    Pkts/s")
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - -")

}

func (u *UI) printPacketsResult(protocol ethr.Protocol, body payloads.BandwidthPayload) {
	fmt.Printf("  %-5s    %03d-%03d sec   %7s   %7s\n", protocol.String(), u.lastPrintSeconds, u.currentPrintSeconds, ui.BytesToRate(body.TotalBandwidth), ui.PpsToString(body.TotalPacketsPerSecond))
}
