//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

type EthrNetStats struct {
	NetDevStats []EthrNetDevStat
	TCPStats    EthrTCPStat
}

type EthrNetDevStat struct {
	InterfaceName string
	RxBytes       uint64
	TxBytes       uint64
	RxPkts        uint64
	TxPkts        uint64
}

type EthrTCPStat struct {
	SegRetrans uint64
}
