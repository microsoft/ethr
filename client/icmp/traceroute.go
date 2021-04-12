package icmp

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestTraceRoute(test *session.Test, gap time.Duration, mtrMode bool, maxHops int, results chan client.TestResult) {
	dstIPAddr, _, err := t.NetTools.LookupIP(test.RemoteIP.String())
	if err != nil {
		results <- client.TestResult{
			Success: false,
			Error:   err,
			Body:    nil,
		}
		return
	}
	hops, err := t.discoverHops(dstIPAddr, maxHops)
	if err != nil {
		results <- client.TestResult{
			Success: false,
			Error:   fmt.Errorf("destination is not responding to ICMP echo: %w", err),
			Body:    nil,
		}
		return
	}
	if !mtrMode {
		if !mtrMode {
			results <- client.TestResult{
				Success: true,
				Error:   nil,
				Body:    payloads.TraceRoutePayload{Hops: hops},
			}
			return
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < len(hops); i++ {
		if hops[i].Addr.String() != "" {
			wg.Add(1)
			go t.probeHop(&wg, test.Done, gap, &hops[i], i)
		}
	}
	results <- client.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.TraceRoutePayload{Hops: hops},
	}
	wg.Wait()

}

func (t Tests) discoverHops(dstIPAddr net.IPAddr, maxHops int) ([]payloads.NetworkHop, error) {
	hops := make([]payloads.NetworkHop, maxHops)
	for i := 0; i < maxHops; i++ {
		var hopData payloads.NetworkHop
		_, peer, err := t.icmpPing(&dstIPAddr, time.Second, i, 1)
		if err != nil && !errors.Is(err, ErrTTLExceeded) {
			hopData.Lost++
			continue
		}

		hopData.Addr = peer
		hopData.Name, hopData.FullName = lookupHopName(hopData.Addr.String())
		hops[i] = hopData
		// we got an echo from the desired addr and didn't exceed ttl so we are done
		if err == nil {
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

func (t Tests) probeHop(wg *sync.WaitGroup, done chan struct{}, gap time.Duration, hopData *payloads.NetworkHop, ttl int) {
	defer wg.Done()
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
