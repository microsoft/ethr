package icmp

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestPing(test *session.Test, g time.Duration, warmupCount uint32) {
	addr, _, err := t.NetTools.LookupIP(test.DialAddr)
	if err != nil {
		test.Results <- session.TestResult{
			Success: false,
			Error:   err,
			Body:    nil,
		}
		return
	}

	// TODO emit raw stats (e.g. sent/lost/received/lat per ping) and aggregate and emit results in a new go routine
	// TODO: Override NumThreads for now, fix it later to support parallel threads
	//threads := test.ClientParam.NumThreads
	threads := uint32(1)
	for th := uint32(0); th < threads; th++ {
		go func() {
			var sent, received, lost uint32
			latencyNumbers := make([]time.Duration, 0)
			for {
				select {
				case <-test.Done:
					result := payloads.NewLatencies(test, int(received), latencyNumbers)
					test.Results <- session.TestResult{
						Success: true,
						Error:   nil,
						Body: payloads.PingPayload{
							Latency:  result,
							Sent:     sent,
							Lost:     lost,
							Received: received,
						},
					}
					return
				default:
					t0 := time.Now()
					if warmupCount > 0 {
						warmupCount--
						_, _ = t.DoPing(&addr)
					} else {
						sent++
						latency, err := t.DoPing(&addr)
						if err == nil {
							received++
							latencyNumbers = append(latencyNumbers, latency)
						} else {
							lost++
						}
					}
					// TODO add failure case. lost > received? all packets lost?
					if received >= 1000 {
						result := payloads.NewLatencies(test, int(received), latencyNumbers)
						test.Results <- session.TestResult{
							Success: true,
							Error:   nil,
							Body: payloads.PingPayload{
								Latency:  result,
								Sent:     sent,
								Lost:     lost,
								Received: received,
							},
						}
						latencyNumbers = make([]time.Duration, 0)
						sent, received, lost = 0, 0, 0
					}
					t1 := time.Since(t0)
					if t1 < g {
						time.Sleep(g - t1)
					}
				}
			}
		}()
	}
}

func (t Tests) DoPing(addr net.Addr) (time.Duration, error) {
	timeout := time.Second
	latency, _, err := t.icmpPing(addr, timeout, 254, 255)
	if err != nil {
		return timeout, err
	}

	return latency, nil
}

func (t Tests) icmpPing(dest net.Addr, timeout time.Duration, hop int, seq int) (time.Duration, net.Addr, error) {
	echoMsg := fmt.Sprintf("Hello: Ethr - %v", hop)

	c, err := t.NetTools.IcmpNewConn(dest.String())
	if err != nil {
		return timeout, nil, fmt.Errorf("failed to create icmp connection: %w", err)
	}
	defer c.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   9999,
			Seq:  hop<<8 | seq,
			Data: []byte(echoMsg),
		},
	}
	if t.NetTools.IPVersion == ethr.IPv6 {
		msg.Type = ipv6.ICMPTypeEchoRequest
	}
	// TODO when to start timer?
	err = t.NetTools.SendICMP(c, dest, hop+1, timeout, &msg)
	if err != nil {
		return timeout, nil, err
	}
	start := time.Now()
	reqBytes, _ := msg.Marshal(nil)
	for {
		neededSeq := hop<<8 | seq
		var resp *icmp.Message
		resp, peer, err := t.NetTools.ReceiveICMPFromPeer(c, timeout, "")
		if err != nil {
			return timeout, peer, fmt.Errorf("failed to receive ICMP reply packet: %w", err)
		}

		// Routing loop
		if resp.Type == ipv4.ICMPTypeTimeExceeded || resp.Type == ipv6.ICMPTypeTimeExceeded {
			body := resp.Body.(*icmp.TimeExceeded).Data
			index := bytes.Index(body, reqBytes[4:8])
			if index > 0 {
				if index < 4 {
					continue
				}
				innerIcmpMsg, _ := icmp.ParseMessage(ethr.ICMPProtocolNumber(t.NetTools.IPVersion), body[index-4:])
				switch innerIcmpMsg.Body.(type) {
				case *icmp.Echo:
					seq := innerIcmpMsg.Body.(*icmp.Echo).Seq
					if seq == neededSeq {
						return timeout, peer, ErrTTLExceeded
					}
				default:
					// Ignore as this is not the right ICMP packet.
					continue
				}
			}
		}
		if resp.Type == ipv4.ICMPTypeEchoReply || resp.Type == ipv6.ICMPTypeEchoReply {
			b, _ := resp.Body.Marshal(1)
			if string(b[4:]) != echoMsg {
				continue
			}

			return time.Since(start), peer, nil
		}
	}
}
