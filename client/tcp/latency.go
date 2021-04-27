package tcp

import (
	"fmt"
	"io"
	"time"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestLatency(test *session.Test, g time.Duration) {
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP, t.NetTools.LocalPort, 0, 0)
	if err != nil {
		test.Results <- session.TestResult{
			Success: false,
			Error:   fmt.Errorf("error dialing the latency connection: %w", err),
			Body:    nil,
		}
		return
	}
	defer conn.Close()
	err = test.Session.HandshakeWithServer(test, conn)
	if err != nil {
		test.Results <- session.TestResult{
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
			test.AddIntermediateResult(session.TestResult{
				Success: true,
				Error:   nil,
				Body: payloads.RawLatencies{
					Latencies: latencyNumbers,
				},
			})
			return
		default:
			t0 := time.Now()
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				n, err := conn.Write(buff)
				if err != nil || n < blen {
					test.AddDirectResult(session.TestResult{
						Success: false,
						Error:   fmt.Errorf("error sending/receiving data on connection: %w", err),
						Body:    nil,
					})
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					test.AddDirectResult(session.TestResult{
						Success: false,
						Error:   fmt.Errorf("error sending/receiving data on connection: %w", err),
						Body:    nil,
					})
					break ExitSelect
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}

			_, _ = conn.Write(buff)
			test.AddIntermediateResult(session.TestResult{
				Success: true,
				Error:   nil,
				Body: payloads.RawLatencies{
					Latencies: latencyNumbers,
				},
			})
			t1 := time.Since(t0)
			if t1 < g {
				time.Sleep(g - t1)
			}
		}
	}
}

func LatencyAggregator(nanos uint64, intermediateResults []session.TestResult) session.TestResult {
	latencies := make([]time.Duration, 0)

	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.RawLatencies); ok && r.Success {
			latencies = append(latencies, body.Latencies...)
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body:    payloads.NewLatencies(latencies),
	}
}
