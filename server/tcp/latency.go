package tcp

import (
	"fmt"
	"io"
	"net"
	"time"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (h Handler) TestLatency(test *session.Test, clientParam ethr.ClientParams, conn net.Conn) error {
	bytes := make([]byte, clientParam.BufferSize)
	rttCount := clientParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	for {
		_, err := io.ReadFull(conn, bytes)
		if err != nil {
			return fmt.Errorf("error receiving data for latency tests: %w", err)
		}
		for i := uint32(0); i < rttCount; i++ {
			s1 := time.Now()
			_, err = conn.Write(bytes)
			if err != nil {
				return fmt.Errorf("error sending data for latency test: %w", err)

			}
			_, err = io.ReadFull(conn, bytes)
			if err != nil {
				return fmt.Errorf("error receiving data for latency test: %w", err)

			}
			e2 := time.Since(s1)
			latencyNumbers[i] = e2
		}

		test.AddIntermediateResult(session.TestResult{
			Success: true,
			Error:   nil,
			Body:    payloads.RawLatencies{Latencies: latencyNumbers},
		})
	}
}
