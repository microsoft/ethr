package tcp

import (
	"fmt"
	"net"
	"sync"
	"time"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestPing(test *session.Test, g time.Duration, warmupCount uint32) {
	threads := test.ClientParam.NumThreads
	var wg sync.WaitGroup
	for th := uint32(0); th < threads; th++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-test.Done:
					return
				default:
					t0 := time.Now()
					if warmupCount > 0 {
						warmupCount--
						_, _ = t.DoPing(test, "[warmup]")
					} else {
						latency, err := t.DoPing(test, "")
						test.AddIntermediateResult(session.TestResult{
							Success: err == nil,
							Error:   err,
							Body: payloads.RawPingPayload{
								Latency: latency,
								Lost:    err == nil,
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
	wg.Wait()
}

func (t Tests) DoPing(test *session.Test, prefix string) (time.Duration, error) {
	t0 := time.Now()
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort, 0, 0) // TODO force client port to 0?
	if err != nil {
		return 0, fmt.Errorf("%sconnection to %s timed out: %w", prefix, test.DialAddr, err)
	}
	timeTaken := time.Since(t0)
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		_ = tcpconn.SetLinger(0)
	}
	_ = conn.Close()
	return timeTaken, nil
}

func PingAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	lost := 0
	received := 0
	latencies := make([]time.Duration, 0, len(intermediateResults))
	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.RawPingPayload); ok && r.Success {
			latencies = append(latencies, body.Latency)
			if body.Lost {
				lost++
			} else {
				received++
			}
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.PingPayload{
			Latency:  payloads.NewLatencies(len(latencies), latencies),
			Sent:     uint32(len(intermediateResults)),
			Lost:     uint32(lost),
			Received: uint32(received),
		},
	}
}
