package payloads

import (
	"fmt"
	"time"

	"weavelab.xyz/ethr/ui"
)

type RawPingPayload struct {
	Latency time.Duration
	Lost    bool
}

func (p RawPingPayload) String() string {
	return fmt.Sprintf("latency: %s lost: %t", ui.DurationToString(p.Latency), p.Lost)
}

type PingPayload struct {
	Latency  LatencyPayload
	Sent     uint32
	Lost     uint32
	Received uint32
}

func (p PingPayload) String() string {
	return fmt.Sprintf("sent: %d lost: %d avg latency: %s", p.Sent, p.Lost, ui.DurationToString(p.Latency.Avg))
}
