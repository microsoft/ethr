// +build windows

package tools

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"

	"weavelab.xyz/ethr/ethr"
)

func (t Tools) setSockOptInt(fd uintptr, level, opt, val int) error {
	err := syscall.SetsockoptInt(syscall.Handle(fd), level, opt, val)
	if err != nil {
		return fmt.Errorf("failed to set socket option (%v) to value (%v): %w", opt, val, err)
	}
	return nil
}

func (t Tools) setTClass(fd uintptr, tos int) error {
	return nil
}

func (t Tools) IsAdmin() bool {
	d, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	_ = d.Close()
	return true
}

const (
	SIO_RCVALL             = syscall.IOC_IN | syscall.IOC_VENDOR | 1
	RCVALL_OFF             = 0
	RCVALL_ON              = 1
	RCVALL_SOCKETLEVELONLY = 2
	RCVALL_IPLEVEL         = 3
)

func (t Tools) IcmpNewConn(address string) (net.PacketConn, error) {
	// This is an attempt to work around the problem described here:
	// https://github.com/golang/go/issues/38427

	// First, get the correct local interface address, as SIO_RCVALL can't be set on a 0.0.0.0 listeners.
	dialedConn, err := net.Dial(ethr.ICMPVersion(t.IPVersion), address)
	if err != nil {
		return nil, err
	}
	localAddr := dialedConn.LocalAddr()
	_ = dialedConn.Close()

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
	conn, err := cfg.ListenPacket(context.Background(), ethr.ICMPVersion(t.IPVersion), localAddr.String())
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
