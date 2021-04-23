package tcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (t Tests) TestTraceRoute(test *session.Test, gap time.Duration, mtrMode bool, maxHops int) {
	hops, err := t.discoverHops(test, maxHops)
	if err != nil {
		test.AddDirectResult(session.TestResult{
			Success: false,
			Error:   fmt.Errorf("destination (%s) not responding to TCP connection", test.RemoteIP),
			Body:    payloads.TraceRoutePayload{Hops: hops},
		})
		return
	}
	if !mtrMode {
		test.AddDirectResult(session.TestResult{
			Success: true,
			Error:   nil,
			Body:    payloads.TraceRoutePayload{Hops: hops},
		})
		return
	}
	for i := 0; i < len(hops); i++ {
		if hops[i].Addr.String() != "" {
			go t.probeHops(test, gap, i, hops)
		}
	}
	test.AddDirectResult(session.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.TraceRoutePayload{Hops: hops},
	})
}

func (t Tests) probeHops(test *session.Test, gap time.Duration, hop int, hops []payloads.NetworkHop) {
	seq := 0
	for {
		select {
		case <-test.Done:
			return
		default:
			t0 := time.Now()
			err, _ := t.probeHop(test, hop+1, hops[hop].Addr.String(), &hops[hop])
			if err == nil {
			}
			seq++
			t1 := time.Since(t0)
			if t1 < gap {
				time.Sleep(gap - t1)
			}
		}
	}
}

func (t Tests) discoverHops(test *session.Test, maxHops int) ([]payloads.NetworkHop, error) {
	hops := make([]payloads.NetworkHop, maxHops)
	for i := 0; i < maxHops; i++ {
		var hopData payloads.NetworkHop
		err, isLast := t.probeHop(test, i+1, "", &hopData)
		if err != nil && errors.Is(err, syscall.EPERM) {
			return nil, err
		}
		if err == nil {
			name := lookupHopName(hopData.Addr.String())
			hopData.Name, hopData.FullName = name, name
		}
		hops[i] = hopData
		if isLast {
			return hops[:i+1], nil
		}
	}
	return nil, os.ErrNotExist
}

func lookupHopName(addr string) string {
	if addr == "" {
		return ""
	}
	names, err := net.LookupAddr(addr)
	if err == nil && len(names) > 0 {
		name := names[0]
		sz := len(name)

		if sz > 0 && name[sz-1] == '.' {
			name = name[:sz-1]
		}
		return name
	}
	return ""
}

func (t Tests) probeHop(test *session.Test, hop int, hopIP string, hopData *payloads.NetworkHop) (error, bool) {
	isLast := false
	icmpConn, err := t.NetTools.IcmpNewConn(test.RemoteIP.String())
	if err != nil {
		return fmt.Errorf("failed to create ICMP connection: %w", err), isLast
	}
	defer icmpConn.Close()
	//localPortNum := uint16(8888)
	//if t.NetTools.LocalPort != 0 {
	//	localPortNum = t.NetTools.LocalPort
	//}
	localPort := t.NetTools.LocalPort + uint16(hop)
	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:], localPort)
	binary.BigEndian.PutUint16(b[2:], test.RemotePort)
	peerAddrChan := make(chan net.Addr)
	endTimeChan := make(chan time.Time)
	go func() {
		var peerAddr net.Addr
		for {
			icmpMsg, peer, err := t.NetTools.ReceiveICMPFromPeer(icmpConn, time.Second*2, hopIP)
			if err != nil {
				fmt.Println("derpenstocks", err.Error())
			}
			if icmpMsg.Type == ipv4.ICMPTypeTimeExceeded || icmpMsg.Type == ipv6.ICMPTypeTimeExceeded {
				body := icmpMsg.Body.(*icmp.TimeExceeded).Data
				index := bytes.Index(body, b[:4])
				if index > 0 {
					peerAddr = peer
					break
				}
			}
		}

		// TODO send one object so timeout is easier
		endTimeChan <- time.Now()
		peerAddrChan <- peerAddr
	}()

	startTime := time.Now()
	var endTime time.Time
	var peerAddr net.Addr

	// For TCP Traceroute an ICMP error message will be sent for everything except the last connection which
	// should establish correctly. The go routine above handles parsing the ICMP error into info used below.
	// TODO dial addr probably shouldn't have port
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP, localPort, hop, 0)
	hopData.Sent++
	if err != nil { // majority case
		endTime = <-endTimeChan
		peerAddr = <-peerAddrChan
	} else {
		_ = conn.Close()
		endTime = time.Now()
		isLast = true
		peerAddr = &net.IPAddr{
			IP:   test.RemoteIP,
			Zone: "",
		}
	}

	elapsed := endTime.Sub(startTime)
	if peerAddr.String() == "" || (hopIP != "" && peerAddr.String() != hopIP) {
		hopData.Lost++
		return fmt.Errorf("failed to complete connection or receive ICMP TTL Exceeded: %w", os.ErrNotExist), isLast
	}
	hopData.UpdateStats(peerAddr, elapsed)
	return nil, isLast
}
