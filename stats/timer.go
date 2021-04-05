package stats

import (
	"sort"
	"time"
	"weavelab.xyz/ethr/ethr"
)

var Logger ethr.Logger

type NetStat struct {
	netDevStats []NetDevStat
	tcpStats    TCPStat
}

type NetDevStat struct {
	interfaceName string
	rxBytes       uint64
	txBytes       uint64
	rxPkts        uint64
	txPkts        uint64
}

type TCPStat struct {
	segRetrans uint64
}

func GetNetStats() NetStat {
	stats := &NetStat{}
	getNetDevStats(stats)
	/*
		devStats, err := osStats.GetNetDevStats()
		if err != nil {
			return stats.EthrNetStats{}, errors.Wrap(err, "getNetworkStats: could not get net device stats")
		}
	*/
	sort.SliceStable(stats.netDevStats, func(i, j int) bool {
		return stats.netDevStats[i].interfaceName < stats.netDevStats[j].interfaceName
	})
	getTCPStats(stats)

	/*
		tcpStats, err := osStats.GetTCPStats()
		if err != nil {
			return stats.EthrNetStats{}, errors.Wrap(err, "getNetworkStats: could not get net TCP stats")
		}

		return stats.EthrNetStats{NetDevStats: devStats, TCPStats: tcpStats}, nil
	*/
	return *stats
}

func DiffNetDevStats(curStats NetDevStat, prevNetStats NetStat, seconds uint64) NetDevStat {
	for _, prevStats := range prevNetStats.netDevStats {
		if prevStats.interfaceName != curStats.interfaceName {
			continue
		}

		if curStats.rxBytes >= prevStats.rxBytes {
			curStats.rxBytes -= prevStats.rxBytes
		} else {
			curStats.rxBytes += (^uint64(0) - prevStats.rxBytes)
		}

		if curStats.txBytes >= prevStats.txBytes {
			curStats.txBytes -= prevStats.txBytes
		} else {
			curStats.txBytes += (^uint64(0) - prevStats.txBytes)
		}

		if curStats.rxPkts >= prevStats.rxPkts {
			curStats.rxPkts -= prevStats.rxPkts
		} else {
			curStats.rxPkts += (^uint64(0) - prevStats.rxPkts)
		}

		if curStats.txPkts >= prevStats.txPkts {
			curStats.txPkts -= prevStats.txPkts
		} else {
			curStats.txPkts += (^uint64(0) - prevStats.txPkts)
		}

		break
	}
	curStats.rxBytes /= seconds
	curStats.txBytes /= seconds
	curStats.rxPkts /= seconds
	curStats.txPkts /= seconds
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

var lastStatsTime time.Time = time.Now()

//func timeToNextTick() time.Duration {
//	nextTick := lastStatsTime.Add(time.Second)
//	return time.Until(nextTick)
//}

func Latest() NetStat {
	return latestStats
}

var latestStats = NetStat{}
func sampleStats() {
	d := time.Since(lastStatsTime)
	lastStatsTime = time.Now()
	seconds := int64(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	latestStats = GetNetStats()
	// Handle UI output externally
	//ui.emitTestResultBegin()
	//emitTestResults(uint64(seconds))
	//ui.emitTestResultEnd()
	//ui.emitStats(getNetworkStats())
	//ui.paint(uint64(seconds))
}

// Get stats from chan/return and print externally
//func emitTestResults(s uint64) {
//	gSessionLock.RLock()
//	defer gSessionLock.RUnlock()
//	for _, k := range gSessionKeys {
//		v := gSessions[k]
//		ui.emitTestResult(v, TCP, s)
//		ui.emitTestResult(v, UDP, s)
//		ui.emitTestResult(v, ICMP, s)
//	}
//}