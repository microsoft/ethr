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

func (a *AggregateStats) ToString(protocol ethr.Protocol) []string {
	var str []string
	if a.Counts.Bandwidth > 1 || a.Counts.ConnectionsPerSecond > 1 || a.Counts.PacketsPerSecond > 1 {
		str = []string{"[SUM]", ethr.ProtocolToString(protocol),
			ui.BytesToRate(a.Stats.Bandwidth),
			ui.CpsToString(a.Stats.ConnectionsPerSecond),
			ui.PpsToString(a.Stats.PacketsPerSecond),
			""}
	}
	return str
}

func (a *AggregateStats) Reset() {
	a.Stats.Bandwidth = 0
	a.Stats.ConnectionsPerSecond = 0
	a.Stats.PacketsPerSecond = 0

	a.Counts.Bandwidth = 0
	a.Counts.ConnectionsPerSecond = 0
	a.Counts.PacketsPerSecond = 0
}
