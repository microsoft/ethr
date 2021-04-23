package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintTraceroute(test *session.Test, result session.TestResult, showHeader bool) {
	if showHeader {
		u.printTracerouteDivider()
		u.printTracerouteHeader(test.RemoteIP)
	}

	switch r := result.Body.(type) {
	case payloads.NetworkHop:
		fmt.Println(r)
	case payloads.TraceRoutePayload:
		if test.ID.Type == ethr.TestTypeMyTraceRoute && result.Success {
			u.printTracerouteDivider()
			u.printTracerouteHeader(test.RemoteIP)
			fmt.Println(r)
		} else {
			fmt.Println("Traceroute complete")
		}
	default:
		if r != nil {
			u.printUnknownResultType()
		}
	}
	if result.Error != nil {
		fmt.Println(result.Error.Error())
	}
}

func (u *UI) printTracerouteHeader(host net.IP) {
	fmt.Printf("Host: %-70s\t%-5s\t%-5s\t%-9s\t%-9s\t%-9s\t%-9s\n", host.String(), "Sent", "Recv", "Last", "Avg", "Best", "Worst")
}

func (u *UI) printTracerouteDivider() {
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
}
