// +build windows

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

import (
	"net"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

var iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

var (
	proc_get_tcp_statistics_ex = iphlpapi.NewProc("GetTcpStatisticsEx")
	proc_get_if_entry2         = iphlpapi.NewProc("GetIfEntry2")
)

type winEthrNetDevInfo struct {
	bytes   uint64
	packets uint64
	drop    uint64
	errs    uint64
}

type osStats struct{}

func (s osStats) GetNetDevStats() ([]EthrNetDevStat, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "GetNetDevStats: error getting network interfaces")
	}

	var res []EthrNetDevStat

	for _, ifi := range ifs {
		if (ifi.Flags&net.FlagUp) == 0 || strings.Contains(ifi.Name, "Pseudo") {
			continue
		}
		row, err := getIfEntry2(uint32(ifi.Index))
		if err != nil {
			return nil, errors.Wrap(err, "GetNetDevStats:")
		}
		rxInfo := winEthrNetDevInfo{
			bytes:   uint64(row.InOctets),
			packets: uint64(row.InUcastPkts),
			drop:    uint64(row.InDiscards),
			errs:    uint64(row.InErrors),
		}
		txInfo := winEthrNetDevInfo{
			bytes:   uint64(row.OutOctets),
			packets: uint64(row.OutUcastPkts),
			drop:    uint64(row.OutDiscards),
			errs:    uint64(row.OutErrors),
		}
		netStats := EthrNetDevStat{
			InterfaceName: ifi.Name,
			RxBytes:       rxInfo.bytes,
			TxBytes:       txInfo.bytes,
			RxPkts:        rxInfo.packets,
			TxPkts:        txInfo.packets,
		}
		res = append(res, netStats)
	}
	return res, nil
}

type mibTCPStats struct {
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
	afInet  = 2
	afInet6 = 23
)

func (s osStats) GetTCPStats() (EthrTCPStat, error) {
	tcpStats := &mibTCPStats{}
	r0, _, _ := syscall.Syscall(proc_get_tcp_statistics_ex.Addr(), 2,
		uintptr(unsafe.Pointer(tcpStats)), uintptr(afInet), 0)

	if r0 != 0 {
		errcode := syscall.Errno(r0)
		return EthrTCPStat{}, errcode
	}
	return EthrTCPStat{uint64(tcpStats.DwRetransSegs)}, nil
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

const (
	maxStringSize        = 256
	maxPhysAddressLength = 32
	pad0for64_4for32     = 0
)

type mibIfRow2 struct {
	InterfaceLuid               uint64
	InterfaceIndex              uint32
	InterfaceGuid               guid
	Alias                       [maxStringSize + 1]uint16
	Description                 [maxStringSize + 1]uint16
	PhysicalAddressLength       uint32
	PhysicalAddress             [maxPhysAddressLength]uint8
	PermanentPhysicalAddress    [maxPhysAddressLength]uint8
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
