package client

import (
	"fmt"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintConnectionsPerSecond(test *session.Test, result *session.TestResult) {
	switch r := result.Body.(type) {
	case payloads.ConnectionsPerSecondPayload:
		u.printConnectionsResult(test.ID.Protocol, r.Connections)
		u.Logger.TestResult(ethr.TestTypeConnectionsPerSecond, result.Success, test.ID.Protocol, test.RemoteIP, test.RemotePort, r)
	default:
		if r != nil {
			u.printUnknownResultType()
		}

	}
}

func (u *UI) PrintConnectionsHeader() {
	fmt.Println("- - - - - - - - - - - - - - - - - - ")
	fmt.Printf("Protocol    Interval      Conn/s\n")
}

func (u *UI) printConnectionsResult(protocol ethr.Protocol, cps uint64) {
	fmt.Printf("  %-5s    %03d-%03d sec   %7s\n", protocol.String(), u.lastPrintSeconds, u.currentPrintSeconds, ui.CpsToString(cps))
}
