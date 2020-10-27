//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"fmt"
	"net"
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

var ctrlPort string
var tcpBandwidthPort, tcpCpsPort, tcpPpsPort, tcpLatencyPort string
var udpBandwidthPort, udpCpsPort, udpPpsPort, udpLatencyPort string
var httpBandwidthPort, httpCpsPort, httpPpsPort, httpLatencyPort string
var httpsBandwidthPort, httpsCpsPort, httpsPpsPort, httpsLatencyPort string

var ctrlBasePort = 8888
var tcpBasePort = 9999
var udpBasePort = 9999
var httpBasePort = 9899
var httpsBasePort = 9799

func generatePortNumbers(customPortString string) {
	portsStr := strings.ToUpper(customPortString)
	data := customPortRegex.FindAllStringSubmatch(portsStr, -1)
	for _, kv := range data {
		k := kv[1]
		v := kv[2]
		p := toInt(v)
		if p == 0 {
			continue
		}
		switch k {
		case "TCP":
			tcpBasePort = p
		case "UDP":
			udpBasePort = p
		case "HTTP":
			httpBasePort = p
		case "HTTPS":
			httpsBasePort = p
		case "CONTROL":
			ctrlBasePort = p
		default:
			ui.printErr("generatePortNumbers: ignoring unexpected key in custom ports: %s", k)
		}
	}
	ctrlPort = toString(ctrlBasePort)
	tcpBandwidthPort = toString(tcpBasePort)
	tcpCpsPort = toString(tcpBasePort - 1)
	tcpPpsPort = toString(tcpBasePort - 2)
	tcpLatencyPort = toString(tcpBasePort - 3)
	udpBandwidthPort = toString(udpBasePort)
	udpCpsPort = toString(udpBasePort - 1)
	udpPpsPort = toString(udpBasePort - 2)
	udpLatencyPort = toString(udpBasePort - 3)
	httpBandwidthPort = toString(httpBasePort)
	httpCpsPort = toString(httpBasePort - 1)
	httpPpsPort = toString(httpBasePort - 2)
	httpLatencyPort = toString(httpBasePort - 3)
	httpsBandwidthPort = toString(httpsBasePort)
	httpsCpsPort = toString(httpsBasePort - 1)
	httpsPpsPort = toString(httpsBasePort - 2)
	httpsLatencyPort = toString(httpsBasePort - 3)
}

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
