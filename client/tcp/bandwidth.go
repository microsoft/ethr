package tcp

import (
	"net"
	"sort"
	"strconv"

	"weavelab.xyz/ethr/session/payloads"

	"weavelab.xyz/ethr/stats"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (t Tests) TestBandwidth(test *session.Test) {
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP, t.NetTools.LocalPort+uint16(th), 0, 0) // referenced gTTL and gTOS which were never modified
		if err != nil {
			continue
		}
		err = test.Session.HandshakeWithServer(test, conn)
		if err != nil {
			_ = conn.Close()
			continue
		}
		go t.handleBandwidthConn(test, conn, strconv.Itoa(int(th)))
	}
}

func (t Tests) handleBandwidthConn(test *session.Test, conn net.Conn, id string) {
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
				Body: payloads.RawBandwidthPayload{
					ConnectionID:     id,
					Bandwidth:        uint64(n),
					PacketsPerSecond: 1,
				},
			})

			if !test.ClientParam.Reverse {
				sentBytes += uint64(n)
				start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, bufferLen)
			}
		}
	}
}

func BandwidthAggregator(nanos uint64, intermediateResults []session.TestResult) session.TestResult {
	totalBandwidth := uint64(0)
	totalPackets := uint64(0)
	connectionAggregates := make(map[string]*payloads.RawBandwidthPayload)

	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.RawBandwidthPayload); ok && r.Success {
			totalBandwidth += body.Bandwidth
			totalPackets += body.PacketsPerSecond

			if connection, ok := connectionAggregates[body.ConnectionID]; ok {
				connection.Bandwidth += body.Bandwidth
				connection.PacketsPerSecond += body.PacketsPerSecond
			} else {
				connectionAggregates[body.ConnectionID] = &body
			}
		}
	}

	connectionBandwidths := make([]payloads.RawBandwidthPayload, 0, len(connectionAggregates))
	for k, v := range connectionAggregates {
		connectionBandwidths = append(connectionBandwidths, payloads.RawBandwidthPayload{
			ConnectionID:     k,
			Bandwidth:        1e9 * v.Bandwidth / nanos,
			PacketsPerSecond: 1e9 * v.PacketsPerSecond / nanos,
		})
	}

	// even with 100s of threads this should be relatively fast
	sort.SliceStable(connectionBandwidths, func(i, j int) bool {
		return connectionBandwidths[i].ConnectionID < connectionBandwidths[j].ConnectionID
	})

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.BandwidthPayload{
			TotalBandwidth:        1e9 * totalBandwidth / nanos,
			TotalPacketsPerSecond: 1e9 * totalPackets / nanos,
			ConnectionBandwidths:  connectionBandwidths,
		},
	}
}
