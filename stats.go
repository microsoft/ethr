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
	tcpStats    ethrTcpStat
}

type ethrNetDevStat struct {
	interfaceName string
	rxBytes       uint64
	txBytes       uint64
	rxPkts        uint64
	txPkts        uint64
}

type ethrTcpStat struct {
	segRetrans uint64
}

func getNetworkStats() ethrNetStat {
	stats := &ethrNetStat{}

	getNetDevStats(stats)
	sort.SliceStable(stats.netDevStats, func(i, j int) bool {
		return stats.netDevStats[i].interfaceName < stats.netDevStats[j].interfaceName
	})
	getTcpStats(stats)

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

/*
func startStatsTimer(test *ethrTest) {
	ticker := time.NewTicker(time.Second)
	statsEnabled = true
	go func() {
		for statsEnabled {
			select {
			case <-test.done:
				break
			case <-ticker.C:
				cPrintStats(test)
			}
		}
		ticker.Stop()
		return
	}()
}
*/
/*
func emitClientStats(remote string, proto EthrProtocol, bw uint64, bwTestOn bool,
    cps uint64, cpsTestOn bool, pps uint64, ppsTestOn bool, lat uint64, latTestOn bool) {
	if proto == Tcp {
        if bwTestOn {
            if gInterval == 0 {
                ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
                ui.printMsg("[ ID]   Protocol    Interval      Bits/s")
            }
			ui.printMsg("[SUM]     %-5s    %03d-%03d sec   %7s",
				protoToString(proto), gInterval, gInterval+1, bytesToRate(bw))
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
        } else if cpsTestOn {
            if gInterval == 0 {
                ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
                ui.printMsg("Protocol    Interval      Conn/s")
            }
            ui.printMsg("  %-5s    %03d-%03d sec   %7s",
                protoToString(proto), gInterval, gInterval+1, cpsToString(cps))
        }
    } else if ppsTestOn {
            if gInterval == 0 {
                ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
                ui.printMsg("Protocol    Interval      Pkts/s")
            }
            ui.printMsg("  %-5s    %03d-%03d sec   %7s",
                protoToString(proto), gInterval, gInterval+1, ppsToString(pps))
	} else if proto == Http && bwTestOn {
		if gInterval == 0 {
			ui.printMsg("- - - - - - - - - - - - - - - - - - - - - - -")
			ui.printMsg("Protocol    Interval      Bits/s")
		}
		ui.printMsg("  %-5s    %03d-%03d sec   %7s",
			protoToString(proto), gInterval, gInterval+1, bytesToRate(bw))
	}
	gInterval++
}
*/

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
		ui.emitTestResult(v, Tcp)
		ui.emitTestResult(v, Udp)
		ui.emitTestResult(v, Http)
		ui.emitTestResult(v, Https)
		ui.emitTestResult(v, Icmp)
	}
}
