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

	"github.com/nsf/termbox-go"
	"golang.org/x/sys/unix"
)

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

func getTCPStats(stats *ethrNetStat) {
}

func hideCursor() {
	termbox.SetCursor(0, 0)
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
