package client

import (
	"context"
	"fmt"
	"time"

	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
)

func (u *UI) PrintTestResults(ctx context.Context, test *session.Test) {
	started := time.Now()
	exiting := false
	displayedHeader := false
	var previousResult, latestResult *session.TestResult

	tickInterval := 250 * time.Millisecond
	if test.ID.Type == ethr.TestTypeBandwidth || test.ID.Type == ethr.TestTypePacketsPerSecond || test.ID.Type == ethr.TestTypeConnectionsPerSecond {
		tickInterval = time.Second
	}
	paintTicker := time.NewTicker(tickInterval)

	for {
		// TODO probably want this a little more accurate than rounded seconds, some things it makes sense to print more than once a second
		u.currentPrintSeconds = uint64(time.Since(started).Seconds())
		switch test.ID.Type {
		case ethr.TestTypePing:
			if latestResult != previousResult && latestResult != nil {
				u.PrintPing(test, latestResult)
			}
		case ethr.TestTypePacketsPerSecond:
			if !displayedHeader {
				u.PrintPacketsPerSecondHeader()
				displayedHeader = true
			}
			if latestResult != previousResult && latestResult != nil {
				u.PrintPacketsPerSecond(test, latestResult)
			}
		case ethr.TestTypeBandwidth:
			if !displayedHeader {
				u.PrintBandwidthHeader(test.ID.Protocol)
				displayedHeader = true
			}
			if latestResult != previousResult && latestResult != nil {
				u.PrintBandwidth(test, latestResult)
			}
		case ethr.TestTypeLatency:
			if !displayedHeader {
				u.PrintLatencyHeader()
				displayedHeader = true
			}
			if latestResult != previousResult && latestResult != nil {
				u.PrintLatency(test, latestResult)
			}
		case ethr.TestTypeConnectionsPerSecond:
			if !displayedHeader {
				u.PrintConnectionsHeader()
				displayedHeader = true
			}
			if latestResult != previousResult && latestResult != nil {
				u.PrintConnectionsPerSecond(test, latestResult)
			}
		case ethr.TestTypeTraceRoute:
			fallthrough
		case ethr.TestTypeMyTraceRoute:
			if !displayedHeader {
				u.PrintTracerouteHeader(test.RemoteIP)
				displayedHeader = true
			}
			// if we are exiting drain the results to make sure everything gets printed
			if exiting {
				for r := range test.Results {
					u.PrintTraceroute(test, &r)
				}
				return
			}

			select {
			case r := <-test.Results:
				u.PrintTraceroute(test, &r)
			default:
			}
		default:
			u.printUnknownResultType()
		}
		// TODO probably want this a little more accurate, some things it makes sense to print more than once a second
		u.lastPrintSeconds = uint64(time.Since(started).Seconds())

		select {
		case <-paintTicker.C:
			// TODO convert each test type to read from results chan
			previousResult = latestResult
			latestResult = test.LatestResult()
			continue
		case <-test.Done:
			// Ensure one last paint
			if exiting {
				return
			}
			exiting = true
		case <-ctx.Done():
			return
		}
	}
}

func (u *UI) printUnknownResultType() {
	fmt.Printf("Unknown test result...\n")
}
