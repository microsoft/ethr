package payloads

import (
	"time"

	"weavelab.xyz/ethr/ethr"
)

type LatencyPayload struct {
	RemoteIP string
	Protocol ethr.Protocol
	Raw      []time.Duration
	Avg      time.Duration
	Min      time.Duration
	Max      time.Duration
	P50      time.Duration
	P90      time.Duration
	P95      time.Duration
	P99      time.Duration
	P999     time.Duration
	P9999    time.Duration
}
