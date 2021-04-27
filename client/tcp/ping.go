package tcp

import (
	"fmt"
	"net"
	"time"

	"weavelab.xyz/ethr/ui"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestPing(test *session.Test, g time.Duration, warmupCount uint32) {
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
						_, _ = t.DoPing(test, "[warmup]")
					} else {
						latency, err := t.DoPing(test, "")
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

func (t Tests) DoPing(test *session.Test, prefix string) (time.Duration, error) {
	t0 := time.Now()
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP, 0, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("%sconnection to %s timed out: %w", prefix, test.DialAddr, err)
	}
	timeTaken := time.Since(t0)
	t.Logger.Info("[tcp] %sConnection from %s to %s: %s",
		prefix, conn.LocalAddr(), conn.RemoteAddr(), ui.DurationToString(timeTaken))
	tcpconn, ok := conn.(*net.TCPConn)
	if ok {
		_ = tcpconn.SetLinger(0)
	}
	_ = conn.Close()
	return timeTaken, nil
}

func PingAggregator(microseconds uint64, intermediateResults []session.TestResult) session.TestResult {
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
