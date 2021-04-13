package tcp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestTraceRoute(test *session.Test, gap time.Duration, mtrMode bool, maxHops int) {
	hops, err := t.discoverHops(test, maxHops)
	if err != nil {
		test.Results <- session.TestResult{
			Success: false,
			Error:   fmt.Errorf("destination (%s) not responding to TCP connection", test.RemoteIP),
			Body:    payloads.TraceRoutePayload{Hops: hops},
		}
		return
	}
	if !mtrMode {
		test.Results <- session.TestResult{
			Success: true,
			Error:   nil,
			Body:    payloads.TraceRoutePayload{Hops: hops},
		}
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < len(hops); i++ {
		if hops[i].Addr.String() != "" {
			wg.Add(1)
			go t.probeHops(&wg, test, gap, i, hops)
		}
	}
	test.Results <- session.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.TraceRoutePayload{Hops: hops},
	}
	wg.Wait()
}

func (t Tests) probeHops(wg *sync.WaitGroup, test *session.Test, gap time.Duration, hop int, hops []payloads.NetworkHop) {
	defer wg.Done()
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
		if err == nil {
			hopData.Name, hopData.FullName = lookupHopName(hopData.Addr.String())
		}
		//if hopData.Addr != "" {
		//	if mtrMode {
		//		t.Logger.Info("%2d.|--%s", i+1, hopData.Addr+" ["+hopData.FullName+"]")
		//	} else {
		//		t.Logger.Info("%2d.|--%-70s %s", i+1, hopData.Addr+" ["+hopData.FullName+"]", ui.DurationToString(hopData.Last))
		//	}
		//} else {
		//	t.Logger.Info("%2d.|--%s", i+1, "???")
		//}
		hops[i] = hopData
		if isLast {
			return hops[:i+1], nil
		}
	}
	return nil, os.ErrNotExist
}

func lookupHopName(addr string) (string, string) {
	name := ""
	tname := ""
	if addr == "" {
		return tname, name
	}
	names, err := net.LookupAddr(addr)
	if err == nil && len(names) > 0 {
		name = names[0]
		sz := len(name)

		if sz > 0 && name[sz-1] == '.' {
			name = name[:sz-1]
		}
		tname = name
		if len(name) > 16 {
			tname = name[:16] + "..."
		}
	}
	return tname, name
}

func (t Tests) probeHop(test *session.Test, hop int, hopIP string, hopData *payloads.NetworkHop) (error, bool) {
	isLast := false
	icmpConn, err := t.NetTools.IcmpNewConn(test.RemoteIP.String())
	if err != nil {
		return fmt.Errorf("failed to create ICMP connection: %w", err), isLast
	}
	defer icmpConn.Close()
	localPortNum := uint16(8888)
	if t.NetTools.LocalPort != 0 {
		localPortNum = t.NetTools.LocalPort
	}
	localPortNum += uint16(hop)
	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:], localPortNum)
	remotePortNum, err := strconv.ParseUint(test.RemotePort, 10, 16)
	binary.BigEndian.PutUint16(b[2:], uint16(remotePortNum))
	peerAddrChan := make(chan net.Addr)
	endTimeChan := make(chan time.Time)
	go func() {
		var peerAddr net.Addr
		// TODO have max messages?
		for {
			icmpMsg, peer, _ := t.NetTools.ReceiveICMPFromPeer(icmpConn, time.Second*2, hopIP)
			if icmpMsg.Type == ipv4.ICMPTypeTimeExceeded || icmpMsg.Type == ipv6.ICMPTypeTimeExceeded {
				body := icmpMsg.Body.(*icmp.TimeExceeded).Data
				index := bytes.Index(body, b[:4])
				if index > 0 {
					peerAddr = peer
					break
				}
			}
		}

		endTimeChan <- time.Now()
		peerAddrChan <- peerAddr
	}()

	startTime := time.Now()
	var endTime time.Time
	var peerAddr net.Addr

	// For TCP Traceroute an ICMP error message will be sent for everything except the last connection which
	// should establish correctly. The go routine above handles parsing the ICMP error into info used below.
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), localPortNum, hop, 0)
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
