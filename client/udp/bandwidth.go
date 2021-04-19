package udp

import (
	"net"
	"sort"
	"strconv"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
	"weavelab.xyz/ethr/stats"
)

func (t Tests) TestBandwidth(test *session.Test) {
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		go func(th uint32) {
			conn, err := t.NetTools.Dial(ethr.UDP, test.DialAddr, t.NetTools.LocalIP, t.NetTools.LocalPort+uint16(th), 0, 0)
			if err != nil {
				return
			}
			go t.handleBandwidthConn(test, conn, strconv.Itoa(int(th)))
		}(th)
	}
}

func (t Tests) handleBandwidthConn(test *session.Test, conn net.Conn, id string) {
	defer conn.Close()

	buffer := make([]byte, test.ClientParam.BufferSize)
	totalBytesToSend := test.ClientParam.BwRate
	sentBytes := uint64(0)
	start, waitTime, bytesToSend := stats.BeginThrottle(totalBytesToSend, len(buffer))
	for {
		select {
		case <-test.Done:
			return
		default:
			n, err := conn.Write(buffer[:bytesToSend])
			if err != nil {
				continue
			}
			if n < bytesToSend {
				continue
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
				start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, len(buffer))
			}
		}
	}
}

func BandwidthAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	totalBandwidth := uint64(0)
	totalPackets := uint64(0)
	connectionAggregates := make(map[string]payloads.RawBandwidthPayload)

	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.RawBandwidthPayload); ok && r.Success {
			totalBandwidth += body.Bandwidth
			totalPackets += body.PacketsPerSecond

			if connection, ok := connectionAggregates[body.ConnectionID]; ok {
				connection.Bandwidth += body.Bandwidth
				connection.PacketsPerSecond += body.PacketsPerSecond
			} else {
				connectionAggregates[body.ConnectionID] = body
			}
		}
	}

	connectionBandwidths := make([]payloads.RawBandwidthPayload, 0, len(connectionAggregates))
	for _, v := range connectionAggregates {
		connectionBandwidths = append(connectionBandwidths, payloads.RawBandwidthPayload{
			Bandwidth:        v.Bandwidth / seconds,
			PacketsPerSecond: v.PacketsPerSecond / seconds,
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
			TotalBandwidth:        totalBandwidth / seconds,
			TotalPacketsPerSecond: totalPackets / seconds,
			ConnectionBandwidths:  connectionBandwidths,
		},
	}
}
