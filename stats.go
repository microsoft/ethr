//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/microsoft/ethr/internal/stats"
)

func getNetworkStats() (stats.EthrNetStats, error) {
	osStats := stats.GetOSStats()
	devStats, err := osStats.GetNetDevStats()
	if err != nil {
		return stats.EthrNetStats{}, errors.Wrap(err, "getNetworkStats: could not get net device stats")
	}
	sort.SliceStable(devStats, func(i, j int) bool {
		return devStats[i].InterfaceName < devStats[j].InterfaceName
	})

	tcpStats, err := osStats.GetTCPStats()
	if err != nil {
		return stats.EthrNetStats{}, errors.Wrap(err, "getNetworkStats: could not get net TCP stats")
	}

	return stats.EthrNetStats{NetDevStats: devStats, TCPStats: tcpStats}, nil
}

func getNetDevStatDiff(curStats stats.EthrNetDevStat, prevNetStats stats.EthrNetStats, seconds uint64) stats.EthrNetDevStat {
	for _, prevStats := range prevNetStats.NetDevStats {
		if prevStats.InterfaceName != curStats.InterfaceName {
			continue
		}

		if curStats.RxBytes >= prevStats.RxBytes {
			curStats.RxBytes -= prevStats.RxBytes
		} else {
			curStats.RxBytes += (^uint64(0) - prevStats.RxBytes)
		}

		if curStats.TxBytes >= prevStats.TxBytes {
			curStats.TxBytes -= prevStats.TxBytes
		} else {
			curStats.TxBytes += (^uint64(0) - prevStats.TxBytes)
		}

		if curStats.RxPkts >= prevStats.RxPkts {
			curStats.RxPkts -= prevStats.RxPkts
		} else {
			curStats.RxPkts += (^uint64(0) - prevStats.RxPkts)
		}

		if curStats.TxPkts >= prevStats.TxPkts {
			curStats.TxPkts -= prevStats.TxPkts
		} else {
			curStats.TxPkts += (^uint64(0) - prevStats.TxPkts)
		}

		break
	}
	curStats.RxBytes /= seconds
	curStats.TxBytes /= seconds
	curStats.RxPkts /= seconds
	curStats.TxPkts /= seconds
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
	stats, err := getNetworkStats()
	if err != nil {
		ui.printErr("could not get network stats")
	}
	ui.emitStats(stats)
	ui.paint(uint64(seconds))
}

func emitTestResults(s uint64) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for _, k := range gSessionKeys {
		v := gSessions[k]
		ui.emitTestResult(v, TCP, s)
		ui.emitTestResult(v, UDP, s)
		ui.emitTestResult(v, HTTP, s)
		ui.emitTestResult(v, HTTPS, s)
		ui.emitTestResult(v, ICMP, s)
	}
}
