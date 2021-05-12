package server

import (
	"fmt"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/ui"
)

type RawUI struct {
	tcpStats  *AggregateStats
	udpStats  *AggregateStats
	icmpStats *AggregateStats
}

func InitRawUI(tcp *AggregateStats, udp *AggregateStats, icmp *AggregateStats) (*RawUI, error) {
	return &RawUI{
		tcpStats:  tcp,
		udpStats:  udp,
		icmpStats: icmp,
	}, nil
}

func (u *RawUI) Paint(seconds uint64) {
	sessions := session.GetSessions()
	if len(sessions) > 0 {
		fmt.Println("- - - - - - - - - - - - - - - - - - - - - - - - - - - - - -")
		u.printTestHeader()
	} else {
		return
	}
	for _, s := range sessions {
		tcpResults := u.getTestResults(&s, ethr.TCP, u.tcpStats)
		u.printTestResults(tcpResults)

		udpResults := u.getTestResults(&s, ethr.UDP, u.udpStats)
		u.printTestResults(udpResults)

		icmpResults := u.getTestResults(&s, ethr.ICMP, u.icmpStats)
		u.printTestResults(icmpResults)
	}

	tcpAgg := u.tcpStats.ToString(ethr.TCP)
	u.tcpStats.Reset()
	u.printTestResults(tcpAgg)

	udpAgg := u.udpStats.ToString(ethr.UDP)
	u.udpStats.Reset()
	u.printTestResults(udpAgg)

	icmpAgg := u.icmpStats.ToString(ethr.ICMP)
	u.icmpStats.Reset()
	u.printTestResults(icmpAgg)
}

func (u *RawUI) printTestHeader() {
	header := []string{"RemoteAddress", "Proto", "Bits/s", "Conn/s", "Pkt/s", "Latency"}
	fmt.Println("-----------------------------------------------------------")
	fmt.Printf("[%13s]  %5s  %7s  %7s  %7s  %8s\n", header[0], header[1], header[2], header[3], header[4], header[5])
}

func (u *RawUI) printTestResults(results []string) {
	fmt.Printf("[%13s]  %5s  %7s  %7s  %7s  %8s\n", ui.TruncateStringFromStart(results[0], 13), results[1], results[2], results[3], results[4], results[5])
}

func (u *RawUI) getTestResults(s *session.Session, protocol ethr.Protocol, agg *AggregateStats) []string {
	var bwTestOn, cpsTestOn, ppsTestOn, latTestOn bool
	var bw, cps, pps uint64
	var lat payloads.LatencyPayload
	test, found := s.Tests[ethr.TestID{Protocol: protocol, Type: ethr.TestTypeServer}]
	if found && test.IsActive {
		result := test.LatestResult()
		if body, ok := result.Body.(payloads.ServerPayload); ok {
			bwTestOn = true
			bw = body.Bandwidth
			agg.Bandwidth += body.Bandwidth

			if protocol == ethr.TCP {
				cpsTestOn = true
				cps = body.ConnectionsPerSecond
				agg.ConnectionsPerSecond += body.ConnectionsPerSecond

				if len(body.Latency.Raw) > 0 {
					latTestOn = true
					lat = body.Latency

					// TODO figure out how to log latencies
				}
			}

			if protocol == ethr.UDP {
				ppsTestOn = true
				pps = body.PacketsPerSecond
				agg.PacketsPerSecond += body.PacketsPerSecond
			}

		}

		if test.IsDormant && !bwTestOn && !cpsTestOn && !ppsTestOn && !latTestOn {
			return []string{}
		}
	}

	if bwTestOn || cpsTestOn || ppsTestOn || latTestOn {
		var bwStr, cpsStr, ppsStr, latStr string = "--  ", "--  ", "--  ", "--  "
		if bwTestOn {
			bwStr = ui.BytesToRate(bw)
		}
		if cpsTestOn {
			cpsStr = ui.CpsToString(cps)
		}
		if ppsTestOn {
			ppsStr = ui.PpsToString(pps)
		}
		if latTestOn {
			latStr = ui.DurationToString(lat.Avg)
		}
		return []string{
			ui.TruncateStringFromStart(test.RemoteIP.String(), 13),
			protocol.String(),
			bwStr,
			cpsStr,
			ppsStr,
			latStr,
		}
	}

	return []string{}
}

func (u *RawUI) AddInfoMsg(msg string) {
	// do nothing
}

func (u *RawUI) AddErrorMsg(msg string) {
	// do nothing
}
