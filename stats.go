//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"sort"
	"time"
)

type ethrNetStat struct {
	netDevStats []ethrNetDevStat
	tcpStats    ethrTCPStat
}

type ethrNetDevStat struct {
	interfaceName string
	rxBytes       uint64
	txBytes       uint64
	rxPkts        uint64
	txPkts        uint64
}

type ethrTCPStat struct {
	segRetrans uint64
}

func getNetworkStats() ethrNetStat {
	stats := &ethrNetStat{}
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

func getNetDevStatDiff(curStats ethrNetDevStat, prevNetStats ethrNetStat, seconds uint64) ethrNetDevStat {
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

var statsEnabled bool

func startStatsTimer() {
	if statsEnabled {
		return
	}

	// In an ideal setup, client and server should print stats at the same time.
	// However, instead of building a whole time synchronization mechanism, a
	// hack is used that starts stat at a second granularity. This is done on
	// both client and sever, and as long as both client & server have time
	// synchronized e.g. with a time server, both would print stats of the running
	// test at _almost_ the same time.
	SleepUntilNextWholeSecond()

	lastStatsTime = time.Now()
	ticker := time.NewTicker(time.Second)
	statsEnabled = true
	go func() {
		for statsEnabled {
			select {
			case <-ticker.C:
				emitStats()
			}
		}
		ticker.Stop()
		return
	}()
}

func stopStatsTimer() {
	statsEnabled = false
}

var lastStatsTime time.Time = time.Now()

func timeToNextTick() time.Duration {
	nextTick := lastStatsTime.Add(time.Second)
	return time.Until(nextTick)
}

func emitStats() {
	d := time.Since(lastStatsTime)
	lastStatsTime = time.Now()
	seconds := int64(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	ui.emitTestResultBegin()
	emitTestResults(uint64(seconds))
	ui.emitTestResultEnd()
	ui.emitStats(getNetworkStats())
	ui.paint(uint64(seconds))
}

func emitTestResults(s uint64) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for _, k := range gSessionKeys {
		v := gSessions[k]
		ui.emitTestResult(v, TCP, s)
		ui.emitTestResult(v, UDP, s)
		ui.emitTestResult(v, ICMP, s)
	}
}
