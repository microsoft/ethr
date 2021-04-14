package payloads

import "time"

type RawPingPayload struct {
	Latency time.Duration
	Lost    bool
}

type PingPayload struct {
	Latency  LatencyPayload
	Sent     uint32
	Lost     uint32
	Received uint32
}
