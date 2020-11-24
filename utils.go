//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
)

//
// Regular expression to parse input for custom ports.
//
var customPortRegex = regexp.MustCompile("(\\w+)=([0-9]+)")

//
// TODO: Use a better way to define ports. The core logic is:
// Find a base port, such as 9999, and the Bandwidth is: base - 0,
// Cps is base - 1, Pps is base - 2 and Latency is base - 3
//
const (
	hostAddr = ""
)

var gEthrPort = 8888
var gEthrPortStr = ""

const (
	// UNO represents 1 unit.
	UNO = 1

	// KILO represents k.
	KILO = 1000

	// MEGA represents m.
	MEGA = 1000 * 1000

	// GIGA represents g.
	GIGA = 1000 * 1000 * 1000

	// TERA represents t.
	TERA = 1000 * 1000 * 1000 * 1000
)

func numberToUnit(num uint64) string {
	unit := ""
	value := float64(num)

	switch {
	case num >= TERA:
		unit = "T"
		value = value / TERA
	case num >= GIGA:
		unit = "G"
		value = value / GIGA
	case num >= MEGA:
		unit = "M"
		value = value / MEGA
	case num >= KILO:
		unit = "K"
		value = value / KILO
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".00")
	return result + unit
}

func unitToNumber(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	i := strings.IndexFunc(s, unicode.IsLetter)

	if i == -1 {
		bytes, err := strconv.ParseFloat(s, 64)
		if err != nil || bytes <= 0 {
			return 0
		}
		return uint64(bytes)
	}

	bytesString, multiple := s[:i], s[i:]
	bytes, err := strconv.ParseFloat(bytesString, 64)
	if err != nil || bytes <= 0 {
		return 0
	}

	switch multiple {
	case "T", "TB", "TIB":
		return uint64(bytes * TERA)
	case "G", "GB", "GIB":
		return uint64(bytes * GIGA)
	case "M", "MB", "MIB":
		return uint64(bytes * MEGA)
	case "K", "KB", "KIB":
		return uint64(bytes * KILO)
	case "B":
		return uint64(bytes)
	default:
		return 0
	}
}

func bytesToRate(bytes uint64) string {
	bits := bytes * 8
	result := numberToUnit(bits)
	return result
}

func cpsToString(cps uint64) string {
	result := numberToUnit(cps)
	return result
}

func ppsToString(pps uint64) string {
	result := numberToUnit(pps)
	return result
}

func testToString(testType EthrTestType) string {
	switch testType {
	case Bandwidth:
		return "Bandwidth"
	case Cps:
		return "Connections/s"
	case Pps:
		return "Packets/s"
	case Latency:
		return "Latency"
	case Ping:
		return "Ping"
	case TraceRoute:
		return "TraceRoute"
	default:
		return "Invalid"
	}
}

func durationToString(d time.Duration) string {
	if d < 0 {
		return d.String()
	}
	ud := uint64(d)
	val := float64(ud)
	unit := ""
	if ud < uint64(60*time.Second) {
		switch {
		case ud < uint64(time.Microsecond):
			unit = "ns"
		case ud < uint64(time.Millisecond):
			val = val / 1000
			unit = "us"
		case ud < uint64(time.Second):
			val = val / (1000 * 1000)
			unit = "ms"
		default:
			val = val / (1000 * 1000 * 1000)
			unit = "s"
		}

		result := strconv.FormatFloat(val, 'f', 3, 64)
		return result + unit
	}

	return d.String()
}

func protoToString(proto EthrProtocol) string {
	switch proto {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	case HTTP:
		return "HTTP"
	case HTTPS:
		return "HTTPS"
	case ICMP:
		return "ICMP"
	}
	return ""
}

func tcp(ipVer ethrIPVer) string {
	switch ipVer {
	case ethrIPv4:
		return "tcp4"
	case ethrIPv6:
		return "tcp6"
	}
	return "tcp"
}

