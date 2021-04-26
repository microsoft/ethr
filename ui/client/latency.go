package client

import (
	"fmt"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (u *UI) PrintLatency(test *session.Test, result *session.TestResult) {
	switch r := result.Body.(type) {
	case payloads.LatencyPayload:
		fmt.Printf("%s\n", r)
	default:
		if r != nil {
			u.printUnknownResultType()
		}
	}
}

func (u *UI) PrintLatencyHeader() {
	fmt.Println("---------------------------------------------------------------------------------------------------")
	fmt.Printf("%9s %9s %9s %9s %9s %9s %9s %9s %9s %9s\n", "Avg", "Min", "50%", "90%", "95%", "99%", "99.9%", "99.99%", "Max", "Jitter")
}
