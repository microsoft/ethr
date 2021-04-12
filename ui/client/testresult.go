package client

import (
	"context"
	"fmt"
	"time"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/session"
)

func (u *UI) PrintTestResults(ctx context.Context, test *session.Test, results chan client.TestResult, seconds uint64) {
	printCount := uint64(0)
	var latestResult client.TestResult

	paintTicker := time.NewTicker(200 * time.Millisecond)
	for {
		switch test.ID.Type {
		case session.TestTypePing:
			u.PrintPing(test, latestResult, printCount == 0)
		case session.TestTypePacketsPerSecond:
			u.PrintPacketsPerSecond(test, latestResult, printCount == 0, printCount)
		case session.TestTypeBandwidth:
			u.PrintBandwidth(test, printCount == 0, seconds, printCount)
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
		case latestResult = <-results:
			continue
		case <-paintTicker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (u *UI) printUnknownResultType() {
	fmt.Printf("Unknown test result...")
}
