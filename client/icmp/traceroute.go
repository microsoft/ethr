package icmp

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"time"

	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (t Tests) TestTraceRoute(test *session.Test, gap time.Duration, mtrMode bool, maxHops int) {
	hops, err := t.discoverHops(test, maxHops)
	if err != nil {
		test.Results <- session.TestResult{
			Success: false,
			Error:   fmt.Errorf("destination is not responding to ICMP echo: %w", err),
			Body:    nil,
		}
		test.Terminate()
		return
	}
	if !mtrMode {
		if !mtrMode {
			test.Results <- session.TestResult{
				Success: true,
				Error:   nil,
				Body:    payloads.TraceRoutePayload{Hops: hops},
			}
			test.Terminate()
			return
		}
	}

	for i := 0; i < len(hops); i++ {
		if hops[i].Addr.String() != "" {
			// FIXME probe hop uses icmpPing which creates a new icmp connection.
			// This results in n connections and reply packets often make it to the
			// incorrect connection. Instead, create once connection and multiplex
			// packets out to callers who can decide if it's relevant to get better stats.
			go t.probeHop(test.Done, gap, &hops[i], i)
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

func (t Tests) discoverHops(test *session.Test, maxHops int) ([]payloads.NetworkHop, error) {
	hops := make([]payloads.NetworkHop, maxHops)
	for i := 0; i < maxHops; i++ {
		hop := payloads.NetworkHop{
			HopNumber: i,
			Sent:      1,
		}
		latency, peer, err := t.icmpPing(&net.IPAddr{IP: test.RemoteIP}, time.Second, i, 1)
		if err != nil && errors.Is(err, syscall.EPERM) {
			return nil, err
		} else if err != nil && (!errors.Is(err, ErrTTLExceeded) || peer == nil) {
			hop.Lost++
			continue
		}

		// expect ErrTTLExceeded for most hops
		hop.UpdateStats(peer, latency)
		name := t.NetTools.LookupHopName(hop.Addr.String())
		hop.Name, hop.FullName = name, name

		hops[i] = hop
		test.AddIntermediateResult(session.TestResult{
			Success: false,
			Error:   nil,
			Body:    hop,
		})

		// we got an echo from the desired addr and didn't exceed ttl so we are done
		if err == nil && peer.String() == test.RemoteIP.String() {
			return hops[:i+1], nil
		}

	}
	return nil, os.ErrNotExist
}

func (t Tests) probeHop(done chan struct{}, gap time.Duration, hopData *payloads.NetworkHop, ttl int) {
	seq := 0
	for {
		select {
		case <-done:
			return
		default:
			t0 := time.Now()
			latency, peer, err := t.icmpPing(hopData.Addr, time.Second, ttl, seq)
			hopData.Sent++
			if err != nil && !errors.Is(err, ErrTTLExceeded) {
				hopData.Lost++
			} else {
				hopData.UpdateStats(peer, latency)
			}
			seq++
			t1 := time.Since(t0)
			if t1 < gap {
				time.Sleep(gap - t1)
			}
		}
	}
}
