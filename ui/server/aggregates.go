package server

import (
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/ui"
)

type AggregateStats struct {
	Stats  *StatsGroup
	Counts *StatsGroup // TODO do we actually need counts?
}

type StatsGroup struct {
	Bandwidth            uint64
	ConnectionsPerSecond uint64
	PacketsPerSecond     uint64
}

func NewAggregateStats() *AggregateStats {
	return &AggregateStats{
		Stats: &StatsGroup{
			Bandwidth:            0,
			ConnectionsPerSecond: 0,
			PacketsPerSecond:     0,
		},
		Counts: &StatsGroup{
			Bandwidth:            0,
			ConnectionsPerSecond: 0,
			PacketsPerSecond:     0,
		},
	}
}

func (a *AggregateStats) ToString(protocol ethr.Protocol) (out []string) {
	if a == nil {
		return
	}
	//if a.Counts.Bandwidth > 1 || a.Counts.ConnectionsPerSecond > 1 || a.Counts.PacketsPerSecond > 1 {
	out = []string{"[SUM]", protocol.String(),
		ui.BytesToRate(a.Stats.Bandwidth),
		ui.CpsToString(a.Stats.ConnectionsPerSecond),
		ui.PpsToString(a.Stats.PacketsPerSecond),
		""}
	//}
	return
}

func (a *AggregateStats) Reset() {
	if a == nil {
		return
	}
	a.Stats.Bandwidth = 0
	a.Stats.ConnectionsPerSecond = 0
	a.Stats.PacketsPerSecond = 0

	a.Counts.Bandwidth = 0
	a.Counts.ConnectionsPerSecond = 0
	a.Counts.PacketsPerSecond = 0
}
