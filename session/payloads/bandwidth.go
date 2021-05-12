package payloads

import (
	"fmt"

	"weavelab.xyz/ethr/ui"
)

type RawBandwidthPayload struct {
	ConnectionID     string
	Bandwidth        uint64
	PacketsPerSecond uint64
}

func (p RawBandwidthPayload) String() string {
	return fmt.Sprintf("id: %s, bandwidth: %s pkt/s: %s", p.ConnectionID, ui.BytesToRate(p.Bandwidth), ui.PpsToString(p.PacketsPerSecond))
}

type BandwidthPayload struct {
	TotalBandwidth        uint64
	TotalPacketsPerSecond uint64
	ConnectionBandwidths  []RawBandwidthPayload
}

func (p BandwidthPayload) String() string {
	return fmt.Sprintf("connections: %d, bandwidth: %s pkt/s: %s", len(p.ConnectionBandwidths), ui.BytesToRate(p.TotalBandwidth), ui.PpsToString(p.TotalPacketsPerSecond))
}
