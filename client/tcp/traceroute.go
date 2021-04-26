package tcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
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
		test.Terminate()
		return
	}
	if !mtrMode {
		test.AddDirectResult(session.TestResult{
			Success: true,
			Error:   nil,
			Body:    payloads.TraceRoutePayload{Hops: hops},
		})
		test.Terminate()
		return
	}
	for i := 0; i < len(hops); i++ {
		if hops[i].Addr != nil && hops[i].Addr.String() != "" {
			// FIXME probe hop uses probeHop which creates a new icmp connection.
			// This results in n connections and reply packets often make it to the
			// incorrect connection. Instead, create once connection and multiplex
			// packets out to callers who can decide if it's relevant to get better stats.
			go t.probeHops(test, gap, i, hops)
		}
	}
	resultsTicker := time.NewTicker(time.Second)
	for {
		select {
		case <-resultsTicker.C:
			test.AddDirectResult(session.TestResult{
				Success: true,
				Error:   nil,
				Body:    payloads.TraceRoutePayload{Hops: hops},
			})
		case <-test.Done:
			return
		}
	}

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
		hop := payloads.NetworkHop{
			HopNumber: i,
		}
		err, isLast := t.probeHop(test, i+1, "", &hop)
		if err != nil && errors.Is(err, syscall.EPERM) {
			return nil, err
		}
		if err == nil {
			name := t.NetTools.LookupHopName(hop.Addr.String())
			hop.Name, hop.FullName = name, name
		}
		hops[i] = hop
		test.AddIntermediateResult(session.TestResult{
			Success: false,
			Error:   nil,
			Body:    hop,
		})
		if isLast {
			return hops[:i+1], nil
		}
	}
	return nil, os.ErrNotExist
}

type basicHop struct {
	addr    net.Addr
	endTime time.Time
}

func (t Tests) probeHop(test *session.Test, hop int, hopIP string, hopData *payloads.NetworkHop) (error, bool) {
	isLast := false
	icmpConn, err := t.NetTools.IcmpNewConn(test.RemoteIP.String())
	if err != nil {
		return fmt.Errorf("failed to create ICMP connection: %w", err), isLast
	}
	defer icmpConn.Close()

	localPort := t.NetTools.LocalPort + uint16(hop)
	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:], localPort)
	binary.BigEndian.PutUint16(b[2:], test.RemotePort)
	icmpTTLChan := make(chan basicHop, 1)
	go func() {
		var peerAddr net.Addr
		for {
			icmpMsg, peer, err := t.NetTools.ReceiveICMPFromPeer(icmpConn, time.Second*2, hopIP)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					break
				}
				// Go, sadly, doesn't export this error yet
				// connection succeeded and closed on final hop
				if strings.Contains(err.Error(), "use of closed network") {
					break
				}
				t.Logger.Debug("failed to get icmp reply, retrying")
				continue
			}
			if icmpMsg.Type == ipv4.ICMPTypeTimeExceeded || icmpMsg.Type == ipv6.ICMPTypeTimeExceeded {
				body := icmpMsg.Body.(*icmp.TimeExceeded).Data
				index := bytes.Index(body, b[:4])
				if index > 0 {
					peerAddr = peer
					break
				}
			}
			if icmpMsg.Type == ipv4.ICMPTypeDestinationUnreachable || icmpMsg.Type == ipv6.ICMPTypeDestinationUnreachable {
				fmt.Println("derpt, moving on")
				break
			}
		}
		icmpTTLChan <- basicHop{addr: peerAddr, endTime: time.Now()}
	}()

	// For TCP Traceroute an ICMP error message will be sent for everything except the last connection which
	// should establish correctly. The go routine above handles parsing the ICMP error into info used below.
	startTime := time.Now()
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, nil, localPort, hop, 0)

	// assume next hop is last hop and overwrite from icmp ttl error if not
	nextHop := basicHop{
		endTime: time.Now(),
		addr:    &net.IPAddr{IP: test.RemoteIP},
	}
	if err != nil { // majority case
		nextHop = <-icmpTTLChan
	} else {
		_ = conn.Close()
		isLast = true
	}

	hopData.Sent++
	hopData.UpdateStats(nextHop.addr, nextHop.endTime.Sub(startTime))
	if nextHop.addr == nil || nextHop.addr.String() == "" || (hopIP != "" && nextHop.addr.String() != hopIP) {
		hopData.Lost++
		return fmt.Errorf("failed to complete connection or receive ICMP TTL Exceeded: %w", os.ErrNotExist), isLast
	}

	return nil, isLast
}
