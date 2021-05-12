package server

import (
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/ui"
)

type AggregateStats struct {
	Bandwidth            uint64
	ConnectionsPerSecond uint64
	PacketsPerSecond     uint64
}

func NewAggregateStats() *AggregateStats {
	return &AggregateStats{
		Bandwidth:            0,
		ConnectionsPerSecond: 0,
		PacketsPerSecond:     0,
	}
}

func (a *AggregateStats) ToString(protocol ethr.Protocol) (out []string) {
	if a == nil {
		return
	}
	//if a.Counts.Bandwidth > 1 || a.Counts.ConnectionsPerSecond > 1 || a.Counts.PacketsPerSecond > 1 {
	out = []string{"[SUM]", protocol.String(),
		ui.BytesToRate(a.Bandwidth),
		ui.CpsToString(a.ConnectionsPerSecond),
		ui.PpsToString(a.PacketsPerSecond),
		""}
	//}
	return
}

func (a *AggregateStats) Reset() {
	if a == nil {
		return
	}
	a.Bandwidth = 0
	a.ConnectionsPerSecond = 0
	a.PacketsPerSecond = 0
}
