package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintPing(test *session.Test, result *session.TestResult) {
	switch r := result.Body.(type) {
	case payloads.PingPayload:
		u.PrintPingHeader(test.RemoteIP)
		u.printPingResult(r.Sent, r.Lost, r.Received)
		u.Logger.TestResult(ethr.TestTypePing, result.Success, test.ID.Protocol, test.RemoteIP, test.RemotePort, r)
		if r.Received > 0 {
			u.PrintLatencyHeader()
			fmt.Printf("%s\n", r.Latency)
		}
	default:
		if r != nil {
			u.printUnknownResultType()
		}
	}
}

func (u *UI) PrintPingHeader(host net.IP) {
	fmt.Println("-----------------------------------------------------------------------------------------")
	fmt.Printf("TCP connect statistics for %s:\n", host.String())
}

func (u *UI) printPingResult(sent uint32, lost uint32, received uint32) {
	fmt.Printf("  Sent = %d, Received = %d, Lost = %d\n", sent, received, lost)
}
