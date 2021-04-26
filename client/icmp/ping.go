package icmp

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"weavelab.xyz/ethr/ui"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"weavelab.xyz/ethr/ethr"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (t Tests) TestPing(test *session.Test, g time.Duration, warmupCount uint32) {
	addr := &net.IPAddr{IP: test.RemoteIP}

	//threads := test.ClientParam.NumThreads
	threads := 1
	for th := 0; th < threads; th++ {
		go func() {
			for {
				select {
				case <-test.Done:
					return
				default:
					t0 := time.Now()
					if warmupCount > 0 {
						warmupCount--
						_, _ = t.DoPing(addr, "[warmup]")
					} else {
						latency, err := t.DoPing(addr, "")
						test.AddIntermediateResult(session.TestResult{
							Success: err == nil,
							Error:   err,
							Body: payloads.RawPingPayload{
								Latency: latency,
								Lost:    err != nil,
							},
						})
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

func (t Tests) DoPing(addr net.Addr, prefix string) (time.Duration, error) {
	timeout := time.Second
	latency, peer, err := t.icmpPing(addr, timeout, 254, 255)
	if err != nil {
		return timeout, err
	}

	if peer.String() != addr.String() {
		t.Logger.Info("[icmp] %sPing to %s: %s",
			prefix, addr, "Non-EchoReply Received.")
		return time.Second, os.ErrNotExist
	}
	t.Logger.Info("[icmp] %sPing to %s: %s",
		prefix, addr, ui.DurationToString(latency))

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
	start := time.Now()
	err = t.NetTools.SendICMP(c, dest, hop+1, timeout, &msg)
	if err != nil {
		return timeout, nil, err
	}
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
						return time.Since(start), peer, ErrTTLExceeded
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

func PingAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	lost := 0
	received := 0
	latencies := make([]time.Duration, 0, len(intermediateResults))
	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.RawPingPayload); ok && r.Success {
			if body.Lost {
				lost++
			} else {
				latencies = append(latencies, body.Latency)
				received++
			}
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.PingPayload{
			Latency:  payloads.NewLatencies(latencies),
			Sent:     uint32(len(intermediateResults)),
			Lost:     uint32(lost),
			Received: uint32(received),
		},
	}
}
