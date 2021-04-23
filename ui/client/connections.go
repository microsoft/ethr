package client

import (
	"fmt"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintConnectionsPerSecond(test *session.Test, result session.TestResult, showHeader bool, printCount uint64) {
	if showHeader {
		u.printConnectionsDivider()
		u.printConnectionsHeader()
	}
	switch r := result.Body.(type) {
	case payloads.ConnectionsPerSecondPayload:
		u.printConnectionsResult(test.ID.Protocol, printCount, r.Connections)
		u.Logger.TestResult(ethr.TestTypeConnectionsPerSecond, result.Success, test.ID.Protocol, test.RemoteIP, test.RemotePort, r)
	default:
		if r != nil {
			u.printUnknownResultType()
		}

	}
}

func (u *UI) printConnectionsHeader() {
	fmt.Printf("Protocol    Interval      Conn/s\n")
}

func (u *UI) printConnectionsDivider() {
	fmt.Println("- - - - - - - - - - - - - - - - - - ")
}

func (u *UI) printConnectionsResult(protocol ethr.Protocol, printCount uint64, cps uint64) {
	fmt.Printf("  %-5s    %03d-%03d sec   %7s\n", protocol.String(), printCount, printCount+1, ui.CpsToString(cps))
}
