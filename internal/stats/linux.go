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

	if len(fields) < 17 {
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

	retransSeg, err := parseSNMPProcfile(reader)
	if err != nil {
		return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: could not parse /proc/net/snmp")
	}
	return EthrTCPStat{retransSeg}, nil
}

// parseSNMPProcfile parses the /proc/net/snmp file to look for the TCP
// retransmission segments count
func parseSNMPProcfile(reader *bufio.Reader) (uint64, error) {
	// Header line we're looking for:
	// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets
	//      CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "Tcp") {
			continue
		}
		fields := strings.Fields(line)
		intField, err := strconv.ParseUint(fields[12], 10, 64)
		if err != nil {
			continue
		}
		return intField, nil
	}
	return 0, errors.New("parseSNMPProcfile: could not find a valid number")
}
