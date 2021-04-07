package tcp

import (
	"fmt"
	"io"
	"sort"
	"time"

	"weavelab.xyz/ethr/client/payloads"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (c Tests) TestLatency(test *session.Test, g time.Duration, results chan client.TestResult) {
	conn, err := c.NetTools.Dial(ethr.TCP, test.DialAddr, c.NetTools.LocalIP.String(), c.NetTools.LocalPort, 0, 0)
	if err != nil {
		results <- client.TestResult{
			Success: false,
			Error:   fmt.Errorf("error dialing the latency connection: %w", err),
			Body:    nil,
		}
		return
	}
	defer conn.Close()
	err = c.NetTools.HandshakeWithServer(test, conn)
	if err != nil {
		results <- client.TestResult{
			Success: false,
			Error:   fmt.Errorf("failed in handshake with the server: %w", err),
			Body:    nil,
		}
		return
	}
	buffSize := test.ClientParam.BufferSize
	buff := make([]byte, buffSize)
	for i := uint32(0); i < buffSize; i++ {
		buff[i] = byte(i)
	}
	blen := len(buff)
	rttCount := test.ClientParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	for {
	ExitSelect:
		select {
		case <-test.Done:
			result := c.calcLatency(test, len(latencyNumbers), latencyNumbers)
			results <- client.TestResult{
				Success: true,
				Error:   nil,
				Body:    result,
			}
			close(results)
			return
		default:
			t0 := time.Now()
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				n, err := conn.Write(buff)
				if err != nil || n < blen {
					results <- client.TestResult{
						Success: false,
						Error:   fmt.Errorf("error sending/receiving data on connection: %w", err), // TODO make an error template so we can check for this to know whether to keep listening
						Body:    nil,
					}
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					results <- client.TestResult{
						Success: false,
						Error:   fmt.Errorf("error sending/receiving data on connection: %w", err), // TODO make an error template so we can check for this to know whether to keep listening
						Body:    nil,
					}
					break ExitSelect
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}
			// TODO temp code, fix it better, this is to allow server to do
			// server side latency measurements as well.
			_, _ = conn.Write(buff)
			result := c.calcLatency(test, len(latencyNumbers), latencyNumbers)
			results <- client.TestResult{
				Success: true,
				Error:   nil,
				Body:    result,
			}
			t1 := time.Since(t0)
			if t1 < g {
				time.Sleep(g - t1)
			}
		}
	}
}

func (c Tests) calcLatency(test *session.Test, rttCount int, latencies []time.Duration) payloads.LatencyPayload {
	sum := int64(0)
	for _, d := range latencies {
		sum += d.Nanoseconds()
	}
	elapsed := time.Duration(sum / int64(rttCount))
	sort.SliceStable(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	//
	// Special handling for rttCount == 1. This prevents negative index
	// in the latencyNumber index. The other option is to use
	// roundUpToZero() but that is more expensive.
	//
	rttCountFixed := rttCount
	if rttCountFixed == 1 {
		rttCountFixed = 2
	}
	return payloads.LatencyPayload{
		RemoteIP: test.RemoteIP,
		Protocol: test.ID.Protocol,
		Raw:      latencies,
		Avg:      elapsed,
		Min:      latencies[0],
		Max:      latencies[rttCount-1],
		P50:      latencies[((rttCountFixed*50)/100)-1],
		P90:      latencies[((rttCountFixed*90)/100)-1],
		P95:      latencies[((rttCountFixed*95)/100)-1],
		P99:      latencies[((rttCountFixed*99)/100)-1],
		P999:     latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
		P9999:    latencies[uint64(((float64(rttCountFixed)*99.9)/100)-1)],
	}
}
