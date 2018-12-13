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
	sort.SliceStable(stats.netDevStats, func(i, j int) bool {
		return stats.netDevStats[i].interfaceName < stats.netDevStats[j].interfaceName
	})
	getTCPStats(stats)

	return *stats
}

func getNetDevStatDiff(curStats ethrNetDevStat, prevNetStats ethrNetStat) ethrNetDevStat {
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
	return curStats
}

var statsEnabled bool

func startStatsTimer() {
	if statsEnabled {
		return
	}
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

func emitStats() {
	ui.emitTestResultBegin()
	emitTestResults()
	ui.emitTestResultEnd()
	ui.emitStats(getNetworkStats())
	ui.paint()
}

func emitTestResults() {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for _, k := range gSessionKeys {
		v := gSessions[k]
		ui.emitTestResult(v, TCP)
		ui.emitTestResult(v, UDP)
		ui.emitTestResult(v, HTTP)
		ui.emitTestResult(v, HTTPS)
		ui.emitTestResult(v, ICMP)
	}
}
