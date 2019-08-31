// +build linux

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type linuxEthrNetDevInfo struct {
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

func (s osStats) GetNetDevStats() ([]EthrNetDevStat, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "GetNetDevStats: error getting network interfaces")
	}

	netStatsFile, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, errors.Wrap(err, "GetNetDevStats: error opening /proc/net/dev")
	}
	defer netStatsFile.Close()

	reader := bufio.NewReader(netStatsFile)

	// Pass the header
	// Inter-|   Receive                                             |  Transmit
	//  face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed
	reader.ReadString('\n')
	reader.ReadString('\n')

	line, err := reader.ReadString('\n')
	if line == "" {
		return nil, nil // TODO: possibly return an error here
	}
	var res []EthrNetDevStat

	netDevStat, err := buildNetDevStat(line)
	if err != nil {
		return nil, err
	}

	if isIfUp(netDevStat.InterfaceName, ifs) {
		devStats, err := buildNetDevStat(line)
		if err != nil {
			return nil, errors.Wrap(err, "GetNetDevStats: could not build interface stats")
		}
		res = append(res, devStats)
	}
	return res, nil
}

func buildNetDevStat(line string) (EthrNetDevStat, error) {
	fields := strings.Fields(line)
	interfaceName := strings.TrimSuffix(fields[0], ":")

	if len(fields) < 18 {
		return EthrNetDevStat{}, errors.New(
			fmt.Sprintf(
				"buildNetDevStat: unexpected net stats file format, erroneous line %s",
				line))
	}

	rxInfo, err := toNetDevInfo(fields[1:9])
	if err != nil {
		return EthrNetDevStat{}, errors.Wrap(err, "buildNetDevStat: error parsing rxInfo")
	}

	txInfo, err := toNetDevInfo(fields[9:17])
	if err != nil {
		return EthrNetDevStat{}, errors.Wrap(err, "buildNetDevStat: error parsing txInfo")
	}

	return EthrNetDevStat{
		InterfaceName: interfaceName,
		RxBytes:       rxInfo.bytes,
		TxBytes:       txInfo.bytes,
		RxPkts:        rxInfo.packets,
		TxPkts:        txInfo.packets,
	}, nil
}

func toNetDevInfo(fields []string) (linuxEthrNetDevInfo, error) {
	var err error

	intFields := [8]uint64{}
	for i := 0; i < 8; i++ {
		intFields[i], err = strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return linuxEthrNetDevInfo{}, errors.Wrap(err,
				"toNetDevInfo: error in string conversion")
		}
	}

	return linuxEthrNetDevInfo{
		bytes:      intFields[0],
		packets:    intFields[1],
		errs:       intFields[2],
		drop:       intFields[3],
		fifo:       intFields[4],
		frame:      intFields[5],
		compressed: intFields[6],
		multicast:  intFields[7],
	}, nil
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

func (s osStats) GetTCPStats() (EthrTCPStat, error) {
	snmpStatsFile, err := os.Open("/proc/net/snmp")
	if err != nil {
		return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: error opening /proc/net/snmp")
	}
	defer snmpStatsFile.Close()

	reader := bufio.NewReader(snmpStatsFile)

	// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets
	//      CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
	line, err := reader.ReadString('\n')
	if err != nil {
		return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: error reading from /proc/net/snmp")
	}
	if line == "" || !strings.HasPrefix(line, "Tcp") {
		return EthrTCPStat{}, errors.New("GetTCPStats: could not find a TCP info")
	}

	// Skip the first line starting with Tcp
	line, err = reader.ReadString('\n')
	if err != nil {
		return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: error reading from /proc/net/snmp")
	}
	if !strings.HasPrefix(line, "Tcp") {
		return EthrTCPStat{}, errors.New("GetTCPStats: could not find TCP info")
	}

	fields := strings.Fields(line)
	intField, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: could not convert field data to integer")
	}
	return EthrTCPStat{intField}, nil
}
