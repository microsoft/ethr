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
var statsTicker *time.Ticker

func startStatsTimer() {
	if statsEnabled {
		return
	}

	// Start stats at second boundary so that client can align to server's timing
	SleepUntilNextWholeSecond()

	lastStatsTime = time.Now()
	statsTicker = time.NewTicker(time.Second)
	statsEnabled = true
	go func() {
		for statsEnabled {
			select {
			case <-statsTicker.C:
				emitStats()
			}
		}
		statsTicker.Stop()
	}()
}

// startStatsTimerAt starts the stats timer synchronized to a specific start time
// This is used when client and server have negotiated an exact start time
// Uses absolute time targets to prevent timer drift
func startStatsTimerAt(startTime time.Time) {
	if statsEnabled {
		return
	}

	// Set lastStatsTime to the synchronized start time
	lastStatsTime = startTime
	statsEnabled = true

	// Run the timer in a goroutine so we don't block the caller
	go func() {
		// Use absolute time targets to prevent drift
		// Each measurement happens at exactly startTime + N seconds
		interval := 1
		for statsEnabled {
			// Calculate the exact target time for this interval
			targetTime := startTime.Add(time.Duration(interval) * time.Second)
			sleepDuration := time.Until(targetTime)
			
			if sleepDuration > 0 {
				time.Sleep(sleepDuration)
			}
			
			if !statsEnabled {
				break
			}
			
			emitStats()
			interval++
		}
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
	// Use actual elapsed time for accurate rate calculations
	seconds := d.Seconds()
	if seconds < 1.0 {
		seconds = 1.0
	}
	ui.emitTestResultBegin()
	emitTestResults(seconds)
	ui.emitTestResultEnd()
	ui.emitStats(getNetworkStats())
	ui.paint(uint64(seconds))
}

func emitTestResults(s float64) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for _, k := range gSessionKeys {
		v := gSessions[k]
		ui.emitTestResult(v, TCP, s)
		ui.emitTestResult(v, UDP, s)
		ui.emitTestResult(v, ICMP, s)
	}
}
