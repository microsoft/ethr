package tcp

import (
	"net"
	"sync"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/stats"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestBandwidth(test *session.Test) {
	var wg sync.WaitGroup
	bandwidthResults := payloads.BandwidthPayload{
		TotalBandwidth:       0,
		ConnectionBandwidths: make([]uint64, 0),
	}
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort+uint16(th), 0, 0) // referenced gTTL and gTOS which were never modified
		if err != nil {
			//t.Logger.Error("error dialing connection: %w", err)
			continue
		}
		err = t.NetTools.Session.HandshakeWithServer(test, conn)
		if err != nil {
			//t.Logger.Error("failed in handshake with the server: %w", err)
			_ = conn.Close()
			continue
		}
		wg.Add(1)
		go t.handleBandwidthConn(test, conn, &wg, th, &bandwidthResults)
	}
	// TODO figure out failure conditions
	test.Results <- session.TestResult{
		Success: true,
		Error:   nil,
		Body:    &bandwidthResults,
	}
	wg.Wait()
	close(test.Results)
}

func (t Tests) handleBandwidthConn(test *session.Test, conn net.Conn, wg *sync.WaitGroup, th uint32, result *payloads.BandwidthPayload) {
	defer wg.Done()
	defer conn.Close()

	size := test.ClientParam.BufferSize
	buff := make([]byte, size)
	for i := uint32(0); i < size; i++ {
		buff[i] = byte(i)
	}
	bufferLen := len(buff)
	totalBytesToSend := test.ClientParam.BwRate
	sentBytes := uint64(0)
	start, waitTime, bytesToSend := stats.BeginThrottle(totalBytesToSend, bufferLen)
	for {
		select {
		case <-test.Done:
			return
		default:
			n := 0
			var err error = nil
			if test.ClientParam.Reverse {
				n, err = conn.Read(buff)
			} else {
				n, err = conn.Write(buff[:bytesToSend])
			}
			if err != nil {
				//t.Logger.Error("error sending/receiving data on a connection for bandwidth test: %w", err)
				return
			}

			test.AddIntermediateResult(session.TestResult{
				Success: true,
				Error:   nil,
				Body: payloads.BandwidthPayload{
					TotalBandwidth:       uint64(n),
					ConnectionBandwidths: nil,
					PacketsPerSecond:     1,
				},
			})

			if !test.ClientParam.Reverse {
				sentBytes += uint64(n)
				start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, bufferLen)
			}
		}
	}
}

func BandwidthAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	totalBandwidth := uint64(0)
	totalPackets := uint64(0)
	// TODO calculate bandwidth of individual threads/conns

	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.BandwidthPayload); ok && r.Success {
			totalBandwidth += body.TotalBandwidth
			totalPackets += body.PacketsPerSecond
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.BandwidthPayload{
			TotalBandwidth:       totalBandwidth / seconds,
			ConnectionBandwidths: nil,
			PacketsPerSecond:     totalPackets / seconds,
		},
	}
}
