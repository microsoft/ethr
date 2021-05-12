package payloads

import (
	"fmt"

	"weavelab.xyz/ethr/ui"
)

type ServerPayload struct {
	PacketsPerSecond     uint64
	ConnectionsPerSecond uint64
	Bandwidth            uint64
	Latency              LatencyPayload
}

func (p ServerPayload) String() string {
	return fmt.Sprintf("bandwidth: %s pkt/s: %s conn/s: %s avg latency: %s", ui.BytesToRate(p.Bandwidth), ui.PpsToString(p.PacketsPerSecond), ui.CpsToString(p.ConnectionsPerSecond), ui.DurationToString(p.Latency.Avg))
}
