package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/session"
)

func (u *UI) PrintPing(test *session.Test, result client.TestResult, showHeader bool) {
	if showHeader {
		u.printPingDivider()
		u.printPingHeader(test.RemoteIP)
	}
	switch r := result.Body.(type) {
	case payloads.PingPayload:
		u.printPingResult(r.Sent, r.Lost, r.Received)
		if r.Received > 0 {
			u.printLatencyHeader()
			u.printLatencyResult(r.Latency)
		}
	default:
		u.printUnknownResultType()
	}
}

func (u *UI) printPingDivider() {
	fmt.Println("-----------------------------------------------------------------------------------------")
}

func (u *UI) printPingHeader(host net.IP) {
	fmt.Printf("TCP connect statistics for %s:\n", host.String())
}

func (u *UI) printPingResult(sent uint32, lost uint32, received uint32) {
	fmt.Printf("  Sent = %d, Received = %d, Lost = %d\n", sent, received, lost)
}
