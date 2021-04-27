package stats

import (
	"sort"
	"time"

	"weavelab.xyz/ethr/ethr"
)

var Logger ethr.Logger

type NetStat struct {
	Devices []DeviceStats
	TCP     TCPStat
}

type DeviceStats struct {
	InterfaceName string
	RXBytes       uint64
	TXBytes       uint64
	RXPackets     uint64
	TXPackets     uint64
}

type TCPStat struct {
	RetransmittedSegments uint64
}

func GetNetStats() NetStat {
	stats := &NetStat{}
	getNetDevStats(stats)

	sort.SliceStable(stats.Devices, func(i, j int) bool {
		return stats.Devices[i].InterfaceName < stats.Devices[j].InterfaceName
	})

	getTCPStats(stats)

	return *stats
}

func DiffNetDevStats(curStats DeviceStats, prevNetStats NetStat, nanos uint64) DeviceStats {
	for _, prevStats := range prevNetStats.Devices {
		if prevStats.InterfaceName != curStats.InterfaceName {
			continue
		}

		if curStats.RXBytes >= prevStats.RXBytes {
			curStats.RXBytes -= prevStats.RXBytes
		} else {
			curStats.RXBytes += (^uint64(0) - prevStats.RXBytes)
		}

		if curStats.TXBytes >= prevStats.TXBytes {
			curStats.TXBytes -= prevStats.TXBytes
		} else {
			curStats.TXBytes += (^uint64(0) - prevStats.TXBytes)
		}

		if curStats.RXPackets >= prevStats.RXPackets {
			curStats.RXPackets -= prevStats.RXPackets
		} else {
			curStats.RXPackets += (^uint64(0) - prevStats.RXPackets)
		}

		if curStats.TXPackets >= prevStats.TXPackets {
			curStats.TXPackets -= prevStats.TXPackets
		} else {
			curStats.TXPackets += (^uint64(0) - prevStats.TXPackets)
		}

		break
	}
	if nanos < 1 {
		nanos = 1
	}
	curStats.RXBytes = 1e9 * curStats.RXBytes / nanos
	curStats.TXBytes = 1e9 * curStats.TXBytes / nanos
	curStats.RXPackets = 1e9 * curStats.RXPackets / nanos
	curStats.TXPackets = 1e9 * curStats.TXPackets / nanos
	return curStats
}

var StatsEnabled bool

func StartTimer() {
	if StatsEnabled {
		return
	}

	// In an ideal setup, client and server should print stats at the same time.
	// However, instead of building a whole time synchronization mechanism, a
	// hack is used that starts stat at a second granularity. This is done on
	// both client and sever, and as long as both client & server have time
	// synchronized e.g. with a time server, both would print stats of the running
	// test at _almost_ the same time.
	sleepUntilNextWholeSecond()

	lastStatsTime = time.Now()
	ticker := time.NewTicker(time.Second)
	StatsEnabled = true
	go func() {
		for StatsEnabled {
			select {
			case <-ticker.C:
				sampleStats()
			}
		}
		ticker.Stop()
		return
	}()
}

func sleepUntilNextWholeSecond() {
	t0 := time.Now()
	t1 := t0.Add(time.Second)
	res := t1.Round(time.Second)
	time.Sleep(time.Until(res))
}

func StopTimer() {
	StatsEnabled = false
}

var lastStatsTime = time.Now()

func LatestStats() NetStat {
	return latestStats
}

func PreviousStats() NetStat {
	return historicalStats
}

var latestStats NetStat
var historicalStats NetStat

func sampleStats() {
	lastStatsTime = time.Now()
	historicalStats = latestStats
	latestStats = GetNetStats()
}
