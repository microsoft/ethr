//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tm "github.com/nsf/termbox-go"
)

var netstatPath = "/usr/sbin/netstat"

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

	for _, netIf := range ifs {
		if netIf.Flags&net.FlagUp == 0 {
			continue
		}

		netIfStats, err := getNetInterfaceStats(netIf.Name)
		if err != nil {
			ui.printErr("Failed to get stats for interface %s: %v", netIf.Name, err)
			continue
		}
		stats.netDevStats = append(stats.netDevStats, netIfStats)
	}
}

func getNetInterfaceStats(name string) (ethrNetDevStat, error) {

	var intfStats ethrNetDevStat

	// use netstat to get the interface stats
	args := []string{"-ib", "-I", name}
	output, err := exec.Command(netstatPath, args...).Output()
	if err != nil {
		return intfStats, err
	}
	lines := strings.Split(string(output), "\n")
	numLines := len(lines)

	if numLines <= 1 {
		return intfStats, fmt.Errorf("No interface stats available for %s", name)
	}

	// Name  Mtu   Network       Address            Ipkts Ierrs     Ibytes    Opkts Oerrs     Obytes  Coll
	// en0   1500  <Link#4>    18:65:90:d2:af:c7   859944     0  773619217   714294     0  183006466     0
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		return ethrNetDevStat{
			interfaceName: name,
			rxBytes:       toInt(fields[6]),
			txBytes:       toInt(fields[9]),
			rxPkts:        toInt(fields[4]),
			txPkts:        toInt(fields[7]),
		}, nil
	}
	return intfStats, nil
}

func getTcpStats(stats *ethrNetStat) {
	// use netstat to get the interface stats
	args := []string{"-s", "-p", "tcp"}
	output, err := exec.Command(netstatPath, args...).Output()
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	match := regexp.MustCompile("(?m)^\\s*(\\d+) data packets \\((\\d+) bytes\\) retransmitted").FindStringSubmatch(string(output))
	stats.tcpStats.segRetrans = toInt(match[1])
}

func toInt(str string) uint64 {
	res, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		panic(err)
	}
	return res
}

func hideCursor() {
	tm.SetCursor(0, 0)
}

func blockWindowResize() {
}
