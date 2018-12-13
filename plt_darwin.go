//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"bytes"
	"encoding/binary"
	"net"

	tm "github.com/nsf/termbox-go"
	"golang.org/x/sys/unix"
)

/*#include <sys/socket.h>
#include <sys/socketvar.h>
#include <netinet/in.h>
#include<netinet/tcp_var.h>*/
import "C"

func getNetDevStats(stats *ethrNetStat) {
	ifs, err := net.Interfaces()
	if err != nil {
		ui.printErr("%v", err)
		return
	}

	for _, iface := range ifs {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		ifaceData, err := getIfaceData(iface.Index)
		if err != nil {
			ui.printErr("failed to load data for interface %q: %v", iface.Name, err)
			continue
		}

		stats.netDevStats = append(stats.netDevStats, ethrNetDevStat{
			interfaceName: iface.Name,
			rxBytes:       ifaceData.Data.Ibytes,
			rxPkts:        ifaceData.Data.Ipackets,
			txBytes:       ifaceData.Data.Obytes,
			txPkts:        ifaceData.Data.Opackets,
		})
	}
}

func getTcpStats(stats *ethrNetStat) {
	var data C.struct_tcpstat
	rawData, err := unix.SysctlRaw("net.inet.tcp.stats")
	if err != nil {
		return
	}
	buf := bytes.NewReader(rawData)
	// This is ugly. Cannot read the full structure since the
	// C struct has unexported fields, which reflect does not like
	binary.Read(buf, binary.LittleEndian, &data.tcps_connattempt)
	binary.Read(buf, binary.LittleEndian, &data.tcps_accepts)
	binary.Read(buf, binary.LittleEndian, &data.tcps_connects)
	binary.Read(buf, binary.LittleEndian, &data.tcps_drops)
	binary.Read(buf, binary.LittleEndian, &data.tcps_conndrops)
	binary.Read(buf, binary.LittleEndian, &data.tcps_closed)
	binary.Read(buf, binary.LittleEndian, &data.tcps_segstimed)
	binary.Read(buf, binary.LittleEndian, &data.tcps_rttupdated)
	binary.Read(buf, binary.LittleEndian, &data.tcps_delack)
	binary.Read(buf, binary.LittleEndian, &data.tcps_timeoutdrop)
	binary.Read(buf, binary.LittleEndian, &data.tcps_rexmttimeo)
	binary.Read(buf, binary.LittleEndian, &data.tcps_persisttimeo)
	binary.Read(buf, binary.LittleEndian, &data.tcps_keeptimeo)
	binary.Read(buf, binary.LittleEndian, &data.tcps_keepprobe)
	binary.Read(buf, binary.LittleEndian, &data.tcps_keepdrops)
	binary.Read(buf, binary.LittleEndian, &data.tcps_sndtotal)
	binary.Read(buf, binary.LittleEndian, &data.tcps_sndpack)
	binary.Read(buf, binary.LittleEndian, &data.tcps_sndbyte)
	binary.Read(buf, binary.LittleEndian, &data.tcps_sndrexmitpack)

	// return the TCP Retransmits
	stats.tcpStats.segRetrans = uint64(data.tcps_sndrexmitpack)
	return
}

func hideCursor() {
	tm.SetCursor(0, 0)
}

func blockWindowResize() {
}

func getIfaceData(index int) (*ifMsghdr2, error) {
	var data ifMsghdr2
	rawData, err := unix.SysctlRaw("net", unix.AF_ROUTE, 0, 0, unix.NET_RT_IFLIST2, index)
	if err != nil {
		return nil, err
	}
	err = binary.Read(bytes.NewReader(rawData), binary.LittleEndian, &data)
	return &data, err
}

type ifMsghdr2 struct {
	Msglen    uint16
	Version   uint8
	Type      uint8
	Addrs     int32
	Flags     int32
	Index     uint16
	_         [2]byte
	SndLen    int32
	SndMaxlen int32
	SndDrops  int32
	Timer     int32
	Data      ifData64
}

type ifData64 struct {
	Type       uint8
	Typelen    uint8
	Physical   uint8
	Addrlen    uint8
	Hdrlen     uint8
	Recvquota  uint8
	Xmitquota  uint8
	Unused1    uint8
	Mtu        uint32
	Metric     uint32
	Baudrate   uint64
	Ipackets   uint64
	Ierrors    uint64
	Opackets   uint64
	Oerrors    uint64
	Collisions uint64
	Ibytes     uint64
	Obytes     uint64
	Imcasts    uint64
	Omcasts    uint64
	Iqdrops    uint64
	Noproto    uint64
	Recvtiming uint32
	Xmittiming uint32
	Lastchange unix.Timeval32
}
