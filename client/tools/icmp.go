package tools

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"weavelab.xyz/ethr/ethr"
)

func (t Tools) ReceiveICMPFromPeer(pc net.PacketConn, timeout time.Duration, neededPeer string) (*icmp.Message, net.Addr, error) {
	err := pc.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set deadline: %w", err)
	}
	for {
		peerAddr := ""
		b := make([]byte, 1500)
		n, peer, err := pc.ReadFrom(b)
		if err != nil {
			//if protocol == ethr.ICMP {
			//	// In case of non-ICMP TraceRoute, it is expected that no packet is received
			//	// in some case, e.g. when packet reach final hop and TCP connection establishes.
			//	ui.printDbg("Failed to receive ICMP packet. Error: %v", err)
			//}
			return nil, nil, fmt.Errorf("failed to receive ICMP packet: %w", err)
		}
		if n == 0 {
			continue
		}
		//t.Logger.Debug("Packet:\n%s", hex.Dump(b[:n]))
		//t.Logger.Debug("Finding Pattern\n%v", hex.Dump(neededSig[:4]))

		peerAddr = peer.String()
		if neededPeer != "" && peerAddr != neededPeer {
			//t.Logger.Debug("Matching peer is not found.")
			continue
		}
		icmpMsg, err := icmp.ParseMessage(ethr.ICMPProtocolNumber(t.IPVersion), b[:n])
		if err != nil {
			//t.Logger.Debug("Failed to parse ICMP message: %w", err)
			continue
		}

		return icmpMsg, peer, nil
	}
}

// https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol
// ICMP errors return the original IP packet header and the first 8 bytes of the original message
// This is useful in ICMP Traceroute to determine if this is a response to our original request
func (t Tools) UnwrapICMPMessage(index int, body []byte) (*icmp.Message, error) {
	if index < 4 {
		return nil, fmt.Errorf("incorrect length of icmp message")
	}
	unwrapped, err := icmp.ParseMessage(ethr.ICMPProtocolNumber(t.IPVersion), body[index-4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap icmp packet")
	}
	return unwrapped, nil
}

// TODO move this to a traceroute helper fn
//if icmpMsg.Type == ipv4.ICMPTypeTimeExceeded || icmpMsg.Type == ipv6.ICMPTypeTimeExceeded {
//	body := icmpMsg.Body.(*icmp.TimeExceeded).Data
//	index := bytes.Index(body, neededSig[:4])
//	if index > 0 {
//		if protocol == ethr.TCP {
//			//c.Logger.Debug("Found correct ICMP error message. PeerAddr: %v", peerAddr)
//			return peerAddr, isLast, nil
//		} else if protocol == ethr.ICMP {
//			if index < 4 {
//				//c.Logger.Debug("Incorrect length of ICMP message.")
//				continue
//			}
//			innerIcmpMsg, _ := icmp.ParseMessage(ethr.ICMPProtocolNumber(c.IPVersion), body[index-4:])
//			switch innerIcmpMsg.Body.(type) {
//			case *icmp.Echo:
//				seq := innerIcmpMsg.Body.(*icmp.Echo).Seq
//				if seq == neededIcmpSeq {
//					return peerAddr, isLast, nil
//				}
//			default:
//				// Ignore as this is not the right ICMP packet.
//				//c.Logger.Debug("unable to recognize packet")
//			}
//		}
//	} else {
//		//c.Logger.Debug("Pattern %v not found.", hex.Dump(neededSig[:4]))
//	}
//}

// TODO move this to a traceroute helper fn
//if protocol == ethr.ICMP && (icmpMsg.Type == ipv4.ICMPTypeEchoReply || icmpMsg.Type == ipv6.ICMPTypeEchoReply) {
//	_ = icmpMsg.Body.(*icmp.Echo)
//	b, _ := icmpMsg.Body.Marshal(1)
//	if string(b[4:]) != string(neededIcmpBody) {
//		continue
//	}
//	isLast = true
//	return peerAddr, isLast, nil
//}
//	}
//}
