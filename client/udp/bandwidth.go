package udp

import (
	"net"
	"sync"
	"sync/atomic"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/client/payloads"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/stats"
)

func (t Tests) TestBandwidth(test *session.Test, results chan client.TestResult) {
	var wg sync.WaitGroup
	bandwidthResults := payloads.BandwidthPayload{
		TotalBandwidth:       0,
		ConnectionBandwidths: make([]uint64, 0),
	}
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		go func(th uint32) {
			conn, err := t.NetTools.Dial(ethr.UDP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort+uint16(th), 0, 0)
			if err != nil {
				return
			}
			wg.Add(1)
			go t.handleBandwidthConn(test, conn, &wg, th, &bandwidthResults)

		}(th)
	}

	// TODO figure out failure conditions
	results <- client.TestResult{
		Success: true,
		Error:   nil,
		Body:    &bandwidthResults,
	}
	wg.Wait()
}

func (t Tests) handleBandwidthConn(test *session.Test, conn net.Conn, wg *sync.WaitGroup, th uint32, result *payloads.BandwidthPayload) {
	defer wg.Done()
	defer conn.Close()
	ec := test.NewConn(conn)

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

			atomic.AddUint64(&ec.Bandwidth, uint64(n))
			atomic.AddUint64(&ec.PacketsPerSecond, 1)
			atomic.AddUint64(&result.ConnectionBandwidths[th], uint64(n))
			atomic.AddUint64(&result.TotalBandwidth, uint64(n))
			atomic.AddUint64(&result.PacketsPerSecond, 1)

			if !test.ClientParam.Reverse {
				sentBytes += uint64(n)
				start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, len(buffer))
			}
		}
	}
}
