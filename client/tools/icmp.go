package tools

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"golang.org/x/net/icmp"
	"weavelab.xyz/ethr/ethr"
)

func (t Tools) ReceiveICMPFromPeer(pc net.PacketConn, timeout time.Duration, neededPeer string) (*icmp.Message, net.Addr, error) {
	err := pc.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set deadline: %w", err)
	}
	for {
		b := make([]byte, 1500)
		n, peer, err := pc.ReadFrom(b)
		if err != nil {
			// In case of non-ICMP TraceRoute, it is expected that no packet is received
			// in some case, e.g. when packet reach final hop and TCP connection establishes.
			return nil, peer, fmt.Errorf("failed to receive ICMP packet: %w", err)
		}
		if n == 0 {
			continue
		}

		if neededPeer != "" && peer.String() != neededPeer {
			continue
		}
		icmpMsg, err := icmp.ParseMessage(ethr.ICMPProtocolNumber(t.IPVersion), b[:n])
		if err != nil {
			t.Logger.Debug("Failed to parse ICMP message: %w", err)
			continue
		}

		return icmpMsg, peer, nil
	}
}

func (t Tools) SendICMP(pc net.PacketConn, dest net.Addr, ttl int, timeout time.Duration, msg *icmp.Message) error {
	//start := time.Now()
	err := t.SetICMPTTL(pc, ttl)
	if err != nil {
		return fmt.Errorf("failed to set icmp ttl: %w", err)
	}
	err = t.setICMPToS(pc, 0)
	if err != nil {
		return fmt.Errorf("failed to set icmp tos: %w", err)
	}

	err = pc.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return fmt.Errorf("failed to set deadline: %w", err)
	}

	wb, err := msg.Marshal(nil)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	if _, err := pc.WriteTo(wb, dest); err != nil {
		return fmt.Errorf("failed to send ICMP data: %w", err)
	}
	return nil
}

func (t Tools) SetICMPTTL(pc net.PacketConn, ttl int) error {
	if t.IPVersion == ethr.IPv4 {
		cIPv4 := ipv4.NewPacketConn(pc)
		return cIPv4.SetTTL(ttl)
	} else if t.IPVersion == ethr.IPv6 {
		cIPv6 := ipv6.NewPacketConn(pc)
		return cIPv6.SetHopLimit(ttl)
	}
	return os.ErrInvalid
}

func (t Tools) setICMPToS(pc net.PacketConn, tos int) error {
	if tos == 0 {
		return nil
	}
	if t.IPVersion == ethr.IPv4 {
		cIPv4 := ipv4.NewPacketConn(pc)
		return cIPv4.SetTOS(tos)
	} else if t.IPVersion == ethr.IPv6 {
		cIPv6 := ipv6.NewPacketConn(pc)
		return cIPv6.SetTrafficClass(tos)
	}
	return os.ErrInvalid
}

// UnwrapICMPMessage parses out as much of the original icmp packet as possible
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
