package client

import (
	"fmt"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ui"
)

func (u *UI) PrintLatency(test *session.Test, result session.TestResult, showHeader bool) {
	if showHeader {
		u.printLatencyDivider()
		u.printLatencyHeader()
	}

	switch r := result.Body.(type) {
	case payloads.LatencyPayload:
		u.printLatencyResult(r)
	default:
		u.printUnknownResultType()
	}

}

func (u *UI) printLatencyDivider() {
	fmt.Println("-----------------------------------------------------------------------------------------")

}
func (u *UI) printLatencyHeader() {
	s := []string{"Avg", "Min", "50%", "90%", "95%", "99%", "99.9%", "99.99%", "Max"}
	fmt.Printf("%9s %9s %9s %9s %9s %9s %9s %9s %9s\n", s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7], s[8])
}

func (u *UI) printLatencyResult(l payloads.LatencyPayload) {
	//logLatency(remote, proto, avg, min, max, p50, p90, p95, p99, p999, p9999)
	fmt.Printf("%9s %9s %9s %9s %9s %9s %9s %9s %9s\n",
		ui.DurationToString(l.Avg),
		ui.DurationToString(l.Min),
		ui.DurationToString(l.P50),
		ui.DurationToString(l.P90),
		ui.DurationToString(l.P95),
		ui.DurationToString(l.P99),
		ui.DurationToString(l.P999),
		ui.DurationToString(l.P9999),
		ui.DurationToString(l.Max))
}
