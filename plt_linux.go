//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"

	tm "github.com/nsf/termbox-go"
)

type ethrNetDevInfo struct {
	bytes      uint64
	packets    uint64
	drop       uint64
	errs       uint64
	fifo       uint64
	frame      uint64
	compressed uint64
	multicast  uint64
}

func getNetDevStats(stats *ethrNetStat) {
	ifs, err := net.Interfaces()
	if err != nil {
		ui.printErr("%v", err)
		return
	}

	netStatsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	defer netStatsFile.Close()

	reader := bufio.NewReader(netStatsFile)

	// Pass the header
	// Inter-|   Receive                                             |  Transmit
	//  face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed
	reader.ReadString('\n')
	reader.ReadString('\n')

	var line string
	for err == nil {
		line, err = reader.ReadString('\n')
		if line == "" {
			continue
		}
		netDevStat := buildNetDevStat(line)
		if isIfUp(netDevStat.interfaceName, ifs) {
			stats.netDevStats = append(stats.netDevStats, buildNetDevStat(line))
		}
	}
}

func buildNetDevStat(line string) ethrNetDevStat {
	fields := strings.Fields(line)
	interfaceName := strings.TrimSuffix(fields[0], ":")
	rxInfo := toNetDevInfo(fields[1:9])
	txInfo := toNetDevInfo(fields[9:17])
	return ethrNetDevStat{
		interfaceName: interfaceName,
		rxBytes:       rxInfo.bytes,
		txBytes:       txInfo.bytes,
		rxPkts:        rxInfo.packets,
		txPkts:        txInfo.packets,
	}
}

func toNetDevInfo(fields []string) ethrNetDevInfo {
	return ethrNetDevInfo{
		bytes:      toInt(fields[0]),
		packets:    toInt(fields[1]),
		errs:       toInt(fields[2]),
		drop:       toInt(fields[3]),
		fifo:       toInt(fields[4]),
		frame:      toInt(fields[5]),
		compressed: toInt(fields[6]),
		multicast:  toInt(fields[7]),
	}
}

func isIfUp(ifName string, ifs []net.Interface) bool {
	for _, ifi := range ifs {
		if ifi.Name == ifName {
			if (ifi.Flags & net.FlagUp) != 0 {
				return true
			}
			return false
		}
	}
	return false
}

func toInt(str string) uint64 {
	res, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		panic(err)
	}
	return res
}

func getTCPStats(stats *ethrNetStat) {
	snmpStatsFile, err := os.Open("/proc/net/snmp")
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	defer snmpStatsFile.Close()

	reader := bufio.NewReader(snmpStatsFile)

	var line string
	for err == nil {
		// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets
		//      CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
		line, err = reader.ReadString('\n')
		if line == "" || !strings.HasPrefix(line, "Tcp") {
			continue
		}
		// Skip the first line starting with Tcp
		line, err = reader.ReadString('\n')
		if !strings.HasPrefix(line, "Tcp") {
			break
		}
		fields := strings.Fields(line)
		stats.tcpStats.segRetrans = toInt(fields[12])
	}
}

func hideCursor() {
	tm.SetCursor(0, 0)
}

func blockWindowResize() {
}
