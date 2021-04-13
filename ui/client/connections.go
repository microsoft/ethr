package client

import (
	"fmt"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintConnectionsPerSecond(test *session.Test, result client.TestResult, showHeader bool, printCount uint64) {
	// TODO make results self contained
	if showHeader {
		u.printConnectionsDivider()
		u.printConnectionsHeader()
	}
	switch r := result.Body.(type) {
	case payloads.ConnectionsPerSecondPayload:
		u.printConnectionsResult(test.ID.Protocol, printCount, r.Connections)
		//logResults([]string{test.session.remoteIP, protoToString(test.testID.Protocol),
		//	"", cpsToString(cps), "", ""})
	default:
		u.printUnknownResultType()

	}
}

func (u *UI) printConnectionsHeader() {
	fmt.Printf("Protocol    Interval      Conn/s\n")
}

func (u *UI) printConnectionsDivider() {
	fmt.Printf("- - - - - - - - - - - - - - - - - - ")
}

func (u *UI) printConnectionsResult(protocol ethr.Protocol, printCount uint64, cps uint64) {
	fmt.Printf("  %-5s    %03d-%03d sec   %7s\n", ethr.ProtocolToString(protocol), printCount, printCount+1, ui.CpsToString(cps))
}