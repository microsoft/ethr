package tcp

import (
	"net"
	"sync"
	"sync/atomic"

	"weavelab.xyz/ethr/ethr"
	"weavelab.xyz/ethr/session"
	"weavelab.xyz/ethr/session/payloads"
)

func (t Tests) TestConnectionsPerSecond(test *session.Test) {
	var wg sync.WaitGroup
	totalConnections := payloads.ConnectionsPerSecondPayload{}
	for th := uint32(0); th < test.ClientParam.NumThreads; th++ {
		wg.Add(1)
		go func(th uint32) {
			defer wg.Done()
			for {
				select {
				case <-test.Done:
					return
				default:
					conn, err := t.NetTools.Dial(ethr.TCP, test.DialAddr, t.NetTools.LocalIP.String(), t.NetTools.LocalPort, 0, 0) // TODO need to force local port to 0?
					if err != nil {
						//t.Logger.Debug("unable to dial TCP connection to %s: %w", test.DialAddr, err)
						continue
					}
					atomic.AddUint64(&totalConnections.Connections, 1)
					tcpconn, ok := conn.(*net.TCPConn)
					if ok {
						_ = tcpconn.SetLinger(0)
					}
					_ = conn.Close()
				}
			}
		}(th)
	}
	test.Results <- session.TestResult{
		Success: false,
		Error:   nil,
		Body:    totalConnections,
	}
	wg.Wait()
}

func ConnectionsAggregator(seconds uint64, intermediateResults []session.TestResult) session.TestResult {
	connections := uint64(0)
	for _, r := range intermediateResults {
		// ignore failed results
		if body, ok := r.Body.(payloads.ConnectionsPerSecondPayload); ok && r.Success {
			connections += body.Connections
		}
	}

	return session.TestResult{
		Success: true,
		Error:   nil,
		Body: payloads.ConnectionsPerSecondPayload{
			Connections: connections / seconds,
		},
	}
}
