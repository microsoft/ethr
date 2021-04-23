package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintPing(test *session.Test, result session.TestResult, showHeader bool) {
	switch r := result.Body.(type) {
	case payloads.PingPayload:
		u.printPingDivider()
		u.printPingHeader(test.RemoteIP)
		u.printPingResult(r.Sent, r.Lost, r.Received)
		if r.Received > 0 {
			u.printLatencyDivider()
			u.printLatencyHeader()
			fmt.Printf("%s\n", r.Latency)
		}
	default:
		if r != nil {
			u.printUnknownResultType()
		}
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
