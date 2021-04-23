package client

import (
	"fmt"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/session"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintBandwidth(test *session.Test, result session.TestResult, showHeader bool, printCount uint64) {
	protocol := test.ID.Protocol
	if protocol != ethr.TCP && protocol != ethr.UDP {
		fmt.Printf("Unsupported protocol for bandwidth test: %s\n", protocol.String())
		return
	}
	if showHeader {
		u.printConnectionsDivider()
		u.printConnectionsHeader()
	}
	if result.Body == nil {
		return
	}
	switch r := result.Body.(type) {
	case payloads.BandwidthPayload:
		if u.ShowConnectionStats {
			for _, conn := range r.ConnectionBandwidths {
				u.printBandwidthResult(protocol, conn.ConnectionID, printCount, printCount+1, conn.Bandwidth, conn.PacketsPerSecond)
			}
		}
		u.printBandwidthResult(protocol, "SUM", printCount, printCount+1, r.TotalBandwidth, r.TotalPacketsPerSecond)
		u.Logger.TestResult(ethr.TestTypeBandwidth, true, protocol, test.RemoteIP, test.RemotePort, r)
	default:
		if r != nil {
			u.printUnknownResultType()
		}
	}
}

func (u *UI) printBandwidthHeader(p ethr.Protocol) {
	// Printing packets only makes sense for UDP as it is a datagram protocol.
	// For TCP, TCP itself decides how to chunk the stream to send as packets.
	if p == ethr.UDP {
		fmt.Printf("%10s %12s %14s %10s %10s\n", "[  ID  ]", "Protocol", "Interval", "Bits/s", "Pkts/s")
	} else {
		fmt.Printf("%10s %12s %14s %10s\n", "[  ID  ]", "Protocol", "Interval", "Bits/s")
	}
}

func (u *UI) printBandwidthResult(p ethr.Protocol, id string, t0, t1, bw, pps uint64) {
	if p == ethr.UDP {
		fmt.Printf("[%5s]     %-5s    %03d-%03d sec   %7s   %7s\n", id, p, t0, t1, ui.BytesToRate(bw), ui.PpsToString(pps))
	} else {
		fmt.Printf("[%5s]     %-5s    %03d-%03d sec   %7s\n", id, p, t0, t1, ui.BytesToRate(bw))
	}
}

func (u *UI) printBandwidthDivider(p ethr.Protocol) {
	if p == ethr.UDP {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - -")
	} else {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - -")
	}
}
