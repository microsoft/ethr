package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintTraceroute(test *session.Test, result *session.TestResult) {
	switch r := result.Body.(type) {
	case payloads.NetworkHop:
		fmt.Println(r)
	case payloads.TraceRoutePayload:
		if test.ID.Type == ethr.TestTypeMyTraceRoute && result.Success {
			u.PrintTracerouteHeader(test.RemoteIP)
			fmt.Println(r)
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

func (u *UI) PrintTracerouteHeader(host net.IP) {
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
	fmt.Printf("Host: %-70s\t%-5s\t%-5s\t%-9s\t%-9s\t%-9s\t%-9s\n", host.String(), "Sent", "Recv", "Last", "Avg", "Best", "Worst")
}