func udp(ipVer ethrIPVer) string {
	switch ipVer {
	case ethrIPv4:
		return "udp4"
	case ethrIPv6:
		return "udp6"
	}
	return "udp"
}

func Icmp(ipVer ethrIPVer) string {
	switch ipVer {
	case ethrIPv6:
		return "ip6:ipv6-icmp"
	default:
		return "ip4:icmp"
	}
}

func IcmpProto(ipVer ethrIPVer) int {
	if ipVer == ethrIPv6 {
		return Icmpv6
	}
	return Icmpv4
}

func ethrUnused(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

func splitString(longString string, maxLen int) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, maxLen; r < len(longString); l, r = r, r+maxLen {
		for !utf8.RuneStart(longString[r]) {
			r--
		}
		splits = append(splits, longString[l:r])
	}
	splits = append(splits, longString[l:])
	return splits
}

func max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func toString(n int) string {
	return fmt.Sprintf("%d", n)
}

func toInt(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		ui.printDbg("Error in string conversion: %v", err)
		return 0
	}
	return res
}

func truncateStringFromStart(str string, num int) string {
	s := str
	l := len(str)
	if l > num {
		if num > 3 {
			s = "..." + str[l-num+3:l]
		} else {
			s = str[l-num : l]
		}
	}
	return s
}

func truncateStringFromEnd(str string, num int) string {
	s := str
	l := len(str)
	if l > num {
		if num > 3 {
			s = str[0:num] + "..."
		} else {
			s = str[0:num]
		}
	}
	return s
}

func roundUpToZero(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

func getFd(conn net.Conn) uintptr {
	var fd uintptr
	var rc syscall.RawConn
	var err error
	switch ct := conn.(type) {
	case *net.TCPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	case *net.UDPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	default:
		return 0
	}
	fn := func(s uintptr) {
		fd = s
	}
	rc.Control(fn)
	return fd
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func SleepUntilNextWholeSecond() {
	t0 := time.Now()
	t1 := t0.Add(time.Second)
	res := t1.Round(time.Second)
	time.Sleep(time.Until(res))
}

func ethrDialForTraceRoute(network, address string, portNum uint16, ttl int) (conn net.Conn, err error) {
	port := fmt.Sprintf(":%v", portNum)
	la, err := net.ResolveTCPAddr(network, port)
	if err != nil {
		ui.printErr("Unable to resolve TCP address. Error: %v", err)
		return
	}
	dialer := &net.Dialer{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if ipVer == ethrIPv4 {
					setSockOptInt(fd, syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
				} else {
					setSockOptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, ttl)

				}
			})
		},
	}
	dialer.LocalAddr = la
	dialer.Timeout = time.Second
	conn, err = dialer.Dial(network, address)
	if err != nil {
		ui.printDbg("ethrDialForTraceRoute Error: %v", err)
	} else {
		tcpconn, ok := conn.(*net.TCPConn)
		if ok {
			tcpconn.SetLinger(0)
		}
		conn.Close()
	}
	return
}

func ethrLookupIP(server string) (net.IPAddr, string, error) {
	var ipAddr net.IPAddr
	var ipStr string

	ip := net.ParseIP(server)
	if ip != nil {
		ipAddr.IP = ip
		ipStr = server
		return ipAddr, ipStr, nil
	}

	ips, err := net.LookupIP(server)
	if err != nil {
		ui.printErr("Failed to looup IP address for the server: %v. Error: %v", server, err)
		return ipAddr, ipStr, err
	}
	for _, ip := range ips {
		if ipVer == ethrIPAny || (ipVer == ethrIPv4 && ip.To4() != nil) || (ipVer == ethrIPv6 && ip.To16() != nil) {
			ipAddr.IP = ip
			ipStr = ip.String()
			ui.printDbg("Resolved server: %v to IP address: %v\n", server, ip)
			return ipAddr, ipStr, nil
		}
	}
	ui.printErr("Unable to resolve the given server: %v to an IP address.", server)
	return ipAddr, ipStr, os.ErrNotExist
}
