package payloads

import "time"

type HopData struct {
	Addr     string
	Sent     uint32
	Rcvd     uint32
	Lost     uint32
	Last     time.Duration
	Best     time.Duration
	Worst    time.Duration
	Total    time.Duration
	Name     string
	FullName string
}

type TraceRoutePayload struct {
	Hops []HopData
}
