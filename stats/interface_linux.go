// +build linux

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"

	tm "github.com/nsf/termbox-go"
)

type NetDevInfo struct {
	bytes      uint64
	packets    uint64
	drop       uint64
	errs       uint64
	fifo       uint64
	frame      uint64
	compressed uint64
	multicast  uint64
}

type osStats struct {
}

func getNetDevStats(stats *NetStat) {
	ifs, err := net.Interfaces()
	if err != nil {
		Logger.Error("%v", err)
		return
	}

	netStatsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		Logger.Error("%v", err)
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
		if isIfUp(netDevStat.InterfaceName, ifs) {
			stats.Devices = append(stats.Devices, buildNetDevStat(line))
		}
	}
}

func getTCPStats(stats *NetStat) {
	snmpStatsFile, err := os.Open("/proc/net/snmp")
	if err != nil {
		Logger.Debug("%v", err)
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
		stats.TCP.RetransmittedSegments = toUInt64(fields[12])
	}
}

func hideCursor() {
	tm.SetCursor(0, 0)
}

func blockWindowResize() {
}

func buildNetDevStat(line string) DeviceStats {
	fields := strings.Fields(line)
	if len(fields) < 17 {
		return DeviceStats{}
	}
	interfaceName := strings.TrimSuffix(fields[0], ":")
	rxInfo := toNetDevInfo(fields[1:9])
	txInfo := toNetDevInfo(fields[9:17])
	return DeviceStats{
		InterfaceName: interfaceName,
		RXBytes:       rxInfo.bytes,
		TXBytes:       txInfo.bytes,
		RXPackets:     rxInfo.packets,
		TXPackets:     txInfo.packets,
	}
}

func toNetDevInfo(fields []string) NetDevInfo {
	return NetDevInfo{
		bytes:      toUInt64(fields[0]),
		packets:    toUInt64(fields[1]),
		errs:       toUInt64(fields[2]),
		drop:       toUInt64(fields[3]),
		fifo:       toUInt64(fields[4]),
		frame:      toUInt64(fields[5]),
		compressed: toUInt64(fields[6]),
		multicast:  toUInt64(fields[7]),
	}
}

func toUInt64(str string) uint64 {
	res, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		Logger.Debug("Error in string conversion: %v", err)
		return 0
	}
	return res
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
