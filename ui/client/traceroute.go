package client

import (
	"fmt"
	"net"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintTraceroute(test *session.Test, result session.TestResult, showHeader bool) {
	if showHeader {
		u.printTracerouteDivider()
		u.printTracerouteHeader(test.RemoteIP)
	}
	switch r := result.Body.(type) {
	case payloads.TraceRoutePayload:
		for idx, hop := range r.Hops {
			u.printTracerouteHop(idx+1, hop)
		}
	default:
		u.printUnknownResultType()
	}
}

func (u *UI) printTracerouteHeader(host net.IP) {
	fmt.Printf("Host: %-40s    Sent    Recv        Last         Avg        Best        Wrst\n", host.String())
}

func (u *UI) printTracerouteDivider() {
	fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - ")
}

func (u *UI) printTracerouteHop(currentHop int, hop payloads.NetworkHop) {
	format := "%2d.|--%-40s   %5d   %5d   %9s   %9s   %9s   %9s\n"
	if hop.Addr.String() != "" && hop.Sent > 0 {
		fmt.Printf(format,
			currentHop,
			hop.Addr.String(),
			hop.Sent,
			hop.Rcvd,
			ui.DurationToString(hop.Last),
			ui.DurationToString(hop.Average),
			ui.DurationToString(hop.Best),
			ui.DurationToString(hop.Worst))
	} else {
		fmt.Printf(format, currentHop, "???", "-", "-", "-", "-", "-", "-")
	}
}
