package tools

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"weavelab.xyz/ethr/ethr"
)

func (t Tools) Dial(p ethr.Protocol, dialAddr, localIP string, localPortNum uint16, ttl int, tos int) (net.Conn, error) {
	localAddr := fmt.Sprintf("%v:%v", localIP, localPortNum)
	var lAddr net.Addr
	var network string
	var err error
	if p == ethr.TCP {
		network = ethr.TCPVersion(t.IPVersion)
		lAddr, err = net.ResolveTCPAddr(network, localAddr)
	} else if p == ethr.UDP {
		network = ethr.UDPVersion(t.IPVersion)
		lAddr, err = net.ResolveUDPAddr(network, localAddr)
	} else {
		return nil, fmt.Errorf("only TCP or UDP are allowed in dial: %w", os.ErrInvalid)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to resolve address: %w", err)
	}

	dialer := &net.Dialer{
		Control: func(network, address string, rc syscall.RawConn) error {
			return rc.Control(func(fd uintptr) {
				_ = t.setTTL(fd, ttl, t.IPVersion)
				_ = t.setTOS(fd, tos, t.IPVersion)
			})
		},
	}
	dialer.LocalAddr = lAddr
	dialer.Timeout = time.Second
	conn, err := dialer.Dial(network, dialAddr)
	if err != nil {
		return nil, fmt.Errorf("error dialing remote: %w", err)
	}
	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		_ = tcpConn.SetLinger(0)
		return tcpConn, nil
	}
	udpConn, ok := conn.(*net.UDPConn)
	if ok {
		err = udpConn.SetWriteBuffer(4 * 1024 * 1024)
		if err != nil {
			return nil, fmt.Errorf("failed to set ReadBuffer on UDP socket: %w", err)
		}
		return udpConn, nil
	}

	return nil, fmt.Errorf("unknown connection type created")
}

func (t Tools) setTTL(fd uintptr, ttl int, ipVersion ethr.IPVersion) error {
	if ttl == 0 {
		return nil
	}
	if ipVersion == ethr.IPv4 {
		return t.setSockOptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
	}
	return t.setSockOptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)
}

func (t Tools) setTOS(fd uintptr, tos int, ipVersion ethr.IPVersion) error {
	if tos == 0 {
		return nil
	}
	if ipVersion == ethr.IPv4 {
		return t.setSockOptInt(fd, syscall.IPPROTO_IP, syscall.IP_TOS, tos)
	}
	return t.setTClass(fd, tos)
}
