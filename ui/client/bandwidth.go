package client

import (
	"fmt"
	"sync/atomic"

	"weavelab.xyz/ethr/session"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/ui"
)

// TODO use results object rather than test.Conn
func (u *UI) PrintBandwidth(test *session.Test, showHeader bool, seconds uint64, printCount uint64) {
	protocol := test.ID.Protocol
	if protocol != ethr.TCP && protocol != ethr.UDP {
		fmt.Printf("Unsupported protocol for bandwidth test: %s\n", ethr.ProtocolToString(test.ID.Protocol))
		return
	}
	if showHeader {
		u.printBandwidthDivider(protocol)
		u.printBandwidthHeader(protocol)
	}

	// TODO make results self contained
	cbw := uint64(0)
	cpps := uint64(0)
	ccount := 0

	test.ConnListDo(func(ec *session.Conn) {
		bw := atomic.SwapUint64(&ec.Bandwidth, 0)
		pps := atomic.SwapUint64(&ec.PacketsPerSecond, 0)
		bw /= seconds
		if u.ShowConnectionStats {
			fd := fmt.Sprintf("%5d", ec.FD)
			u.printBandwidthResult(protocol, fd, printCount, printCount+1, bw, pps)
		}
		cbw += bw
		cpps += pps
		ccount++
	})

	if ccount > 1 || !u.ShowConnectionStats {
		u.printBandwidthResult(protocol, "SUM", printCount, printCount+1, cbw, cpps)
		if u.ShowConnectionStats {
			u.printBandwidthDivider(protocol)
		}
	}

	//logResults([]string{test.RemoteIP.String(), ethr.ProtocolToString(protocol),
	//	ui.BytesToRate(cbw), "", ui.PpsToString(cpps), ""})
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

func (u *UI) printBandwidthResult(p ethr.Protocol, fd string, t0, t1, bw, pps uint64) {
	if p == ethr.UDP {
		fmt.Printf("[%5s]     %-5s    %03d-%03d sec   %7s   %7s", fd, ethr.ProtocolToString(p), t0, t1, ui.BytesToRate(bw), ui.PpsToString(pps))
	} else {
		fmt.Printf("[%5s]     %-5s    %03d-%03d sec   %7s", fd, ethr.ProtocolToString(p), t0, t1, ui.BytesToRate(bw))
	}
}

func (u *UI) printBandwidthDivider(p ethr.Protocol) {
	if p == ethr.UDP {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - -")
	} else {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - -")
	}
}
