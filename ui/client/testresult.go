package client

import (
	"context"
	"fmt"
	"time"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
)

func (u *UI) PrintTestResults(ctx context.Context, test *session.Test) {
	// TODO get rid of printCount nonsense
	printCount := uint64(0)
	var latestResult session.TestResult
	exiting := false
	paintTicker := time.NewTicker(time.Second)
	for {
		switch test.ID.Type {
		case ethr.TestTypePing:
			u.PrintPing(test, latestResult, printCount == 0)
		case ethr.TestTypePacketsPerSecond:
			u.PrintPacketsPerSecond(test, latestResult, printCount == 0, printCount)
		case ethr.TestTypeBandwidth:
			u.PrintBandwidth(test, latestResult, printCount == 0, printCount)
		case ethr.TestTypeLatency:
			u.PrintLatency(test, latestResult, printCount == 0)
		case ethr.TestTypeConnectionsPerSecond:
			u.PrintConnectionsPerSecond(test, latestResult, printCount == 0, printCount)
		case ethr.TestTypeTraceRoute:
			fallthrough
		case ethr.TestTypeMyTraceRoute:
			select {
			case r := <-test.Results:
				u.PrintTraceroute(test, r, false)
			default:
				if printCount == 0 {
					u.PrintTraceroute(test, session.TestResult{}, true)
				}
			}
		default:
			u.printUnknownResultType()
		}
		printCount++

		select {
		case <-paintTicker.C:
			latestResult = test.LatestResult()
			continue
		case <-test.Done:
			// Ensure one last paint
			if exiting {
				return
			}
			latestResult = test.LatestResult()
			exiting = true
		case <-ctx.Done():
			return
		}
	}
}

func (u *UI) printUnknownResultType() {
	fmt.Printf("Unknown test result...\n")
}
