package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintTraceroute(test *session.Test, result session.TestResult, showHeader bool) {
	if showHeader {
		u.printTracerouteDivider()
		u.printTracerouteHeader(test.RemoteIP)
	}

	if result.Error != nil {
		fmt.Printf("error running traceroute: %v\n", result.Error)
		return
	}
	switch r := result.Body.(type) {
	case payloads.TraceRoutePayload:
		fmt.Println(r.String())
	default:
		if r != nil {
			u.printUnknownResultType()
		}
	}
}

func (u *UI) printTracerouteHeader(host net.IP) {
	fmt.Printf("Host: %-40s    Sent    Recv        Last         Avg        Best        Worst\n", host.String())
}

func (u *UI) printTracerouteDivider() {
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
}
