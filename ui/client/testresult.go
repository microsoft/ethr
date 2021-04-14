package client

import (
	"context"
	"fmt"
	"time"

	"weavelab.xyz/ethr/session"
)

func (u *UI) PrintTestResults(ctx context.Context, test *session.Test) {
	// TODO get rid of printCount nonsense
	printCount := uint64(0)
	var latestResult session.TestResult

	paintTicker := time.NewTicker(time.Second)
	for {
		switch test.ID.Type {
		case session.TestTypePing:
			u.PrintPing(test, latestResult, printCount == 0)
		case session.TestTypePacketsPerSecond:
			u.PrintPacketsPerSecond(test, latestResult, printCount == 0, printCount)
		case session.TestTypeBandwidth:
			u.PrintBandwidth(test, latestResult, printCount == 0, printCount)
		case session.TestTypeLatency:
			u.PrintLatency(test, latestResult, printCount == 0)
		case session.TestTypeConnectionsPerSecond:
			u.PrintConnectionsPerSecond(test, latestResult, printCount == 0, printCount)
		case session.TestTypeTraceRoute:
			fallthrough
		case session.TestTypeMyTraceRoute:
			u.PrintTraceroute(test, latestResult, printCount == 0)
		default:
			u.printUnknownResultType()
		}
		printCount++

		select {
		case <-paintTicker.C:
			latestResult = test.LatestResult()
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (u *UI) printUnknownResultType() {
	fmt.Printf("Unknown test result...")
}
