//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"net"
	"strings"
	"syscall"
	"unsafe"

	tm "github.com/nsf/termbox-go"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var user32 = syscall.NewLazyDLL("user32.dll")
var iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

var (
	proc_get_tcp_statistics_ex = iphlpapi.NewProc("GetTcpStatisticsEx")
	proc_get_console_window    = kernel32.NewProc("GetConsoleWindow")
	proc_get_system_menu       = user32.NewProc("GetSystemMenu")
	proc_delete_menu           = user32.NewProc("DeleteMenu")
	proc_get_if_entry2         = iphlpapi.NewProc("GetIfEntry2")
)

type ethrNetDevInfo struct {
	bytes   uint64
	packets uint64
	drop    uint64
	errs    uint64
}

func getNetDevStats(stats *ethrNetStat) {
	ifs, err := net.Interfaces()
	if err != nil {
		ui.printErr("%v", err)
		return
	}

	for _, ifi := range ifs {
		if (ifi.Flags&net.FlagUp) == 0 || strings.Contains(ifi.Name, "Pseudo") {
			continue
		}
		row := mibIfRow2{InterfaceIndex: uint32(ifi.Index)}
		e := getIfEntry2(&row)
		if e != nil {
			ui.printErr("%v", err)
			return
		}
		rxInfo := ethrNetDevInfo{
			bytes:   uint64(row.InOctets),
			packets: uint64(row.InUcastPkts),
			drop:    uint64(row.InDiscards),
			errs:    uint64(row.InErrors),
		}
		txInfo := ethrNetDevInfo{
			bytes:   uint64(row.OutOctets),
			packets: uint64(row.OutUcastPkts),
			drop:    uint64(row.OutDiscards),
			errs:    uint64(row.OutErrors),
		}
		netStats := ethrNetDevStat{
			interfaceName: ifi.Name,
			rxBytes:       rxInfo.bytes,
			txBytes:       txInfo.bytes,
			rxPkts:        rxInfo.packets,
			txPkts:        txInfo.packets,
		}
		stats.netDevStats = append(stats.netDevStats, netStats)
	}
}

type mib_tcpstats struct {
	DwRtoAlgorithm uint32
	DwRtoMin       uint32
	DwRtoMax       uint32
	DwMaxConn      uint32
	DwActiveOpens  uint32
	DwPassiveOpens uint32
	DwAttemptFails uint32
	DwEstabResets  uint32
	DwCurrEstab    uint32
	DwInSegs       uint32
	DwOutSegs      uint32
	DwRetransSegs  uint32
	DwInErrs       uint32
	DwOutRsts      uint32
	DwNumConns     uint32
}

const (
	AF_INET  = 2
	AF_INET6 = 23
)

func getTCPStats(stats *ethrNetStat) (errcode error) {
	tcpStats := &mib_tcpstats{}
	r0, _, _ := syscall.Syscall(proc_get_tcp_statistics_ex.Addr(), 2,
		uintptr(unsafe.Pointer(tcpStats)),
		uintptr(AF_INET),
		0)

	if r0 != 0 {
		errcode = syscall.Errno(r0)
		return
	}
	stats.tcpStats.segRetrans = uint64(tcpStats.DwRetransSegs)
	return
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

const (
	MAX_STRING_SIZE         = 256
	MAX_PHYS_ADDRESS_LENGTH = 32
	pad0for64_4for32        = 0
)

type mibIfRow2 struct {
	InterfaceLuid               uint64
	InterfaceIndex              uint32
	InterfaceGuid               guid
	Alias                       [MAX_STRING_SIZE + 1]uint16
	Description                 [MAX_STRING_SIZE + 1]uint16
	PhysicalAddressLength       uint32
	PhysicalAddress             [MAX_PHYS_ADDRESS_LENGTH]uint8
	PermanentPhysicalAddress    [MAX_PHYS_ADDRESS_LENGTH]uint8
	Mtu                         uint32
	Type                        uint32
	TunnelType                  uint32
	MediaType                   uint32
	PhysicalMediumType          uint32
	AccessType                  uint32
	DirectionType               uint32
	InterfaceAndOperStatusFlags uint32
	OperStatus                  uint32
	AdminStatus                 uint32
	MediaConnectState           uint32
	NetworkGuid                 guid
	ConnectionType              uint32
	padding1                    [pad0for64_4for32]byte
	TransmitLinkSpeed           uint64
	ReceiveLinkSpeed            uint64
	InOctets                    uint64
	InUcastPkts                 uint64
	InNUcastPkts                uint64
	InDiscards                  uint64
	InErrors                    uint64
	InUnknownProtos             uint64
	InUcastOctets               uint64
	InMulticastOctets           uint64
	InBroadcastOctets           uint64
	OutOctets                   uint64
	OutUcastPkts                uint64
	OutNUcastPkts               uint64
	OutDiscards                 uint64
	OutErrors                   uint64
	OutUcastOctets              uint64
	OutMulticastOctets          uint64
	OutBroadcastOctets          uint64
	OutQLen                     uint64
}

func getIfEntry2(row *mibIfRow2) (errcode error) {
	r0, _, _ := syscall.Syscall(proc_get_if_entry2.Addr(), 1,
		uintptr(unsafe.Pointer(row)), 0, 0)
	if r0 != 0 {
		errcode = syscall.Errno(r0)
	}
	return
}

func hideCursor() {
	tm.HideCursor()
}

const (
	MF_BYCOMMAND = 0x00000000
	SC_CLOSE     = 0xF060
	SC_MINIMIZE  = 0xF020
	SC_MAXIMIZE  = 0xF030
	SC_SIZE      = 0xF000
)

func blockWindowResize() {
	h, _, err := syscall.Syscall(proc_get_console_window.Addr(), 0, 0, 0, 0)
	if err != 0 {
		return
	}

	sysMenu, _, err := syscall.Syscall(proc_get_system_menu.Addr(), 2, h, 0, 0)
	if err != 0 {
		return
	}

	syscall.Syscall(proc_delete_menu.Addr(), 3, sysMenu, SC_MAXIMIZE, MF_BYCOMMAND)
	syscall.Syscall(proc_delete_menu.Addr(), 3, sysMenu, SC_SIZE, MF_BYCOMMAND)
}
