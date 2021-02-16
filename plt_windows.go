// +build windows

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"context"
	"net"
	"os"
	"strings"
	"syscall"
	"unsafe"

	tm "github.com/nsf/termbox-go"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	proc_get_tcp_statistics_ex = iphlpapi.NewProc("GetTcpStatisticsEx")
	proc_get_if_entry2         = iphlpapi.NewProc("GetIfEntry2")
	proc_get_console_window    = kernel32.NewProc("GetConsoleWindow")
	proc_get_system_menu       = user32.NewProc("GetSystemMenu")
	proc_delete_menu           = user32.NewProc("DeleteMenu")
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
		row, err := getIfEntry2(uint32(ifi.Index))
		if err != nil {
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
		uintptr(unsafe.Pointer(tcpStats)), uintptr(AF_INET), 0)

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

func getIfEntry2(ifIndex uint32) (mibIfRow2, error) {
	var res *mibIfRow2

	res = &mibIfRow2{InterfaceIndex: ifIndex}
	r0, _, _ := syscall.Syscall(proc_get_if_entry2.Addr(), 1,
		uintptr(unsafe.Pointer(res)), 0, 0)
	if r0 != 0 {
		return mibIfRow2{}, syscall.Errno(r0)
	}
	return *res, nil
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

func setSockOptInt(fd uintptr, level, opt, val int) (err error) {
	err = syscall.SetsockoptInt(syscall.Handle(fd), level, opt, val)
	if err != nil {
		ui.printErr("Failed to set socket option (%v) to value (%v) during Dial. Error: %s", opt, val, err)
	}
	return
}

const (
	SIO_RCVALL             = syscall.IOC_IN | syscall.IOC_VENDOR | 1
	RCVALL_OFF             = 0
	RCVALL_ON              = 1
	RCVALL_SOCKETLEVELONLY = 2
	RCVALL_IPLEVEL         = 3
)

func IcmpNewConn(address string) (net.PacketConn, error) {
	// This is an attempt to work around the problem described here:
	// https://github.com/golang/go/issues/38427

	// First, get the correct local interface address, as SIO_RCVALL can't be set on a 0.0.0.0 listeners.
	dialedConn, err := net.Dial(Icmp(), address)
	if err != nil {
		return nil, err
	}
	localAddr := dialedConn.LocalAddr()
	dialedConn.Close()

	// Configure the setup routine in order to extract the socket handle.
	var socketHandle syscall.Handle
	cfg := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(s uintptr) {
				socketHandle = syscall.Handle(s)
			})
		},
	}

	// Bind to interface.
	conn, err := cfg.ListenPacket(context.Background(), Icmp(), localAddr.String())
	if err != nil {
		return nil, err
	}

	// Set socket option to receive all packets, such as ICMP error messages.
	// This is somewhat dirty, as there is guarantee that socketHandle is still valid.
	// WARNING: The Windows Firewall might just drop the incoming packets you might want to receive.
	unused := uint32(0) // Documentation states that this is unused, but WSAIoctl fails without it.
	flag := uint32(RCVALL_IPLEVEL)
	size := uint32(unsafe.Sizeof(flag))
	err = syscall.WSAIoctl(socketHandle, SIO_RCVALL, (*byte)(unsafe.Pointer(&flag)), size, nil, 0, &unused, nil, 0)
	if err != nil {
		// Ignore the error as for ICMP related TraceRoute, this is not required.
	}

	return conn, nil
}

func VerifyPermissionForTest(testID EthrTestID) {
	if (testID.Type == TraceRoute || testID.Type == MyTraceRoute) &&
		(testID.Protocol == TCP) {
		if !IsAdmin() {
			ui.printMsg("Warning: You are not running as administrator. For %s based %s",
				protoToString(testID.Protocol), testToString(testID.Type))
			ui.printMsg("test, running as administrator is required.\n")
		}
	}
}

func IsAdmin() bool {
	c, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		ui.printDbg("Process is not running as admin. Error: %v", err)
		return false
	}
	c.Close()
	return true
}

func SetTClass(fd uintptr, tos int) {
	return
}
