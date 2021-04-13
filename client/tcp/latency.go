package tcp

import (
	"fmt"
	"io"
	"time"

	"weavelab.xyz/ethr/client/payloads"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestLatency(test *session.Test, g time.Duration) {
	conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort, 0, 0)
	if err != nil {
		test.Results <- session.TestResult{
			Success: false,
			Error:   fmt.Errorf("error dialing the latency connection: %w", err),
			Body:    nil,
		}
		return
	}
	defer conn.Close()
	err = t.NetTools.HandshakeWithServer(test, conn)
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
			result := payloads.NewLatencies(test, len(latencyNumbers), latencyNumbers)
			test.Results <- session.TestResult{
				Success: true,
				Error:   nil,
				Body:    result,
			}
			close(test.Results)
			return
		default:
			t0 := time.Now()
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				n, err := conn.Write(buff)
				if err != nil || n < blen {
					test.Results <- session.TestResult{
						Success: false,
						Error:   fmt.Errorf("error sending/receiving data on connection: %w", err), // TODO make an error template so we can check for this to know whether to keep listening
						Body:    nil,
					}
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					test.Results <- session.TestResult{
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
			result := payloads.NewLatencies(test, len(latencyNumbers), latencyNumbers)
			test.Results <- session.TestResult{
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
