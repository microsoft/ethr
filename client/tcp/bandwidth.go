package tcp

import (
	"net"
	"sync"
	"sync/atomic"

	"weavelab.xyz/ethr/client/payloads"

	"weavelab.xyz/ethr/stats"

	"weavelab.xyz/ethr/client"
	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
)

func (c Tests) TestBandwidth(test *session.Test, results chan client.TestResult) {
	var wg sync.WaitGroup
	bandwidthResults := payloads.BandwidthPayload{
		TotalBandwidth:       0,
		ConnectionBandwidths: make([]uint64, 0),
	}
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		conn, err := c.NetTools.Dial(ethr.TCP, test.DialAddr, c.NetTools.LocalIP.String(), c.NetTools.LocalPort+uint16(th), 0, 0) // referenced gTTL and gTOS which were never modified
		if err != nil {
			//c.Logger.Error("error dialing connection: %w", err)
			continue
		}
		err = c.NetTools.HandshakeWithServer(test, conn)
		if err != nil {
			//c.Logger.Error("failed in handshake with the server: %w", err)
			_ = conn.Close()
			continue
		}
		wg.Add(1)
		go c.handleBandwidthConn(test, conn, &wg, th, &bandwidthResults)
	}
	wg.Wait()
	// TODO figure out failure conditions
	results <- client.TestResult{
		Success: true,
		Error:   nil,
		Body:    bandwidthResults,
	}
}

func (c Tests) handleBandwidthConn(test *session.Test, conn net.Conn, wg *sync.WaitGroup, th uint32, result *payloads.BandwidthPayload) {
	defer wg.Done()
	defer conn.Close()
	ec := test.NewConn(conn)
	//rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
	//lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())

	//c.Logger.Info("[%3d] local %s port %s connected to %s port %s", ec.FD, lserver, lport, rserver, rport)
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
				//c.Logger.Error("error sending/receiving data on a connection for bandwidth test: %w", err)
				return
			}
			atomic.AddUint64(&ec.Bandwidth, uint64(n))
			atomic.AddUint64(&result.ConnectionBandwidths[th], uint64(n))
			atomic.AddUint64(&result.TotalBandwidth, uint64(n))
			if !test.ClientParam.Reverse {
				sentBytes += uint64(n)
				start, waitTime, sentBytes, bytesToSend = stats.EnforceThrottle(start, waitTime, totalBytesToSend, sentBytes, bufferLen)
			}
		}
	}
}
