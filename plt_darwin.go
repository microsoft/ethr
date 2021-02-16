// +build darwin

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
	"syscall"

	tm "github.com/nsf/termbox-go"
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
			ui.printErr("Failed to load data for interface %q: %v", iface.Name, err)
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
	var data tcpStat
	rawData, err := unix.SysctlRaw("net.inet.tcp.stats")
	if err != nil {
		// return EthrTCPStat{}, errors.Wrap(err, "GetTCPStats: could not get net.inet.tcp.stats")
		return
	}
	buf := bytes.NewReader(rawData)
	binary.Read(buf, binary.LittleEndian, &data)

	// return EthrTCPStat{uint64(data.Sndrexmitpack)}, nil
	// return the TCP Retransmits
	stats.tcpStats.segRetrans = uint64(data.Sndrexmitpack)
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

type tcpStat struct {
	Connattempt                      uint32
	Accepts                          uint32
	Connects                         uint32
	Drops                            uint32
	Conndrops                        uint32
	Closed                           uint32
	Segstimed                        uint32
	Rttupdated                       uint32
	Delack                           uint32
	Timeoutdrop                      uint32
	Rexmttimeo                       uint32
	Persisttimeo                     uint32
	Keeptimeo                        uint32
	Keepprobe                        uint32
	Keepdrops                        uint32
	Sndtotal                         uint32
	Sndpack                          uint32
	Sndbyte                          uint32
	Sndrexmitpack                    uint32
	Sndrexmitbyte                    uint32
	Sndacks                          uint32
	Sndprobe                         uint32
	Sndurg                           uint32
	Sndwinup                         uint32
	Sndctrl                          uint32
	Rcvtotal                         uint32
	Rcvpack                          uint32
	Rcvbyte                          uint32
	Rcvbadsum                        uint32
	Rcvbadoff                        uint32
	Rcvmemdrop                       uint32
	Rcvshort                         uint32
	Rcvduppack                       uint32
	Rcvdupbyte                       uint32
	Rcvpartduppack                   uint32
	Rcvpartdupbyte                   uint32
	Rcvoopack                        uint32
	Rcvoobyte                        uint32
	Rcvpackafterwin                  uint32
	Rcvbyteafterwin                  uint32
	Rcvafterclose                    uint32
	Rcvwinprobe                      uint32
	Rcvdupack                        uint32
	Rcvacktoomuch                    uint32
	Rcvackpack                       uint32
	Rcvackbyte                       uint32
	Rcvwinupd                        uint32
	Pawsdrop                         uint32
	Predack                          uint32
	Preddat                          uint32
	Pcbcachemiss                     uint32
	Cachedrtt                        uint32
	Cachedrttvar                     uint32
	Cachedssthresh                   uint32
	Usedrtt                          uint32
	Usedrttvar                       uint32
	Usedssthresh                     uint32
	Persistdrop                      uint32
	Badsyn                           uint32
	Mturesent                        uint32
	Listendrop                       uint32
	Minmssdrops                      uint32
	Sndrexmitbad                     uint32
	Badrst                           uint32
	Sc_added                         uint32
	Sc_retransmitted                 uint32
	Sc_dupsyn                        uint32
	Sc_dropped                       uint32
	Sc_completed                     uint32
	Sc_bucketoverflow                uint32
	Sc_cacheoverflow                 uint32
	Sc_reset                         uint32
	Sc_stale                         uint32
	Sc_aborted                       uint32
	Sc_badack                        uint32
	Sc_unreach                       uint32
	Sc_zonefail                      uint32
	Sc_sendcookie                    uint32
	Sc_recvcookie                    uint32
	Hc_added                         uint32
	Hc_bucketoverflow                uint32
	Sack_recovery_episode            uint32
	Sack_rexmits                     uint32
	Sack_rexmit_bytes                uint32
	Sack_rcv_blocks                  uint32
	Sack_send_blocks                 uint32
	Sack_sboverflow                  uint32
	Bg_rcvtotal                      uint32
	Rxtfindrop                       uint32
	Fcholdpacket                     uint32
	Coalesced_pack                   uint32
	Flowtbl_full                     uint32
	Flowtbl_collision                uint32
	Lro_twopack                      uint32
	Lro_multpack                     uint32
	Lro_largepack                    uint32
	Limited_txt                      uint32
	Early_rexmt                      uint32
	Sack_ackadv                      uint32
	Rcv_swcsum                       uint32
	Rcv_swcsum_bytes                 uint32
	Rcv6_swcsum                      uint32
	Rcv6_swcsum_bytes                uint32
	Snd_swcsum                       uint32
	Snd_swcsum_bytes                 uint32
	Snd6_swcsum                      uint32
	Snd6_swcsum_bytes                uint32
	Msg_unopkts                      uint32
	Msg_unoappendfail                uint32
	Msg_sndwaithipri                 uint32
	Invalid_mpcap                    uint32
	Invalid_joins                    uint32
	Mpcap_fallback                   uint32
	Join_fallback                    uint32
	Estab_fallback                   uint32
	Invalid_opt                      uint32
	Mp_outofwin                      uint32
	Mp_reducedwin                    uint32
	Mp_badcsum                       uint32
	Mp_oodata                        uint32
	Mp_switches                      uint32
	Mp_rcvtotal                      uint32
	Mp_rcvbytes                      uint32
	Mp_sndpacks                      uint32
	Mp_sndbytes                      uint32
	Join_rxmts                       uint32
	Tailloss_rto                     uint32
	Reordered_pkts                   uint32
	Recovered_pkts                   uint32
	Pto                              uint32
	Rto_after_pto                    uint32
	Tlp_recovery                     uint32
	Tlp_recoverlastpkt               uint32
	Ecn_client_success               uint32
	Ecn_recv_ece                     uint32
	Ecn_sent_ece                     uint32
	Detect_reordering                uint32
	Delay_recovery                   uint32
	Avoid_rxmt                       uint32
	Unnecessary_rxmt                 uint32
	Nostretchack                     uint32
	Rescue_rxmt                      uint32
	Pto_in_recovery                  uint32
	Pmtudbh_reverted                 uint32
	Dsack_disable                    uint32
	Dsack_ackloss                    uint32
	Dsack_badrexmt                   uint32
	Dsack_sent                       uint32
	Dsack_recvd                      uint32
	Dsack_recvd_old                  uint32
	Mp_sel_symtomsd                  uint32
	Mp_sel_rtt                       uint32
	Mp_sel_rto                       uint32
	Mp_sel_peer                      uint32
	Mp_num_probes                    uint32
	Mp_verdowngrade                  uint32
	Drop_after_sleep                 uint32
	Probe_if                         uint32
	Probe_if_conflict                uint32
	Ecn_client_setup                 uint32
	Ecn_server_setup                 uint32
	Ecn_server_success               uint32
	Ecn_lost_synack                  uint32
	Ecn_lost_syn                     uint32
	Ecn_not_supported                uint32
	Ecn_recv_ce                      uint32
	Ecn_conn_recv_ce                 uint32
	Ecn_conn_recv_ece                uint32
	Ecn_conn_plnoce                  uint32
	Ecn_conn_pl_ce                   uint32
	Ecn_conn_nopl_ce                 uint32
	Ecn_fallback_synloss             uint32
	Ecn_fallback_reorder             uint32
	Ecn_fallback_ce                  uint32
	Tfo_syn_data_rcv                 uint32
	Tfo_cookie_req_rcv               uint32
	Tfo_cookie_sent                  uint32
	Tfo_cookie_invalid               uint32
	Tfo_cookie_req                   uint32
	Tfo_cookie_rcv                   uint32
	Tfo_syn_data_sent                uint32
	Tfo_syn_data_acked               uint32
	Tfo_syn_loss                     uint32
	Tfo_blackhole                    uint32
	Tfo_cookie_wrong                 uint32
	Tfo_no_cookie_rcv                uint32
	Tfo_heuristics_disable           uint32
	Tfo_sndblackhole                 uint32
	Mss_to_default                   uint32
	Mss_to_medium                    uint32
	Mss_to_low                       uint32
	Ecn_fallback_droprst             uint32
	Ecn_fallback_droprxmt            uint32
	Ecn_fallback_synrst              uint32
	Mptcp_rcvmemdrop                 uint32
	Mptcp_rcvduppack                 uint32
	Mptcp_rcvpackafterwin            uint32
	Timer_drift_le_1_ms              uint32
	Timer_drift_le_10_ms             uint32
	Timer_drift_le_20_ms             uint32
	Timer_drift_le_50_ms             uint32
	Timer_drift_le_100_ms            uint32
	Timer_drift_le_200_ms            uint32
	Timer_drift_le_500_ms            uint32
	Timer_drift_le_1000_ms           uint32
	Timer_drift_gt_1000_ms           uint32
	Mptcp_handover_attempt           uint32
	Mptcp_interactive_attempt        uint32
	Mptcp_aggregate_attempt          uint32
	Mptcp_fp_handover_attempt        uint32
	Mptcp_fp_interactive_attempt     uint32
	Mptcp_fp_aggregate_attempt       uint32
	Mptcp_heuristic_fallback         uint32
	Mptcp_fp_heuristic_fallback      uint32
	Mptcp_handover_success_wifi      uint32
	Mptcp_handover_success_cell      uint32
	Mptcp_interactive_success        uint32
	Mptcp_aggregate_success          uint32
	Mptcp_fp_handover_success_wifi   uint32
	Mptcp_fp_handover_success_cell   uint32
	Mptcp_fp_interactive_success     uint32
	Mptcp_fp_aggregate_success       uint32
	Mptcp_handover_cell_from_wifi    uint32
	Mptcp_handover_wifi_from_cell    uint32
	Mptcp_interactive_cell_from_wifi uint32
	_                                [4]byte
	Mptcp_handover_cell_bytes        uint64
	Mptcp_interactive_cell_bytes     uint64
	Mptcp_aggregate_cell_bytes       uint64
	Mptcp_handover_all_bytes         uint64
	Mptcp_interactive_all_bytes      uint64
	Mptcp_aggregate_all_bytes        uint64
	Mptcp_back_to_wifi               uint32
	Mptcp_wifi_proxy                 uint32
	Mptcp_cell_proxy                 uint32
	_                                [4]byte
}

func setSockOptInt(fd uintptr, level, opt, val int) (err error) {
	err = syscall.SetsockoptInt(int(fd), level, opt, val)
	if err != nil {
		ui.printErr("Failed to set socket option (%v) to value (%v) during Dial. Error: %s", opt, val, err)
	}
	return
}

func IcmpNewConn(address string) (net.PacketConn, error) {
	dialedConn, err := net.Dial(Icmp(), address)
	if err != nil {
		return nil, err
	}
	localAddr := dialedConn.LocalAddr()
	dialedConn.Close()
	conn, err := net.ListenPacket(Icmp(), localAddr.String())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func VerifyPermissionForTest(testID EthrTestID) {
	if testID.Protocol == ICMP || (testID.Protocol == TCP &&
		(testID.Type == TraceRoute || testID.Type == MyTraceRoute)) {
		if !IsAdmin() {
			ui.printMsg("Warning: You are not running as administrator. For %s based %s",
				protoToString(testID.Protocol), testToString(testID.Type))
			ui.printMsg("test, running as administrator is required.\n")
		}
	}
}

func IsAdmin() bool {
	return true
}

func SetTClass(fd uintptr, tos int) {
	setSockOptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_TCLASS, tos)
}
